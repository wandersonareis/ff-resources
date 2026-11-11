package ztr_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/families/white/codecs/ztr/encoding"
)

// Fixtures validados contra a implementação de referência e os arquivos
// originais do jogo: cada caso tem o .ztr original, o dump "id |:| linha
// crua\r\n", o blob de IDs gravados em debug e o txt oráculo. A regra de
// desempate que reproduz cada original foi medida empiricamente (verificação
// cruzada com o codec Python de referência): os txtres seguem o gerador
// original (TieMinLast); o auto_yus segue o modo -c2 de referência (TieFirst).
//
// O jogo de cada arquivo também é empírico: o src só reproduz seu txt
// oráculo com os dicionários do FF132 ({Icon Objective} em (F0,61) só
// existe no xiii2; (F1,5F) = {Btn DPadLeftRight} como no xiii/xiii2).
var fixtures = []struct {
	dir  string
	game encoding.GameCode
	tie  ztr.TieRule
	txt  string // oráculo principal
}{
	{"auto_yus", encoding.FF132, ztr.TieFirst, "txt.txt"},
	{"v1", encoding.FF131, ztr.TieMinLast, "txt.txt"},
	{"v2", encoding.FF132, ztr.TieMinLast, "txt_ff132.txt"},
	{"src", encoding.FF132, ztr.TieMinLast, "txt.txt"},
}

