package encoding

import (
	"bytes"
	"strings"
	"testing"
)

// accentPair é um par (lead 0x85, trail) -> runa, com o par canônico usado
// pelos binários de data/ do FF13.
type accentPair struct {
	trail byte
	r     rune
}

// sourceAccents são os acentuados efetivamente encontrados nos binários de
// data/ (part1), com os idiomas em que aparecem:
//
//	85:80 ¿ sp                85:A0 Á sp/br        85:A1 Â fr/br
//	85:A2 Ã br                85:A3 Ä gr           85:A6 Ç fr/br
//	85:A7 È it/fr             85:A8 É sp/fr/br     85:A9 Ê fr/br
//	85:AC Í sp/br             85:B2 Ó sp/br        85:B3 Ô fr/br
//	85:B4 Õ br                85:B5 Ö gr           85:B9 Ú sp/br
//	85:BA Û fr                85:BB Ü gr
var sourceAccents = []accentPair{
	{0x80, '¿'},
	{0xA0, 'Á'},
	{0xA1, 'Â'},
	{0xA2, 'Ã'},
	{0xA3, 'Ä'},
	{0xA6, 'Ç'},
	{0xA7, 'È'},
	{0xA8, 'É'},
	{0xA9, 'Ê'},
	{0xAC, 'Í'},
	{0xB2, 'Ó'},
	{0xB3, 'Ô'},
	{0xB4, 'Õ'},
	{0xB5, 'Ö'},
	{0xB9, 'Ú'},
	{0xBA, 'Û'},
	{0xBB, 'Ü'},
}

func accentedCodec(t *testing.T) *TextCodec {
	t.Helper()
	if err := PrepareTextCodec(FF131, LatinJapanese); err != nil {
		t.Fatalf("PrepareTextCodec: %v", err)
	}
	tc, err := TextCodecFor(FF131, LatinJapanese)
	if err != nil {
		t.Fatalf("TextCodecFor: %v", err)
	}
	return tc
}

// TestAccentedGlyphsRoundTrip prova, para todos os acentuados da fonte e
// para o bloco Latin-1 inteiro, que:
//
//	raw0 -> decode -> encode -> decode -> encode
//
// mantém o TEXTO e os BYTES estáveis. Nos pares canônicos de data/, o
// encode devolve exatamente o par de origem (nada de {RAW:..}).
func TestAccentedGlyphsRoundTrip(t *testing.T) {
	tc := accentedCodec(t)

	pairs := append([]accentPair{}, sourceAccents...)
	// Bloco canônico completo: 0x9F..0xDE = U+00C0..U+00FF (À..ÿ).
	for trail := 0x9F; trail <= 0xDE; trail++ {
		pairs = append(pairs, accentPair{byte(trail), rune(0x00C0 + trail - 0x9F)})
	}

	for _, p := range pairs {
		raw0 := []byte{0x85, p.trail, 0x00, 0x00}
		text0, err := tc.DecodeLine(raw0)
		if err != nil {
			t.Fatalf("DecodeLine(85:%02X): %v", p.trail, err)
		}
		if want := string(p.r); text0 != want {
			t.Errorf("DecodeLine(85:%02X) = %q; queria %q", p.trail, text0, want)
		}
		if strings.Contains(text0, "{RAW") {
			t.Errorf("DecodeLine(85:%02X) ainda emite tag RAW: %q", p.trail, text0)
		}

		raw1, err := tc.EncodeLine(text0)
		if err != nil {
			t.Fatalf("EncodeLine(%q): %v", text0, err)
		}
		if !bytes.Equal(raw1, raw0) {
			t.Errorf("EncodeLine(%q) = % X; queria o par canônico % X", text0, raw1, raw0)
		}

		text1, err := tc.DecodeLine(raw1)
		if err != nil {
			t.Fatalf("DecodeLine(encode(%q)): %v", text0, err)
		}
		raw2, err := tc.EncodeLine(text1)
		if err != nil {
			t.Fatalf("EncodeLine(%q) no 2º passo: %v", text1, err)
		}
		if text1 != text0 {
			t.Errorf("texto instável em 85:%02X: %q vs %q", p.trail, text0, text1)
		}
		if !bytes.Equal(raw2, raw1) {
			t.Errorf("bytes instáveis em 85:%02X: % X vs % X", p.trail, raw1, raw2)
		}
	}
}

