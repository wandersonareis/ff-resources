package imgbcore

import (
	"bytes"
	"testing"
)

func TestColorAsBGRA(t *testing.T) {
	in := []byte{1, 2, 3, 4, 10, 20, 30, 40} // A R G B | A R G B
	got := ColorAsBGRA(in)
	want := []byte{4, 3, 2, 1, 40, 30, 20, 10}
	if !bytes.Equal(got, want) {
		t.Fatalf("ColorAsBGRA = %v, want %v", got, want)
	}
}

// Para width=4/height=2 o desembaralhamento Morton produz a permutação
// m -> pixel: [0,1,4,5,2,3,6,7]. Escrevendo in[m] no pixel de destino,
// o resultado passa a ser [0,1,4,5,2,3,6,7].
func TestMortonUnswizzle4x2(t *testing.T) {
	g := &GTEX{Width: 4, Height: 2}
	swizzled := make([]byte, 4*2*4)
	for m := 0; m < 8; m++ {
		swizzled[m*4] = byte(m)
	}

	got := MortonUnswizzle(g, swizzled)

	want := []byte{0, 1, 4, 5, 2, 3, 6, 7}
	for p, w := range want {
		if got[p*4] != w {
			t.Fatalf("pixel %d = %d, want %d (buffer=%v)", p, got[p*4], w, got)
		}
	}
}

func TestMortonUnswizzle1x1Identity(t *testing.T) {
	g := &GTEX{Width: 1, Height: 1}
	in := []byte{9, 8, 7, 6}
	if got := MortonUnswizzle(g, in); !bytes.Equal(got, in) {
		t.Fatalf("1x1 deve ser identidade, got %v", got)
	}
}

// Uma imagem swizzled (formato 4 + tipo 4) deve ser desembaralhada e ter a
// ordem dos canais corrigida para BGRA.
func TestProcessPS3ImageDataSwizzled(t *testing.T) {
	g := &GTEX{Width: 4, Height: 2, Format: GTEXFormatRGBA, Type: GTEXTypeClassicConsole}

	swizzled := make([]byte, 4*2*4)
	for m := 0; m < 8; m++ {
		swizzled[m*4+0] = byte(m) // A
		swizzled[m*4+1] = 0x11    // R
		swizzled[m*4+2] = 0x22    // G
		swizzled[m*4+3] = 0x33    // B
	}

	got := ProcessPS3ImageData(g, swizzled)

	// O pixel de destino 2 recebe a origem 4: A=4,R=0x11,G=0x22,B=0x33
	// e após a correção de cor fica B=0x33,G=0x22,R=0x11,A=4.
	want := []byte{0x33, 0x22, 0x11, 4}
	if !bytes.Equal(got[2*4:2*4+4], want) {
		t.Fatalf("pixel 2 = %v, want %v", got[2*4:2*4+4], want)
	}
}

// Sem o flag de swizzle (tipo 0) apenas a cor é corrigida.
func TestProcessPS3ImageDataColorOnly(t *testing.T) {
	g := &GTEX{Width: 1, Height: 1, Format: GTEXFormatRGBA, Type: GTEXTypeClassic}
	got := ProcessPS3ImageData(g, []byte{1, 2, 3, 4})
	if want := []byte{4, 3, 2, 1}; !bytes.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Formatos comprimidos (ex.: DXT neste port) passam inalterados.
func TestProcessPS3ImageDataCompressedUntouched(t *testing.T) {
	g := &GTEX{Width: 1, Height: 1, Format: 0 /* DXT1 */, Type: GTEXTypeClassicConsole}
	in := []byte{1, 2, 3, 4}
	if got := ProcessPS3ImageData(g, in); !bytes.Equal(got, in) {
		t.Fatalf("formato comprimido não deve ser alterado, got %v", got)
	}
}

func TestParseGTEXAcceptsConsoleTypes(t *testing.T) {
	for _, typ := range []GTEXType{GTEXTypeClassicConsole, GTEXTypeCubemapConsole} {
		buf := make([]byte, 32)
		copy(buf[0:4], []byte("GTEX"))
		h := 4
		buf[h+0] = 1                    // Version
		buf[h+2] = byte(GTEXFormatRGBA) // Format
		buf[h+3] = 1                    // MipCount
		buf[h+5] = byte(typ)            // Type
		buf[h+6] = 16                   // Width lo
		buf[h+8] = 16                   // Height lo

		g, err := ParseGTEX(buf)
		if err != nil {
			t.Fatalf("ParseGTEX type %d: %v", typ, err)
		}
		if g.Type != typ || !g.IsValid {
			t.Fatalf("type %d não reconhecido: %+v", typ, g)
		}
	}
}
