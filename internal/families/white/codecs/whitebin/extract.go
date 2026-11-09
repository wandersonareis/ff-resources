// Package whitebin implementa operações sobre o "white bin" (o container
// grande referenciado pelo filelist). Porte de WhiteBinTools.Unpack.
//
// É compatível com PC (win32), PS3 e Xbox 360: em todas as três o filelist
// aponta para um deslocamento (setores de 2048 bytes) e um tamanho no white
// bin, e os dados são copiados ou descomprimidos com zlib do mesmo jeito. A
// extração foi validada byte a byte contra o WhiteBinTools nos três formatos.
package whitebin

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ffresources/internal/families/white/codecs/filelist"
)

// State indica como o arquivo foi extraído.
type State string

const (
	// Decompressed indica que os dados estavam comprimidos (zlib) e foram
	// descomprimidos.
	Decompressed State = "Decompressed"
	// Copied indica que os dados foram copiados sem compressão.
	Copied State = "Copied"
)

// Result descreve uma extração.
type Result struct {
	VirtualPath string
	OutFile     string
	Position    int64
	UncmpSize   uint32
	CmpSize     uint32
	Compressed  bool
	State       State
	// Overwrote é true quando uma ocorrência anterior do mesmo caminho já
	// havia sido gravada (repetição no filelist).
	Overwrote bool
}

// ExtractFile extrai do white bin o(s) arquivo(s) cujo caminho virtual é
// virtualPath. Quando o caminho aparece repetido no filelist, todas as
// ocorrências são extraídas na ordem da coleção (a última vence), como no
// WhiteBinTools.
//
// O white bin é lido por posição (ReadAt), sem carregar o arquivo inteiro.
func ExtractFile(whiteBinPath, outDir, virtualPath string, coll *filelist.Collection) ([]Result, error) {
	vpath := normalizePath(virtualPath)
	refs := coll.FindByPath(vpath)
	if len(refs) == 0 {
		return nil, fmt.Errorf("arquivo não encontrado no filelist: %s", vpath)
	}

	f, err := os.Open(whiteBinPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	binSize := info.Size()

	written := make(map[string]bool)
	results := make([]Result, 0, len(refs))

	for _, ref := range refs {
		e := &coll.Chunks[ref.Chunk].Entries[ref.Entry]
		loc, err := filelist.ParseFileInfo(e.FileInfo)
		if err != nil {
			return results, fmt.Errorf("fileInfo inválido (%q): %w", e.FileInfo, err)
		}

		pos := int64(loc.PosSectors) * 2048
		readLen := int64(loc.CmpSize)
		if !loc.IsCompressed() {
			readLen = int64(loc.UncmpSize)
		}
		if pos < 0 || readLen < 0 || pos+readLen > binSize {
			return results, fmt.Errorf("região fora do white bin: %s (pos=%d len=%d bin=%d)",
				vpath, pos, readLen, binSize)
		}

		outFile := filepath.Join(outDir, filepath.FromSlash(vpath))
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			return results, err
		}

		overwrote := written[outFile]
		state, err := writeOne(f, pos, loc, outFile)
		if err != nil {
			return results, fmt.Errorf("extraindo %s: %w", vpath, err)
		}
		written[outFile] = true

		results = append(results, Result{
			VirtualPath: vpath,
			OutFile:     outFile,
			Position:    pos,
			UncmpSize:   loc.UncmpSize,
			CmpSize:     loc.CmpSize,
			Compressed:  loc.IsCompressed(),
			State:       state,
			Overwrote:   overwrote,
		})
	}

	return results, nil
}

// ExtractBytes extrai do white bin o(s) arquivo(s) cujo caminho virtual é
// virtualPath e devolve os BYTES — sem gravar nada em disco.
//
// A semântica é a de ExtractFile: quando o caminho aparece repetido no
// filelist, todas as ocorrências são lidas na ordem da coleção e a ÚLTIMA
// vence (é o mesmo arquivo que o ExtractFile deixaria no disco).
func ExtractBytes(whiteBinPath, virtualPath string, coll *filelist.Collection) ([]byte, error) {
	vpath := normalizePath(virtualPath)
	refs := coll.FindByPath(vpath)
	if len(refs) == 0 {
		return nil, fmt.Errorf("arquivo não encontrado no filelist: %s", vpath)
	}

	f, err := os.Open(whiteBinPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	binSize := info.Size()

	var out []byte
	for _, ref := range refs {
		e := &coll.Chunks[ref.Chunk].Entries[ref.Entry]
		loc, err := filelist.ParseFileInfo(e.FileInfo)
		if err != nil {
			return nil, fmt.Errorf("fileInfo inválido (%q): %w", e.FileInfo, err)
		}

		pos := int64(loc.PosSectors) * 2048
		readLen := int64(loc.CmpSize)
		if !loc.IsCompressed() {
			readLen = int64(loc.UncmpSize)
		}
		if pos < 0 || readLen < 0 || pos+readLen > binSize {
			return nil, fmt.Errorf("região fora do white bin: %s (pos=%d len=%d bin=%d)",
				vpath, pos, readLen, binSize)
		}

		var buf bytes.Buffer
		if _, err := copyLocated(&buf, f, pos, loc); err != nil {
			return nil, fmt.Errorf("extraindo %s: %w", vpath, err)
		}
		out = buf.Bytes()
	}

	return out, nil
}

// writeOne lê a região indicada e grava o arquivo (descomprimindo se preciso).
func writeOne(f *os.File, pos int64, loc filelist.FileLocation, outFile string) (State, error) {
	out, err := os.Create(outFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	return copyLocated(out, f, pos, loc)
}

// copyLocated copia a região indicada do white bin para w (descomprimindo com
// zlib quando o arquivo está comprimido).
func copyLocated(w io.Writer, f *os.File, pos int64, loc filelist.FileLocation) (State, error) {
	if loc.IsCompressed() {
		sr := io.NewSectionReader(f, pos, int64(loc.CmpSize))
		zr, err := zlib.NewReader(sr)
		if err != nil {
			return "", fmt.Errorf("abrindo stream zlib: %w", err)
		}
		defer zr.Close()

		n, err := io.Copy(w, zr)
		if err != nil {
			return "", fmt.Errorf("descomprimindo: %w", err)
		}
		if uint32(n) != loc.UncmpSize {
			return "", fmt.Errorf("tamanho descomprimido %d != esperado %d", n, loc.UncmpSize)
		}
		return Decompressed, nil
	}

	sr := io.NewSectionReader(f, pos, int64(loc.UncmpSize))
	n, err := io.Copy(w, sr)
	if err != nil {
		return "", err
	}
	if uint32(n) != loc.UncmpSize {
		return "", fmt.Errorf("copiados %d bytes, esperado %d", n, loc.UncmpSize)
	}
	return Copied, nil
}

func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