// TestLegacyAccentedAliases garante que os pares do bloco compacto legado
// (0x81..0x9D) continuam decodificando os mesmos caracteres, mas o encode
// normaliza para o par canônico de data/ (A0..BC). Isso mantém a leitura de
// arquivos antigos sem perpetuar os bytes legados.
func TestLegacyAccentedAliases(t *testing.T) {
	tc := accentedCodec(t)

	cases := []struct {
		trail     byte
		r         rune
		canonical byte
	}{
		{0x7F, '¿', 0x80},
		{0x81, 'Á', 0xA0},
		{0x87, 'Ç', 0xA6},
		{0x89, 'É', 0xA8},
		{0x9D, 'Ý', 0xBC},
	}

	for _, c := range cases {
		raw0 := []byte{0x85, c.trail, 0x00, 0x00}
		text0, err := tc.DecodeLine(raw0)
		if err != nil {
			t.Fatalf("DecodeLine(85:%02X): %v", c.trail, err)
		}
		if want := string(c.r); text0 != want {
			t.Errorf("DecodeLine(85:%02X) = %q; queria %q", c.trail, text0, want)
		}

		raw1, err := tc.EncodeLine(text0)
		if err != nil {
			t.Fatalf("EncodeLine(%q): %v", text0, err)
		}
		wantPair := []byte{0x85, c.canonical, 0x00, 0x00}
		if !bytes.Equal(raw1, wantPair) {
			t.Errorf("EncodeLine(%q) = % X; queria normalizar para % X", text0, raw1, wantPair)
		}

		text1, err := tc.DecodeLine(raw1)
		if err != nil {
			t.Fatalf("DecodeLine(normalizado %q): %v", text0, err)
		}
		raw2, err := tc.EncodeLine(text1)
		if err != nil {
			t.Fatalf("EncodeLine(%q) no 2º passo: %v", text1, err)
		}
		if text1 != text0 {
			t.Errorf("texto instável em 85:%02X: %q vs %q", c.trail, text0, text1)
		}
		if !bytes.Equal(raw2, raw1) {
			t.Errorf("bytes instáveis em 85:%02X: % X vs % X", c.trail, raw1, raw2)
		}
	}
}

// TestAccentedSourceLines usa as frases reais dos binários de data/ e
// confere encode -> decode -> encode sem perda de texto nem de bytes.
func TestAccentedSourceLines(t *testing.T) {
	tc := accentedCodec(t)

	lines := []string{
		"Genial, ¿y ahora?",
		"¿Dónde estás, Serah?",
		"SITUAÇÃO",
		"PARAMÈTRES",
		"CRITÈRE",
		"Öffentlichkeit",
		"OPÇÕES",
		"CINEMÁTICA",
		"Última Esperança",
		"ÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖØÙÚÛÜÝ",
		"àáâãäåæçèéêëìíîïðñòóôõö÷øùúûüýþÿ",
		"¡¿€‚„…†‡‰Š‹ŒŽ‘’“”•‐—™š›œžŸ",
	}

	for _, want := range lines {
		raw0, err := tc.EncodeLine(want)
		if err != nil {
			t.Fatalf("EncodeLine(%q): %v", want, err)
		}
		text0, err := tc.DecodeLine(raw0)
		if err != nil {
			t.Fatalf("DecodeLine(%q): %v", want, err)
		}
		if text0 != want {
			t.Errorf("texto = %q; queria %q", text0, want)
		}
		if strings.Contains(text0, "{RAW") {
			t.Errorf("%q contém tag RAW", text0)
		}
		raw1, err := tc.EncodeLine(text0)
		if err != nil {
			t.Fatalf("EncodeLine(%q) no 2º passo: %v", text0, err)
		}
		if !bytes.Equal(raw1, raw0) {
			t.Errorf("bytes instáveis em %q: % X vs % X", want, raw0, raw1)
		}
	}
}

// TestSourceAccentRawBytes fixa o caso do relatório:
// {RAW:85:80} era "Genial, ¿y ahora?" e {RAW:85:A8} era "É".
func TestSourceAccentRawBytes(t *testing.T) {
	tc := accentedCodec(t)

	raw := append([]byte("Genial, "), 0x85, 0x80)
	raw = append(raw, []byte("y ahora?")...)
	raw = append(raw, 0x00, 0x00)
	const want = "Genial, ¿y ahora?"

	text, err := tc.DecodeLine(raw)
	if err != nil {
		t.Fatal(err)
	}
	if text != want {
		t.Errorf("DecodeLine = %q; queria %q", text, want)
	}
	if strings.Contains(text, "{RAW") {
		t.Errorf("decode ainda emite RAW: %q", text)
	}
	back, err := tc.EncodeLine(text)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, raw) {
		t.Errorf("EncodeLine(%q) = % X; queria % X", text, back, raw)
	}

	simples := []byte{0x85, 0xA8, 0x00, 0x00}
	if got, err := tc.DecodeLine(simples); err != nil || got != "É" {
		t.Fatalf("DecodeLine(85:A8) = %q, %v; queria É", got, err)
	}
	if got, err := tc.EncodeLine("É"); err != nil || !bytes.Equal(got, simples) {
		t.Fatalf("EncodeLine(É) = % X, %v; queria % X", got, err, simples)
	}
}
