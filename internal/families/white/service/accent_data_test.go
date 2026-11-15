package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ffresources/internal/families/white/codecs/ztr"
	ztenc "ffresources/internal/families/white/codecs/ztr/encoding"
)

// accentTrails são os pares 0x85:XX que os binários de data/ usam para os
// acentuados latinos (bloco canônico U+00C0..U+00FF + ¿).
var accentTrails = map[byte]rune{
	0x80: '¿',
	0xA0: 'Á',
	0xA1: 'Â',
	0xA2: 'Ã',
	0xA3: 'Ä',
	0xA6: 'Ç',
	0xA7: 'È',
	0xA8: 'É',
	0xA9: 'Ê',
	0xAC: 'Í',
	0xB2: 'Ó',
	0xB3: 'Ô',
	0xB4: 'Õ',
	0xB5: 'Ö',
	0xB9: 'Ú',
	0xBA: 'Û',
	0xBB: 'Ü',
}

// scanAccentedZtr percorre uma árvore de .ztr e devolve:
//   - seen: trails de acento efetivamente presentes nos bytes;
//   - raw: amostras de linhas que ainda decodificam para {RAW:85:XX}.
func scanAccentedZtr(t *testing.T, root, part string) (seen map[byte]bool, raw []string) {
	t.Helper()
	seen = map[byte]bool{}
	codecs := map[ztenc.Codepage]*ztenc.TextCodec{}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".ztr") {
			return nil
		}
		rawBytes, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		zf, zerr := ztr.Read(rawBytes)
		if zerr != nil {
			return nil
		}

		base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		lang := base
		if i := strings.LastIndexByte(base, '_'); i >= 0 {
			lang = base[i+1:]
		}
		cp := CodepageOf(lang)
		tc := codecs[cp]
		if tc == nil {
			var cerr error
			tc, cerr = TextCodecFor(part, cp)
			if cerr != nil {
				t.Fatalf("%s: TextCodecFor(%d): %v", path, cp, cerr)
			}
			codecs[cp] = tc
		}

		for i, line := range zf.Lines {
			if !bytes.Contains(line, []byte{0x85}) {
				continue
			}
			for trail := range accentTrails {
				if bytes.Contains(line, []byte{0x85, trail}) {
					seen[trail] = true
				}
			}
			text, derr := tc.DecodeLine(line)
			if derr != nil {
				t.Errorf("%s linha %d: DecodeLine: %v", path, i, derr)
				continue
			}
			for trail, want := range accentTrails {
				tag := fmt.Sprintf("{RAW:85:%02X}", trail)
				if strings.Contains(text, tag) {
					id := ""
					if i < len(zf.IDs) {
						id = zf.IDs[i]
					}
					raw = append(raw, fmt.Sprintf("%s:%s: %s (queria %q)",
						filepath.ToSlash(path), id, tag, string(want)))
					if len(raw) >= 20 {
						return filepath.SkipAll
					}
				}
			}
		}
		return nil
	})
	return seen, raw
}

// TestDataAccentsDecodeWithoutRaw é a trava principal: nos binários de data/
// nenhum acentuado pode continuar saindo como {RAW:85:XX}.
func TestDataAccentsDecodeWithoutRaw(t *testing.T) {
	allSeen := map[byte]bool{}
	for _, part := range []string{"part1", "part2", "part3"} {
		data := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", part)
		if _, err := os.Stat(data); err != nil {
			t.Logf("%s ausente; pulando: %v", data, err)
			continue
		}
		seen, raw := scanAccentedZtr(t, data, part)
		for trail := range seen {
			allSeen[trail] = true
		}
		for trail, want := range accentTrails {
			if !seen[trail] {
				continue
			}
			tc, err := TextCodecFor(part, ztenc.LatinJapanese)
			if err != nil {
				t.Fatal(err)
			}
			text, err := tc.DecodeLine([]byte{0x85, trail, 0x00, 0x00})
			if err != nil {
				t.Fatalf("%s: DecodeLine(85:%02X): %v", part, trail, err)
			}
			if text != string(want) {
				t.Errorf("%s: DecodeLine(85:%02X) = %q; queria %q", part, trail, text, want)
			}
		}
		if len(raw) > 0 {
			t.Errorf("%s: %d amostra(s) de acentuado ainda em RAW:", part, len(raw))
			for _, s := range raw {
				t.Errorf("  %s", s)
			}
		}
		t.Logf("%s: pares de acento encontrados = %d/%d", part, len(seen), len(accentTrails))
	}
	t.Logf("pares de acento encontrados em data/: %d/%d", len(allSeen), len(accentTrails))
}

// TestDataAccentExample fixa a linha do relatório no arquivo real de data/.
func TestDataAccentExample(t *testing.T) {
	data := partDataRoot(t, "part1")
	path := filepath.Join(data, "zone", "z105", "txtres", "ac", "ac_eark_001", "txtres_sp.ztr")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("arquivo do exemplo ausente: %v", err)
	}
	zf, err := ztr.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := TextCodecFor("part1", ztenc.LatinJapanese)
	if err != nil {
		t.Fatal(err)
	}
	const id = "$aeark001_070sz"
	const want = "Genial, ¿y ahora?"
	for i, key := range zf.IDs {
		if key != id {
			continue
		}
		text, derr := tc.DecodeLine(zf.Lines[i])
		if derr != nil {
			t.Fatal(derr)
		}
		if text != want {
			t.Errorf("%s = %q; queria %q", id, text, want)
		}
		if strings.Contains(text, "{RAW") {
			t.Errorf("%s ainda contém RAW: %q", id, text)
		}
		return
	}
	t.Fatalf("ID %s não encontrado em %s", id, path)
}

// TestModsAccentsResidualRaw é DIAGNÓSTICO: a tradução de mods/ foi gerada
// por outra ferramenta; se algum acentuado ainda decodificar como RAW,
// avisamos sem falhar o teste.
func TestModsAccentsResidualRaw(t *testing.T) {
	for _, part := range []string{"part1", "part2", "part3"} {
		mods := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods", part)
		if _, err := os.Stat(mods); err != nil {
			t.Logf("%s ausente; pulando: %v", mods, err)
			continue
		}
		_, raw := scanAccentedZtr(t, mods, part)
		if len(raw) == 0 {
			t.Logf("mods/%s: nenhum acentuado residual em RAW", part)
			continue
		}
		t.Logf("mods/%s: %d amostra(s) de acentuado ainda em RAW (não corrigidas):", part, len(raw))
		for _, s := range raw {
			t.Logf("  %s", s)
		}
	}
}
