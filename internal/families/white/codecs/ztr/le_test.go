package ztr_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
)

// TestReadLittleEndianPair monta o par LE à mão (layout do Pulse) e confere
// que Read detecta o tipo 1 e devolve chave/valor.
func TestReadLittleEndianPair(t *testing.T) {
	key := []byte("$foo")
	value := []byte("hello")

	keyOff := 12
	valueOff := keyOff + len(key) + 1
	raw := make([]byte, 0, valueOff+len(value)+2)
	var b4 [4]byte
	binary.LittleEndian.PutUint32(b4[:], 1)
	raw = append(raw, b4[:]...)
	binary.LittleEndian.PutUint32(b4[:], uint32(keyOff))
	raw = append(raw, b4[:]...)
	binary.LittleEndian.PutUint32(b4[:], uint32(valueOff))
	raw = append(raw, b4[:]...)
	raw = append(raw, key...)
	raw = append(raw, 0)
	raw = append(raw, value...)
	raw = append(raw, 0, 0)

	f, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("Read(LE pair): %v", err)
	}
	if len(f.IDs) != 1 || f.IDs[0] != "$foo" {
		t.Fatalf("IDs: %v", f.IDs)
	}
	if !bytes.Equal(f.Lines[0], []byte("hello\x00\x00")) {
		t.Fatalf("line: %q", f.Lines[0])
	}
}

// TestReadLittleEndianDictionary monta o dicionário LE à mão e confere a
// ordem das entradas.
func TestReadLittleEndianDictionary(t *testing.T) {
	ids := []string{"$a", "$bb"}
	texts := []string{"um", "dois"}
	count := len(ids)

	offsets := make([]uint32, count*2)
	pos := uint32(4 + count*2*4)
	for i := range ids {
		offsets[i*2] = pos
		pos += uint32(len(ids[i])) + 1
		offsets[i*2+1] = pos
		pos += uint32(len(texts[i])) + 2
	}

	raw := make([]byte, 0, pos)
	var b4 [4]byte
	binary.LittleEndian.PutUint32(b4[:], uint32(count))
	raw = append(raw, b4[:]...)
	for _, off := range offsets {
		binary.LittleEndian.PutUint32(b4[:], off)
		raw = append(raw, b4[:]...)
	}
	for i := range ids {
		raw = append(raw, []byte(ids[i])...)
		raw = append(raw, 0)
		raw = append(raw, []byte(texts[i])...)
		raw = append(raw, 0, 0)
	}

	f, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("Read(LE dict): %v", err)
	}
	if len(f.IDs) != count {
		t.Fatalf("IDs: %v", f.IDs)
	}
	for i := range ids {
		if f.IDs[i] != ids[i] {
			t.Errorf("ID %d: %q, quer %q", i, f.IDs[i], ids[i])
		}
		if !bytes.Equal(f.Lines[i], []byte(texts[i]+"\x00\x00")) {
			t.Errorf("linha %d: %q", i, f.Lines[i])
		}
	}
}

// TestReadLittleEndianSingleZeroValue cobre os .ztr antigos do part1 que
// terminam o valor com UM zero (em vez de 00 00). O parser usa o offset da
// próxima chave como limite, então aceita os dois terminadores.
func TestReadLittleEndianSingleZeroValue(t *testing.T) {
	ids := []string{"$a", "$b"}
	texts := []string{"um", "dois"}
	count := len(ids)
	offsets := make([]uint32, count*2)
	pos := uint32(4 + count*2*4)
	for i := range ids {
		offsets[i*2] = pos
		pos += uint32(len(ids[i])) + 1
		offsets[i*2+1] = pos
		pos += uint32(len(texts[i])) + 1 // só UM zero
	}
	raw := make([]byte, 0, pos)
	var b4 [4]byte
	binary.LittleEndian.PutUint32(b4[:], uint32(count))
	raw = append(raw, b4[:]...)
	for _, off := range offsets {
		binary.LittleEndian.PutUint32(b4[:], off)
		raw = append(raw, b4[:]...)
	}
	for i := range ids {
		raw = append(raw, []byte(ids[i])...)
		raw = append(raw, 0)
		raw = append(raw, []byte(texts[i])...)
		raw = append(raw, 0)
	}

	f, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("Read(LE single-zero): %v", err)
	}
	for i := range ids {
		if f.IDs[i] != ids[i] || !bytes.Equal(f.Lines[i], []byte(texts[i]+"\x00\x00")) {
			t.Errorf("entrada %d: (%q,%q)", i, f.IDs[i], f.Lines[i])
		}
	}
}

// TestWriteLERoundTrip confere os dois tipos de saída LE do WriteLE.
func TestWriteLERoundTrip(t *testing.T) {
	cases := []struct {
		name string
		file *ztr.File
	}{
		{
			name: "pair",
			file: &ztr.File{IDs: []string{"$one"}, Lines: [][]byte{[]byte("texto\x00\x00")}},
		},
		{
			name: "dictionary",
			file: &ztr.File{
				IDs:   []string{"$a", "$bb", "$ccc"},
				Lines: [][]byte{[]byte("um\x00\x00"), []byte("dois\x00\x00"), []byte("três\x00\x00")},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw, err := ztr.WriteLE(c.file)
			if err != nil {
				t.Fatalf("WriteLE: %v", err)
			}
			f, err := ztr.Read(raw)
			if err != nil {
				t.Fatalf("Read(WriteLE): %v", err)
			}
			if len(f.IDs) != len(c.file.IDs) {
				t.Fatalf("IDs: %v", f.IDs)
			}
			for i := range c.file.IDs {
				if f.IDs[i] != c.file.IDs[i] {
					t.Errorf("ID %d: %q, quer %q", i, f.IDs[i], c.file.IDs[i])
				}
				if !bytes.Equal(f.Lines[i], c.file.Lines[i]) {
					t.Errorf("linha %d: %q, quer %q", i, f.Lines[i], c.file.Lines[i])
				}
			}
		})
	}
}

// TestWriteFormatBigEndian mantém a regressão do formato padrão do app.
func TestWriteFormatBigEndian(t *testing.T) {
	f := &ztr.File{
		IDs:   []string{"$a", "$b"},
		Lines: [][]byte{[]byte("um\x00\x00"), []byte("dois\x00\x00")},
	}
	raw, err := ztr.WriteFormat(f, ztr.FormatBigEndianCompressed, ztr.Compressed, ztr.TieMinLast)
	if err != nil {
		t.Fatalf("WriteFormat(BE): %v", err)
	}
	got, err := ztr.Read(raw)
	if err != nil {
		t.Fatalf("Read(BE): %v", err)
	}
	if len(got.IDs) != 2 || got.IDs[0] != "$a" || !bytes.Equal(got.Lines[1], []byte("dois\x00\x00")) {
		t.Fatalf("round-trip BE: %+v", got)
	}
}
