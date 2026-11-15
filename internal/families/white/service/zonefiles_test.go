package service

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"ffresources/internal/families/white/profile/config"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/profile/parts"
)

// TestZoneFileStatus lista TODOS os arquivos de white_data/zone e diz, para
// cada um, se tem índice (par) e qual é.
func TestZoneFileStatus(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	filelists, bins := ScanPartFiles(p)
	indexes := DiscoverFilelists(p, roots, bins)

	// Índice -> chave de par e vice-versa.
	byKey := map[string]string{} // chave -> nome do índice (real ou .bin2)
	for key := range filelists {
		byKey[key] = filepath.Base(filelists[key])
	}
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		base := filepath.Base(idx.Virtual)
		key, ok := p.PairKey(base)
		if !ok {
			continue
		}
		if strings.HasSuffix(base, ".bin2") {
			byKey[key+"#2"] = base
			continue
		}
		byKey[key] = base
	}
	// As entradas .bin2 só aparecem dentro dos índices globais (não são
	// extraídas): varre os índices globais para achá-las.
	for _, root := range roots {
		coll, lerr := LoadFilelist(root.Path, "part1")
		if lerr != nil {
			continue
		}
		for _, ep := range coll.Paths() {
			base := filepath.Base(ep)
			if !strings.HasSuffix(strings.ToLower(base), ".bin2") {
				continue
			}
			key, ok := p.PairKey(base)
			if ok {
				byKey[key+"#2"] = base
			}
		}
	}

	zone := filepath.Join(p.DataDir(config.GameRoot("part1")), "zone")
	entries, err := os.ReadDir(zone)
	if err != nil {
		t.Fatalf("lendo zone: %v", err)
	}
	type row struct {
		name   string
		size   int64
		status string
	}
	rows := []row{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".bin") {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		key, ok := p.PairKey(e.Name())
		if !ok {
			continue
		}
		status := "SEM PAR"
		if idxName, found := byKey[key]; found {
			status = "par: " + idxName
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), "_img2.win32.bin") {
			if idxName, found := byKey[key+"#2"]; found {
				status = "par: " + idxName
			} else {
				status = "SEM PAR (2o arquivo)"
			}
		}
		rows = append(rows, row{e.Name(), info.Size(), status})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	semPar := 0
	for _, r := range rows {
		if strings.HasPrefix(r.status, "SEM PAR") {
			semPar++
			t.Logf("SEM PAR: %-32s %10d bytes", r.name, r.size)
		}
	}
	t.Logf("zone: %d arquivos, %d sem par", len(rows), semPar)
}

// TestOrphanBinSelfIndex procura, DENTRO dos white bins órfãos, um índice
// (filelist) embutido: varre o bin em blocos de 2048 bytes e tenta parsear
// um filelist em cada ponto plausível.
func TestOrphanBinSelfIndex(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	filelists, bins := ScanPartFiles(p)
	indexes := DiscoverFilelists(p, roots, bins)
	covered := map[string]bool{}
	for key := range filelists {
		covered[key] = true
	}
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		if key, ok := p.PairKey(filepath.Base(idx.Virtual)); ok {
			covered[key] = true
		}
	}

	const window = 4 << 20 // 4 MB por tentativa de parse
	for key, bin := range bins {
		if covered[key] {
			continue
		}
		found, off, err := scanFilelistInBin(bin, window)
		if err != nil {
			t.Logf("%s: erro na varredura: %v", filepath.Base(bin), err)
			continue
		}
		if found != nil {
			t.Logf("ACHEI índice embutido em %s (offset %d, %d entradas)", filepath.Base(bin), off, found.FileCount())
		} else {
			t.Logf("sem índice embutido: %s", filepath.Base(bin))
		}
	}
}

