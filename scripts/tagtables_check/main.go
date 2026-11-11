// Command tagtables_check valida as invariantes das tabelas de tags do FF13
// (internal/families/white/codecs/ztr/encoding/tags).
//
// Uso: go run ./scripts/tagtables_check
//
// Invariantes:
//
//  1. nenhum par de bytes aparece em duas categorias do mesmo (jogo,
//     codepage) — é o que permite uma ordem de consulta única;
//  2. dentro de uma classe, o nome de display é único;
//  3. a paleta existe e é própria de cada idioma (os bytes mudam entre
//     LJ/Ch/Kr);
//  4. SPACE existe no lugar do antigo TAB e mantém o par 85:60.
package main

import (
	"fmt"
	"os"

	"ffresources/internal/families/white/codecs/ztr/encoding"
)

var games = []encoding.GameCode{encoding.FF131, encoding.FF132, encoding.FF133}

var cps = []encoding.Codepage{encoding.LatinJapanese, encoding.Chinese, encoding.Korean}

// category espelha os campos de par do TextCodec, na ordem de consulta.
type category struct {
	name  string
	field func(*encoding.TextCodec) []encoding.Tag
}

func categories() []category {
	return []category{
		{"colors", func(c *encoding.TextCodec) []encoding.Tag { return c.Colors }},
		{"icons", func(c *encoding.TextCodec) []encoding.Tag { return c.Icons }},
		{"buttons", func(c *encoding.TextCodec) []encoding.Tag { return c.Buttons }},
		{"specials", func(c *encoding.TextCodec) []encoding.Tag { return c.Specials }},
		{"raw", func(c *encoding.TextCodec) []encoding.Tag { return c.Raw }},
		{"raw_extra", func(c *encoding.TextCodec) []encoding.Tag { return c.RawExtra }},
	}
}

func main() {
	encoding.ClearTextCodecs()
	for _, g := range games {
		for _, cp := range cps {
			if err := encoding.PrepareTextCodec(g, cp); err != nil {
				fatal(err)
			}
		}
	}

	problems := 0
	for _, g := range games {
		for _, cp := range cps {
			tc, err := encoding.TextCodecFor(g, cp)
			if err != nil {
				fatal(err)
			}
			pairs := map[int]string{}
			for _, cat := range categories() {
				names := map[string]bool{}
				for _, t := range cat.field(tc) {
					key := int(t.B1)<<8 | int(t.B2)
					if prev, dup := pairs[key]; dup {
						fmt.Fprintf(os.Stderr, "%s/%s: par %s em %s e %s\n",
							g, cpName(cp), t.Text(), prev, cat.name)
						problems++
						continue
					}
					pairs[key] = cat.name
					if t.Name == "" {
						continue
					}
					if names[t.Name] {
						fmt.Fprintf(os.Stderr, "%s/%s: nome %q repetido em %s\n",
							g, cpName(cp), t.Name, cat.name)
						problems++
					}
					names[t.Name] = true
				}
			}

			// A paleta é fechada e própria do idioma.
			if len(tc.Colors) == 0 {
				fmt.Fprintf(os.Stderr, "%s/%s: paleta de cores vazia\n", g, cpName(cp))
				problems++
			}
			space := 0
			for _, t := range tc.Specials {
				if t.Class == encoding.ClassSpace {
					space++
					if t.Text() != "{SPACE:85:60}" {
						fmt.Fprintf(os.Stderr, "%s/%s: SPACE = %s; queria {SPACE:85:60}\n",
							g, cpName(cp), t.Text())
						problems++
					}
				}
			}
			if space != 1 {
				fmt.Fprintf(os.Stderr, "%s/%s: %d tags SPACE; queria 1\n", g, cpName(cp), space)
				problems++
			}
		}

		// O MESMO nome de cor tem bytes diferentes por codepage.
		white := map[encoding.Codepage][2]byte{}
		for _, cp := range cps {
			tc, err := encoding.TextCodecFor(g, cp)
			if err != nil {
				fatal(err)
			}
			for _, t := range tc.Colors {
				if t.Name == "White" {
					white[cp] = [2]byte{t.B1, t.B2}
				}
			}
		}
		if white[encoding.LatinJapanese][0] == white[encoding.Chinese][0] &&
			white[encoding.LatinJapanese][0] == white[encoding.Korean][0] {
			fmt.Fprintf(os.Stderr, "%s: White não muda de bytes entre codepages (%v)\n", g, white)
			problems++
		}
	}

	// Relatório da composição.
	for _, g := range games {
		for _, cp := range cps {
			tc, err := encoding.TextCodecFor(g, cp)
			if err != nil {
				fatal(err)
			}
			line := fmt.Sprintf("%-6s %-14s", g, cpName(cp))
			for _, cat := range categories() {
				line += fmt.Sprintf(" %s=%-3d", cat.name, len(cat.field(tc)))
			}
			line += fmt.Sprintf(" glyphs=%d controls=%d", len(tc.Glyphs), len(tc.Controls))
			fmt.Println(line)
		}
	}

	if problems > 0 {
		fatal(fmt.Errorf("%d problema(s) de invariante", problems))
	}
	fmt.Println("tag tables ok")
}

func cpName(cp encoding.Codepage) string {
	switch cp {
	case encoding.LatinJapanese:
		return "LatinJapanese"
	case encoding.Chinese:
		return "Chinese"
	case encoding.Korean:
		return "Korean"
	}
	return "?"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "tagtables_check:", err)
	os.Exit(1)
}
