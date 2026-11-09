package wpd

import (
	"bytes"
	"testing"
)

// TestWriteFromPartsRoundTrip confere que Read -> WriteFromParts -> Read
// devolve os MESMOS registros e payloads, com o magic preservado. E o que
// garante que o reempacotamento do FF13 (container WPD com o .ztr
// reescrito) mantem os registros alheios byte a byte.
func TestWriteFromPartsRoundTrip(t *testing.T) {
	parts := []Part{
		{Name: "auto_c01", Extension: "wdb", Data: []byte("WDB-BYTES")},
		{Name: "auto_c01_us", Extension: "ztr", Data: []byte{0x00, 0x01, 0x02, 0x00, 0x00}},
		{Name: "sistema", Extension: "bin", Data: bytes.Repeat([]byte{0xAB}, 300)},
	}

	blob, err := WriteFromParts(parts)
	if err != nil {
		t.Fatalf("WriteFromParts: %v", err)
	}
	f, err := Read(blob)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if f.Header.Magic != Magic {
		t.Errorf("magic: %#x, esperado %#x", f.Header.Magic, Magic)
	}
	if len(f.Header.Records) != len(parts) {
		t.Fatalf("registros: %d, esperado %d", len(f.Header.Records), len(parts))
	}
	for i, r := range f.Header.Records {
		if r.Name != parts[i].Name || r.Extension != parts[i].Extension {
			t.Errorf("registro %d: %s.%s, esperado %s.%s", i, r.Name, r.Extension, parts[i].Name, parts[i].Extension)
		}
		got, ok := f.ExtractRecord(r.Name, r.Extension)
		if !ok {
			t.Errorf("registro %d ausente no arquivo", i)
			continue
		}
		if !bytes.Equal(got, parts[i].Data) {
			t.Errorf("registro %d: payload diferente (%d bytes)", i, len(got))
		}
	}

	// Reempacotar o que foi lido devolve o mesmo conteudo.
	again := make([]Part, 0, len(f.Header.Records))
	for _, r := range f.Header.Records {
		data, ok := f.ExtractRecord(r.Name, r.Extension)
		if !ok {
			t.Fatalf("registro %s.%s ausente", r.Name, r.Extension)
		}
		again = append(again, Part{Name: r.Name, Extension: r.Extension, Data: data})
	}
	blob2, err := WriteFromParts(again)
	if err != nil {
		t.Fatalf("WriteFromParts (2): %v", err)
	}
	f2, err := Read(blob2)
	if err != nil {
		t.Fatalf("Read (2): %v", err)
	}
	for i, r := range f2.Header.Records {
		got, ok := f2.ExtractRecord(r.Name, r.Extension)
		if !ok {
			t.Errorf("registro %d ausente no round-trip", i)
			continue
		}
		if !bytes.Equal(got, parts[i].Data) {
			t.Errorf("round-trip do registro %d difere", i)
		}
	}
}

// TestReadTable confere a leitura so do cabecalho + tabela (sem os
// payloads): e o que a varredura do FF13 usa para saber se um .bin e um
// container WPD sem ler o arquivo inteiro.
func TestReadTable(t *testing.T) {
	parts := []Part{
		{Name: "auto_c01", Extension: "wdb", Data: []byte("WDB-BYTES")},
		{Name: "auto_c01_us", Extension: "ztr", Data: []byte{0x01, 0x02}},
	}
	blob, err := WriteFromParts(parts)
	if err != nil {
		t.Fatalf("WriteFromParts: %v", err)
	}
	table := blob[:HeaderOffset+len(parts)*RecordSize]
	h, err := ReadTable(table)
	if err != nil {
		t.Fatalf("ReadTable: %v", err)
	}
	if h.Magic != Magic {
		t.Errorf("magic: %#x", h.Magic)
	}
	if len(h.Records) != len(parts) {
		t.Fatalf("registros: %d, esperado %d", len(h.Records), len(parts))
	}
	if h.Records[1].Name != "auto_c01_us" || h.Records[1].Extension != "ztr" {
		t.Errorf("registro 1: %s.%s", h.Records[1].Name, h.Records[1].Extension)
	}
}