// TestOrphanBinCompressedIndex procura, dentro dos white bins órfãos, um
// índice EMBUTIDO COMPRIMIDO (zlib) — o formato em que os índices de zona
// vivem dentro do bin global.
func TestOrphanBinCompressedIndex(t *testing.T) {
	testGameRoot(t, "part1")

	p := parts.Get("part1")
	var svc Service
	roots, err := svc.ListFilelistRoots("part1")
	if err != nil {
		t.Fatalf("ListFilelistRoots: %v", err)
	}
	filelists, bins := ScanPartFiles(p)
	indexes := DiscoverFilelists(p, roots, bins)
	covered := map[string]bool{}
	for key := range filelists {
		covered[key] = true
	}
	for _, idx := range indexes {
		if idx.Level == 0 {
			continue
		}
		if key, ok := p.PairKey(filepath.Base(idx.Virtual)); ok {
			covered[key] = true
		}
	}

	// Controle positivo: o bin global CONTÉM o índice de zona (comprimido).
	for _, idx := range indexes {
		if idx.Level != 1 {
			continue
		}
		parent := idx.Parent
		container := ""
		for _, root := range roots {
			if root.Path == parent {
				container = root.WhiteBin
			}
		}
		if container == "" {
			continue
		}
		found, off, err := scanCompressedFilelist(container, 1<<20)
		t.Logf("CONTROLE %s: encontrado=%v offset=%d err=%v", filepath.Base(container), found != nil, off, err)
		break
	}

	for key, bin := range bins {
		if covered[key] {
			continue
		}
		found, off, err := scanCompressedFilelist(bin, 1<<20)
		if err != nil {
			t.Logf("%s: erro: %v", filepath.Base(bin), err)
			continue
		}
		if found != nil {
			t.Logf("ACHEI índice comprimido em %s (offset %d, %d entradas)", filepath.Base(bin), off, found.FileCount())
		} else {
			t.Logf("sem índice comprimido: %s", filepath.Base(bin))
		}
	}
}

// scanCompressedFilelist procura streams zlib que, descomprimidos, formem um
// filelist válido. maxIn limita o tamanho lido de cada stream.
func scanCompressedFilelist(path string, maxIn int64) (*fl.Collection, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	size := info.Size()
	const sector = 2048
	buf := make([]byte, maxIn)
	for off := int64(0); off+2 <= size; off += sector {
		head := make([]byte, 2)
		if _, err := f.ReadAt(head, off); err != nil {
			break
		}
		if head[0] != 0x78 || (head[1] != 0xDA && head[1] != 0x9C && head[1] != 0x01) {
			continue
		}
		n, _ := f.ReadAt(buf, off)
		if n < 2 {
			continue
		}
		zr, zerr := zlib.NewReader(bytes.NewReader(buf[:n]))
		if zerr != nil {
			continue
		}
		out, derr := io.ReadAll(io.LimitReader(zr, 8<<20))
		zr.Close()
		if derr != nil || len(out) < 16 {
			continue
		}
		// Guarda de sanidade: o cabeçalho precisa ter contagens plausíveis
		// (senão o Parse aloca gigabytes).
		if binary.LittleEndian.Uint32(out[8:12]) == 0 || binary.LittleEndian.Uint32(out[8:12]) > 500000 {
			continue
		}
		if binary.LittleEndian.Uint32(out[0:4]) > 1<<20 || binary.LittleEndian.Uint32(out[4:8]) > 1<<28 {
			continue
		}
		coll, perr := fl.Parse(out, fl.GameFF131)
		if perr == nil && coll.FileCount() > 0 {
			return coll, off, nil
		}
	}
	return nil, 0, nil
}

// scanFilelistInBin procura um filelist dentro do bin, alinhado em setores de
// 2048 bytes. Só tenta parsear onde o cabeçalho é plausível.
func scanFilelistInBin(path string, window int) (*fl.Collection, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	size := info.Size()
	const sector = 2048
	attempts := 0
	for off := int64(0); off+12 <= size; off += sector {
		header := make([]byte, 12)
		if _, err := f.ReadAt(header, off); err != nil {
			break
		}
		chunkInfo := binary.LittleEndian.Uint32(header[0:])
		chunkData := binary.LittleEndian.Uint32(header[4:])
		fileCount := binary.LittleEndian.Uint32(header[8:])
		if fileCount == 0 || fileCount > 500000 {
			continue
		}
		if chunkInfo > 1<<20 || chunkData > 1<<28 {
			continue
		}
		attempts++
		if attempts > 32 {
			break
		}
		buf := make([]byte, window)
		n, _ := f.ReadAt(buf, off)
		if n < 12 {
			continue
		}
		coll, perr := fl.Parse(buf[:n], fl.GameFF131)
		if perr == nil && coll.FileCount() > 0 {
			return coll, off, nil
		}
	}
	return nil, 0, nil
}
