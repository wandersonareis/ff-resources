package whitebin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestAppendBlobUncompressed cobre o append básico: alinhamento de 2048,
// tamanhos iguais e leitura de volta pelo ExtractAt.
func TestAppendBlobUncompressed(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "white_test.win32.bin")

	// 100 bytes: o próximo setor começa em 2048.
	initial := bytes.Repeat([]byte{0xA5}, 100)
	if err := os.WriteFile(bin, initial, 0o644); err != nil {
		t.Fatal(err)
	}

	blob := []byte("conteudo do mod")
	res, err := AppendBlob(bin, blob, false)
	if err != nil {
		t.Fatalf("AppendBlob: %v", err)
	}
	if res.Offset != 2048 {
		t.Fatalf("offset = %d, esperado 2048", res.Offset)
	}
	if res.PosSectors != 1 {
		t.Fatalf("posSectors = %d, esperado 1", res.PosSectors)
	}
	if res.UncmpSize != uint32(len(blob)) || res.CmpSize != uint32(len(blob)) {
		t.Fatalf("tamanhos = %d/%d, esperado %d/%d",
			res.UncmpSize, res.CmpSize, len(blob), len(blob))
	}

	st, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(2048 + len(blob)); st.Size() != want {
		t.Fatalf("tamanho do bin = %d, esperado %d", st.Size(), want)
	}

	got, err := ExtractAt(bin, res.PosSectors, res.UncmpSize, res.CmpSize)
	if err != nil {
		t.Fatalf("ExtractAt: %v", err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("bytes lidos diferem: %q vs %q", got, blob)
	}
}

// TestAppendBlobCompressed cobre a compressão no nível do white bin: o
// payload gravado é menor que o original e o ExtractAt descomprime.
func TestAppendBlobCompressed(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "white_test.win32.bin")
	if err := os.WriteFile(bin, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	blob := bytes.Repeat([]byte("texto repetido para comprimir; "), 200)
	res, err := AppendBlob(bin, blob, true)
	if err != nil {
		t.Fatalf("AppendBlob: %v", err)
	}
	if res.UncmpSize != uint32(len(blob)) {
		t.Fatalf("uncmp = %d, esperado %d", res.UncmpSize, len(blob))
	}
	if res.CmpSize >= res.UncmpSize {
		t.Fatalf("cmp = %d deveria ser menor que uncmp = %d", res.CmpSize, res.UncmpSize)
	}
	if res.Offset != 2048 {
		t.Fatalf("offset = %d, esperado 2048", res.Offset)
	}

	got, err := ExtractAt(bin, res.PosSectors, res.UncmpSize, res.CmpSize)
	if err != nil {
		t.Fatalf("ExtractAt: %v", err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("bytes descomprimidos diferem")
	}
}

// TestAppendPayloadRefusesOverlap garante que o append não sobrescreve um
// setor que já cresceu além do planejado.
func TestAppendPayloadRefusesOverlap(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "white_test.win32.bin")
	if err := os.WriteFile(bin, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AppendPayload(bin, 2048, []byte("x")); err == nil {
		t.Fatal("AppendPayload deveria recusar offset dentro do arquivo")
	}
}
