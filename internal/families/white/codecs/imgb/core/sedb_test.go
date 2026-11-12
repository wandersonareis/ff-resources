package imgbcore

import (
	"encoding/binary"
	"testing"
)

// TestParseTRBLayout fixes the SeDbResHeader layout do Pulse: a contagem fica
// em 0x30, a tabela em 0x40 e o Offset das entradas é relativo a
// Count*16+0x40.
func TestParseTRBLayout(t *testing.T) {
	buf := make([]byte, 0x40+16+16)
	copy(buf[0:8], []byte("SEDBRES "))
	binary.LittleEndian.PutUint32(buf[8:12], 4000)
	binary.LittleEndian.PutUint32(buf[12:16], 0x00400200)
	binary.LittleEndian.PutUint32(buf[48:52], 1) // Count
	binary.LittleEndian.PutUint32(buf[52:56], 0)
	binary.LittleEndian.PutUint32(buf[56:60], 0)
	binary.LittleEndian.PutUint32(buf[60:64], 0)
	// entry 0: Index=0, Offset=0, Length=16, Unknown=2
	off := 0x40
	binary.LittleEndian.PutUint32(buf[off+0:off+4], 0)
	binary.LittleEndian.PutUint32(buf[off+4:off+8], 0)
	binary.LittleEndian.PutUint32(buf[off+8:off+12], 16)
	binary.LittleEndian.PutUint32(buf[off+12:off+16], 2)
	copy(buf[0x50:], []byte("SEDBtxb\x00"))

	h, entries, err := ParseTRB(buf)
	if err != nil {
		t.Fatal(err)
	}
	if h.Count != 1 || len(entries) != 1 {
		t.Fatalf("count=%d entries=%d", h.Count, len(entries))
	}
	if h.DataOffset != 0x50 {
		t.Fatalf("DataOffset=%#x, quer 0x50", h.DataOffset)
	}
	sec, err := h.SectionData(buf, entries[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(sec[:8]) != "SEDBtxb\x00" {
		t.Fatalf("seção=%q", sec[:8])
	}
}
