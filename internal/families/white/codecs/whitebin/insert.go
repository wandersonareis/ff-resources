package whitebin

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"

	"ffresources/internal/families/white/codecs/filelist"
)

// INSERÇÃO no white bin: o arquivo novo vai para o FIM do container,
// alinhado a 2048 bytes (um setor), e o filelist passa a apontar para lá.
//
// O formato não permite criar entradas novas: só reapontar caminhos que já
// existem no índice. Por isso a inserção é sempre feita em DUAS fases:
//
//  1. PrepareBlob — decide a compressão e monta os bytes que serão gravados;
//  2. AppendPayload — grava no offset planejado.
//
// A separação permite ao serviço montar o plano inteiro (vários arquivos,
// incluindo os filelists embutidos na cadeia) antes de tocar no disco.

// AppendResult descreve onde um blob foi gravado.
type AppendResult struct {
	// Offset é o offset real em bytes (múltiplo de 2048).
	Offset int64
	// PosSectors é o valor gravado no fileInfo (offset / 2048).
	PosSectors uint64
	// UncmpSize é o tamanho descomprimido (o tamanho do arquivo original).
	UncmpSize uint32
	// CmpSize é o tamanho efetivamente armazenado (igual a UncmpSize quando
	// não comprimido).
	CmpSize uint32
}

// PreparedBlob são os bytes que vão para o white bin e os tamanhos que o
// filelist deve registrar.
type PreparedBlob struct {
	Payload   []byte
	UncmpSize uint32
	CmpSize   uint32
}

// PrepareBlob monta o payload de um arquivo. Quando compress=true, aplica
// zlib (a mesma compressão que o WhiteBinTools usa no nível do container);
// senão grava os bytes como estão.
func PrepareBlob(data []byte, compress bool) (PreparedBlob, error) {
	if uint64(len(data)) > 0xFFFFFFFF {
		return PreparedBlob{}, fmt.Errorf("arquivo grande demais para o filelist: %d bytes", len(data))
	}
	if !compress {
		return PreparedBlob{
			Payload:   data,
			UncmpSize: uint32(len(data)),
			CmpSize:   uint32(len(data)),
		}, nil
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return PreparedBlob{}, fmt.Errorf("comprimindo arquivo: %w", err)
	}
	if err := zw.Close(); err != nil {
		return PreparedBlob{}, fmt.Errorf("comprimindo arquivo: %w", err)
	}
	if uint64(buf.Len()) > 0xFFFFFFFF {
		return PreparedBlob{}, fmt.Errorf("arquivo comprimido grande demais: %d bytes", buf.Len())
	}
	return PreparedBlob{
		Payload:   buf.Bytes(),
		UncmpSize: uint32(len(data)),
		CmpSize:   uint32(buf.Len()),
	}, nil
}

// AppendPayload grava o payload no offset indicado, preenchendo com zeros o
// espaço entre o fim atual do arquivo e o offset (o alinhamento de 2048).
//
// O offset NUNCA pode apontar para dentro de dados já existentes: se o
// arquivo tiver crescido além do planejado, a gravação é recusada para não
// corromper um setor que outra escrita já usa.
func AppendPayload(whiteBinPath string, offset int64, payload []byte) error {
	if offset < 0 {
		return fmt.Errorf("offset negativo: %d", offset)
	}
	f, err := os.OpenFile(whiteBinPath, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size > offset {
		return fmt.Errorf("white bin cresceu durante a inserção: tamanho %d > offset %d", size, offset)
	}
	if size < offset {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return err
		}
		if err := writeZeros(f, offset-size); err != nil {
			return fmt.Errorf("alinhando white bin: %w", err)
		}
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	if _, err := f.Write(payload); err != nil {
		return err
	}
	return f.Sync()
}

// AppendBlob é o caminho simples (um arquivo, já com o container aberto no
// tamanho atual): calcula o alinhamento, prepara e grava.
//
// O serviço usa PrepareBlob + AppendPayload para planejar em lote; esta
// função fica para testes e para ferramentas pontuais.
func AppendBlob(whiteBinPath string, data []byte, compress bool) (AppendResult, error) {
	info, err := os.Stat(whiteBinPath)
	if err != nil {
		return AppendResult{}, err
	}
	aligned, posSectors := filelist.PlanAppend(info.Size())
	prepared, err := PrepareBlob(data, compress)
	if err != nil {
		return AppendResult{}, err
	}
	if err := AppendPayload(whiteBinPath, aligned, prepared.Payload); err != nil {
		return AppendResult{}, err
	}
	return AppendResult{
		Offset:     aligned,
		PosSectors: posSectors,
		UncmpSize:  prepared.UncmpSize,
		CmpSize:    prepared.CmpSize,
	}, nil
}

// ExtractAt lê um arquivo direto do white bin por posição/tamanhos, sem
// precisar da Collection. É o que a verificação pós-inserção usa para
// conferir o que foi gravado.
func ExtractAt(whiteBinPath string, posSectors uint64, uncmpSize, cmpSize uint32) ([]byte, error) {
	f, err := os.Open(whiteBinPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	pos := int64(posSectors) * 2048
	readLen := int64(cmpSize)
	loc := filelist.FileLocation{PosSectors: posSectors, UncmpSize: uncmpSize, CmpSize: cmpSize}
	if !loc.IsCompressed() {
		readLen = int64(uncmpSize)
	}
	if pos < 0 || readLen < 0 || pos+readLen > info.Size() {
		return nil, fmt.Errorf("região fora do white bin (pos=%d len=%d bin=%d)", pos, readLen, info.Size())
	}
	var buf bytes.Buffer
	if _, err := copyLocated(&buf, f, pos, loc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeZeros grava n bytes zero em w, em blocos, para não alocar um buffer
// do tamanho do alinhamento.
func writeZeros(w io.Writer, n int64) error {
	var zeros [4096]byte
	for n > 0 {
		chunk := int64(len(zeros))
		if n < chunk {
			chunk = n
		}
		if _, err := w.Write(zeros[:chunk]); err != nil {
			return err
		}
		n -= chunk
	}
	return nil
}
