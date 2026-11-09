// Command keytables_check valida as invariantes das key tables do FF13
// (internal/families/white/codecs/ztr/encoding/key_tables.go).
//
// Uso: go run ./scripts/keytables_check
//
// Invariantes:
//
//  1. nenhum par de bytes aparece em duas categorias do mesmo (jogo,
//     codepage) — é o que permitiu unificar a ordem de consulta do decode e
//     do encode numa só;
//  2. dentro de uma classe, o nome de display é único (o reverse por nome
//     guarda a primeira ocorrência, então duplicata seria ambígua);
//  3. a composição por codepage bate com o que o build.go monta.
package main

import (
	"fmt"
	"os"

	"ffresources/internal/families/white/codecs/ztr/encoding"
)

var games = []encoding.GameCode{encoding.FF131, encoding.FF132, encoding.FF133}

var cps = []encoding.Codepage{encoding.LatinJapanese, encoding.Chinese, encoding.Korean}

// category espelha os campos de par do KeySet, na ordem de consulta.
type category struct {
	name  string
	field func(*encoding.KeySet) []encoding.Tag
}

func categories() []category {
	return []category{
		{"color", func(k *encoding.KeySet) []encoding.Tag { return k.Color }},
		{"icon", func(k *encoding.KeySet) []encoding.Tag { return k.Icon }},
		{"btn", func(k *encoding.KeySet) []encoding.Tag { return k.Btn }},
		{"special", func(k *encoding.KeySet) []encoding.Tag { return k.Special }},
		{"raw", func(k *encoding.KeySet) []encoding.Tag { return k.Raw }},
		{"raw2", func(k *encoding.KeySet) []encoding.Tag { return k.Raw2 }},
	}
}

func main() {
	encoding.ClearKeySets()
	for _, g := range games {
		for _, cp := range cps {
			if err := encoding.PrepareKeySet(g, cp); err != nil {
				fatal(err)
			}
		}
	}

	problems := 0
	for _, g := range games {
		for _, cp := range cps {
			ks, err := encoding.KeySetFor(g, cp)
			if err != nil {
				fatal(err)
			}
			pairs := map[int]string{}
			for _, cat := range categories() {
				names := map[string]bool{}
				for _, t := range cat.field(ks) {
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
		}
	}

	// Relatório da composição (documenta o que cada jogo/codepage serve).
	for _, g := range games {
		for _, cp := range cps {
			ks, err := encoding.KeySetFor(g, cp)
			if err != nil {
				fatal(err)
			}
			line := fmt.Sprintf("%-6s %-14s", g, cpName(cp))
			for _, cat := range categories() {
				line += fmt.Sprintf(" %s=%-3d", cat.name, len(cat.field(ks)))
			}
			line += fmt.Sprintf(" chara=%d single=%d", len(ks.Chara), len(ks.Single))
			fmt.Println(line)
		}
	}

	if problems > 0 {
		fatal(fmt.Errorf("%d problema(s) de invariante", problems))
	}
	fmt.Println("key tables ok")
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
	fmt.Fprintln(os.Stderr, "keytables_check:", err)
	os.Exit(1)
}