func testdata(parts ...string) string {
	return filepath.Join(append([]string{"testdata"}, parts...)...)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testdata(name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readZtr(t *testing.T, dir string) *ztr.File {
	t.Helper()
	f, err := ztr.Read(readFixture(t, filepath.Join(dir, "orig.ztr")))
	if err != nil {
		t.Fatalf("Read(%s): %v", dir, err)
	}
	return f
}

// TestReadOracle confere o unpack: o conteúdo lido tem que reproduzir
// byte a byte o _raw.txt (id |:| linha crua) e o debug_IDs-dump de referência.
func TestReadOracle(t *testing.T) {
	for _, fx := range fixtures {
		t.Run(fx.dir, func(t *testing.T) {
			f := readZtr(t, fx.dir)

			var dump bytes.Buffer
			for i := range f.Lines {
				dump.WriteString(f.IDs[i])
				dump.WriteString(" |:| ")
				dump.Write(f.Lines[i])
				dump.WriteString("\r\n")
			}
			if got, want := dump.Bytes(), readFixture(t, filepath.Join(fx.dir, "raw.txt")); !bytes.Equal(got, want) {
				t.Errorf("raw dump diverge: got %d bytes, want %d", len(got), len(want))
			}

			var ids bytes.Buffer
			for _, id := range f.IDs {
				ids.WriteString(id)
				ids.WriteByte(0)
			}
			if got, want := ids.Bytes(), readFixture(t, filepath.Join(fx.dir, "ids.bin")); !bytes.Equal(got, want) {
				t.Errorf("ids divergem: got %d bytes, want %d", len(got), len(want))
			}
		})
	}
}

// TestWriteCompressedOriginal prova o algoritmo de compressão: reempacotar
// o conteúdo lido no modo comprimido, com a regra de desempate correta, tem
// que reproduzir o .ztr original do jogo byte a byte — e a regra oposta tem
// que divergir (atribuição empírica da regra por arquivo).
func TestWriteCompressedOriginal(t *testing.T) {
	for _, fx := range fixtures {
		t.Run(fx.dir, func(t *testing.T) {
			f := readZtr(t, fx.dir)
			want := readFixture(t, filepath.Join(fx.dir, "orig.ztr"))

			out, err := ztr.WriteTie(f, ztr.Compressed, fx.tie)
			if err != nil {
				t.Fatalf("WriteTie: %v", err)
			}
			if !bytes.Equal(out, want) {
				t.Errorf("WriteTie(Compressed, %v) diverge do original: got %d bytes, want %d",
					fx.tie, len(out), len(want))
			}

			other := ztr.TieMinLast
			if fx.tie == ztr.TieMinLast {
				other = ztr.TieFirst
			}
			out2, err := ztr.WriteTie(f, ztr.Compressed, other)
			if err != nil {
				t.Fatalf("WriteTie: %v", err)
			}
			if bytes.Equal(out2, want) {
				t.Errorf("a regra %v TAMÉM reproduz %s — a atribuição de regras está ambígua",
					other, fx.dir)
			}
		})
	}
}

// TestWriteUncompressed confere o modo sem compressão (-c) contra o arquivo
// gerado pela implementação de referência e faz o round-trip pelos dois modos.
func TestWriteUncompressed(t *testing.T) {
	f := readZtr(t, "auto_yus")

	out, err := ztr.Write(f, ztr.Uncompressed)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if want := readFixture(t, filepath.Join("auto_yus", "uncompressed.ztr")); !bytes.Equal(out, want) {
		t.Errorf("Write(Uncompressed) diverge do modo -c de referência: got %d bytes, want %d", len(out), len(want))
	}

	// O modo sem compressão relê para o mesmo conteúdo lógico.
	f2, err := ztr.Read(out)
	if err != nil {
		t.Fatalf("Read(uncompressed): %v", err)
	}
	if len(f2.Lines) != len(f.Lines) || len(f2.IDs) != len(f.IDs) {
		t.Fatalf("round-trip uncompressed: %d/%d vs %d/%d",
			len(f2.Lines), len(f2.IDs), len(f.Lines), len(f.IDs))
	}
	for i := range f.Lines {
		if !bytes.Equal(f2.Lines[i], f.Lines[i]) || f2.IDs[i] != f.IDs[i] {
			t.Fatalf("round-trip uncompressed: linha %d diverge", i)
		}
	}
}

// stripGlyphTags converte um txt no formato antigo (glifo com chaves: "{—}")
// para o formato novo (glifo como texto: "—"), usando as próprias tabelas —
// é a única diferença esperada entre os oráculos e a saída do encoding Go.
// Exceção documentada: o glifo '-' decodifica como U+2010 (hífen tipográfico)
// para não colidir com o ASCII.
func stripGlyphTags(tc *encoding.TextCodec, data []byte) []byte {
	s := strings.ReplaceAll(string(data), "{-}", "\u2010")
	for _, e := range tc.Glyphs {
		s = strings.ReplaceAll(s, "{"+string(e.Rune)+"}", string(e.Rune))
	}
	return []byte(s)
}

func textCodec(t *testing.T, game encoding.GameCode) *encoding.TextCodec {
	t.Helper()
	if err := encoding.PrepareTextCodec(game, encoding.LatinJapanese); err != nil {
		t.Fatal(err)
	}
	tc, err := encoding.TextCodecFor(game, encoding.LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	return tc
}

// TestTextDecodeEncode prova o pipeline completo de texto: decodificar as
// linhas cruas tem que dar o txt do oráculo (formato novo: chara sem
// chaves), re-encodar tem que devolver os bytes crus originais linha a
// linha, e re-empacotar comprimido tem que reproduzir o .ztr original.
func TestTextDecodeEncode(t *testing.T) {
	for _, fx := range fixtures {
		t.Run(fx.dir, func(t *testing.T) {
			tc := textCodec(t, fx.game)
			f := readZtr(t, fx.dir)

			texts, err := tc.DecodeLines(f.Lines)
			if err != nil {
				t.Fatalf("DecodeLines: %v", err)
			}

			want := stripGlyphTags(tc, readFixture(t, filepath.Join(fx.dir, fx.txt)))
			if got := ztr.WriteTxt(f.IDs, texts); !bytes.Equal(got, want) {
				i := 0
				for i < len(got) && i < len(want) && got[i] == want[i] {
					i++
				}
				t.Errorf("txt diverge do oráculo no byte %d: got %q, want %q",
					i, snippet(got, i), snippet(want, i))
				// Atualiza os oráculos com UPDATE_ORACLES=1 (a troca de
				// sintaxe das tags exige regenerar os .txt).
				if os.Getenv("UPDATE_ORACLES") != "" {
					if err := os.WriteFile(testdata(fx.dir, fx.txt), got, 0o644); err != nil {
						t.Fatalf("gravando oráculo: %v", err)
					}
					t.Logf("oráculo atualizado: %s", fx.txt)
				}
			}

			// round-trip por linha: encode(decode(raw)) == raw
			raws, err := tc.EncodeLines(texts)
			if err != nil {
				t.Fatalf("EncodeLines: %v", err)
			}
			for i := range raws {
				if !bytes.Equal(raws[i], f.Lines[i]) {
					t.Errorf("linha %d: encode(decode(raw)) diverge: % x vs % x",
						i, raws[i], f.Lines[i])
				}
			}

			// end-to-end: txt -> encode -> pack -> original
			f2 := &ztr.File{IDs: f.IDs, Lines: raws}
			out, err := ztr.WriteTie(f2, ztr.Compressed, fx.tie)
			if err != nil {
				t.Fatalf("WriteTie: %v", err)
			}
			if want := readFixture(t, filepath.Join(fx.dir, "orig.ztr")); !bytes.Equal(out, want) {
				t.Errorf("reempacotamento end-to-end diverge do original (%d vs %d bytes)",
					len(out), len(want))
			}
		})
	}
}

// TestV2PerGame valida os dicionários por jogo: o MESMO arquivo v2
// decodificado com cada codec tem que produzir o txt oráculo daquele jogo.
func TestV2PerGame(t *testing.T) {
	f := readZtr(t, "v2")
	perGame := map[encoding.GameCode]string{
		encoding.FF131: "txt_ff131.txt",
		encoding.FF132: "txt_ff132.txt",
		encoding.FF133: "txt_ff133.txt",
	}
	for game, txtName := range perGame {
		tc := textCodec(t, game)
		texts, err := tc.DecodeLines(f.Lines)
		if err != nil {
			t.Fatalf("%s: DecodeLines: %v", game, err)
		}
		want := stripGlyphTags(tc, readFixture(t, filepath.Join("v2", txtName)))
		if got := ztr.WriteTxt(f.IDs, texts); !bytes.Equal(got, want) {
			t.Errorf("%s: txt diverge do oráculo (%d vs %d bytes)", game, len(got), len(want))
			if os.Getenv("UPDATE_ORACLES") != "" {
				if err := os.WriteFile(testdata("v2", txtName), got, 0o644); err != nil {
					t.Fatalf("gravando oráculo: %v", err)
				}
				t.Logf("oráculo atualizado: %s", txtName)
			}
		}
	}
}

// TestTxtRoundTrip cobre o formatter txt (ReadTxt/WriteTxt).
func TestTxtRoundTrip(t *testing.T) {
	ids := []string{"id_a", "id_b", "id_c"}
	texts := []string{"texto um", "{Color White}— com chara", "três"}
	data := ztr.WriteTxt(ids, texts)
	ids2, texts2, err := ztr.ReadTxt(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ids {
		if ids[i] != ids2[i] || texts[i] != texts2[i] {
			t.Errorf("round-trip %d: (%q,%q) != (%q,%q)", i, ids[i], texts[i], ids2[i], texts2[i])
		}
	}
}

func snippet(b []byte, at int) string {
	lo := at - 20
	if lo < 0 {
		lo = 0
	}
	hi := at + 20
	if hi > len(b) {
		hi = len(b)
	}
	return string(b[lo:hi])
}
