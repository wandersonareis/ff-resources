package main

// CATÁLOGO de tags do FF13 para o autocomplete do editor.
//
// Espelha o formato que o frontend compartilhado consome (`TagCatalog`:
// tag/prefix/hint/values com tag/label/key), com os valores das tabelas do
// keyset da part — as MESMAS tags que o codec emite e o editor mostra como
// chip.
//
// Só as classes que o tradutor pode inserir entram: COLOR (a cor é escolhida
// por nome), ICON/BTN (hex + nome) e as classe de valor do jogo
// (VAR/CTR/KEY) e estruturais (PAGE/NL/TAB). `RAW:` e `B:` não entram: são
// bytes crus de slots desconhecidos — ninguém deve inseri-los à mão.

import (
	"fmt"
	"strings"

	"ffresources/apps/ff13/parts"
	ztenc "ffresources/internal/families/white/codecs/ztr/encoding"
)

// tagSuggestion é um valor completo, pronto para virar chip atômico.
type tagSuggestion struct {
	// Tag é a tag completa e canônica, ex.: {ICON:F0:40:Clock}.
	Tag string `json:"tag"`
	// Label é o texto exibido no popup, ex.: Clock.
	Label string `json:"label"`
	// Key é a identidade estável sem o texto humano: "F0:40" (pares) ou o
	// nome da cor.
	Key string `json:"key"`
}

// tagCatalogEntry é uma classe de tag: o prefixo do estágio 1 e os valores
// do estágio 2.
type tagCatalogEntry struct {
	Tag    string          `json:"tag"`
	Prefix string          `json:"prefix"`
	Hint   string          `json:"hint"`
	Values []tagSuggestion `json:"values"`
}

// tagCatalog é o catálogo servido ao frontend via binding GetTagCatalog.
type tagCatalog struct {
	Version string            `json:"version"`
	Tags    []tagCatalogEntry `json:"tags"`
}

// GetTagCatalog monta o catálogo de tags da part. Usa o keyset LatinJapanese
// (o dos arquivos "_us", que são os editáveis) — as demais codepages têm os
// mesmos nomes em bytes diferentes.
func (FF13Service) GetTagCatalog(part string) (tagCatalog, error) {
	part = normalizePart(part)
	ks, err := keySet(part)
	if err != nil {
		return tagCatalog{}, err
	}
	p := parts.Get(part)

	entries := []tagCatalogEntry{
		{Tag: "COLOR", Prefix: "{COLOR:", Hint: "Cor do texto", Values: colorValues(ks)},
		{Tag: "ICON", Prefix: "{ICON:", Hint: "Ícone", Values: pairValues(ks.Icon, p)},
		{Tag: "BTN", Prefix: "{BTN:", Hint: "Botão do controle", Values: pairValues(ks.Btn, p)},
		{Tag: "VAR", Prefix: "{VAR:", Hint: "Variável do jogo (item, nome, número)", Values: specialValues(ks, ztenc.ClassVar)},
		{Tag: "CTR", Prefix: "{CTR:", Hint: "Contador", Values: specialValues(ks, ztenc.ClassCtr)},
		{Tag: "KEY", Prefix: "{KEY:", Hint: "Item-chave", Values: specialValues(ks, ztenc.ClassKey)},
		{Tag: "PAGE", Prefix: "{PAGE:", Hint: "Quebra de página", Values: specialValues(ks, ztenc.ClassPage)},
		{Tag: "NL", Prefix: "{NL:", Hint: "Quebra de linha", Values: specialValues(ks, ztenc.ClassNL)},
		{Tag: "TAB", Prefix: "{TAB:", Hint: "Tabulação", Values: specialValues(ks, ztenc.ClassTab)},
	}
	return tagCatalog{Version: part, Tags: entries}, nil
}

// colorValues: a cor é escolhida por NOME (o byte sai da codepage).
func colorValues(ks *ztenc.KeySet) []tagSuggestion {
	out := make([]tagSuggestion, 0, len(ks.Color))
	for _, t := range ks.Color {
		out = append(out, tagSuggestion{Tag: t.Text(), Label: t.Name, Key: t.Name})
	}
	return out
}

// pairValues: ICON/BTN — hex autoritativo, nome é display.
func pairValues(list []ztenc.Tag, p parts.Part) []tagSuggestion {
	out := make([]tagSuggestion, 0, len(list))
	for _, t := range list {
		key := fmt.Sprintf("%02X:%02X", t.B1, t.B2)
		label := t.Name
		if label == "" {
			label = key
		}
		out = append(out, tagSuggestion{Tag: t.Text(), Label: label, Key: key})
	}
	return out
}

// specialValues: as classes de par da tabela special (VAR/CTR/KEY/PAGE/NL/TAB).
func specialValues(ks *ztenc.KeySet, class ztenc.TagClass) []tagSuggestion {
	out := make([]tagSuggestion, 0, 4)
	for _, t := range ks.Special {
		if t.Class != class {
			continue
		}
		key := fmt.Sprintf("%02X:%02X", t.B1, t.B2)
		out = append(out, tagSuggestion{Tag: t.Text(), Label: key, Key: key})
	}
	return out
}

// tagCatalogHint é o texto do estágio 1 quando o catálogo não carregou.
func tagCatalogHint(tag string) string {
	if i := strings.IndexByte(tag, ':'); i >= 0 {
		return tag[:i]
	}
	return tag
}
