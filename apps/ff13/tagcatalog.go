package main

// CATÁLOGO de tags do FF13 para o autocomplete/editor.
//
// Espelha o formato que o frontend compartilhado consome (`TagCatalog`:
// tag/prefix/hint/values com tag/label/key), com os valores das tabelas do
// codec do IDIOMA pedido — as MESMAS tags que o codec emite e o editor mostra
// como chip.
//
// Só as classes que o tradutor pode inserir entram: CLR (a cor é escolhida
// por nome/swatch, mas a tag leva os bytes do idioma), ICON/BTN (hex + nome)
// e as classes de valor do jogo (VAR/CTR/KEY) e estruturais (PAGE/NL/SPACE).
// `RAW:` e `B:` não entram: são bytes crus de slots desconhecidos.

import (
	"fmt"
	"strings"

	ztenc "ffresources/internal/families/white/codecs/ztr/encoding"
)

// tagSuggestion é um valor completo, pronto para virar chip atômico.
type tagSuggestion struct {
	// Tag é a tag completa e canônica, ex.: {CLR:F9:40:White}.
	Tag string `json:"tag"`
	// Label é o texto exibido no popup, ex.: White / NBSP.
	Label string `json:"label"`
	// Key é a identidade estável sem o texto humano: "F9:40" (pares).
	Key string `json:"key"`
	// Hex é o swatch representativo (só CLR). Nunca vira byte.
	Hex string `json:"hex,omitempty"`
	// Group separa a paleta principal ("") do grupo "ex".
	Group string `json:"group,omitempty"`
	// SemiTransparent marca as cores Ex (semitransparentes no jogo).
	SemiTransparent bool `json:"semiTransparent,omitempty"`
	// Reset marca White (volta à cor natural; não é cor de paleta).
	Reset bool `json:"reset,omitempty"`
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
	Lang    string            `json:"lang"`
	Tags    []tagCatalogEntry `json:"tags"`
}

// GetTagCatalog monta o catálogo de tags da part NO IDIOMA pedido. O idioma
// vazio é o default ("us"); idioma fora do catálogo falha com
// ErrPaletteUnavailable — nunca devolve cores com bytes de outro idioma.
func (FF13Service) GetTagCatalog(part, lang string) (tagCatalog, error) {
	part = normalizePart(part)
	lang = ztenc.NormalizeLanguage(lang)
	if _, ok := ztenc.CodepageForLanguage(lang); !ok {
		return tagCatalog{}, fmt.Errorf("%w para o idioma %s", ztenc.ErrPaletteUnavailable, lang)
	}
	tc, err := textCodecForLang(part, lang)
	if err != nil {
		return tagCatalog{}, err
	}

	entries := []tagCatalogEntry{
		{Tag: "CLR", Prefix: "{CLR:", Hint: "Cor do texto", Values: colorValues(tc)},
		{Tag: "ICON", Prefix: "{ICON:", Hint: "Ícone", Values: pairValues(tc.Icons)},
		{Tag: "BTN", Prefix: "{BTN:", Hint: "Botão do controle", Values: pairValues(tc.Buttons)},
		{Tag: "VAR", Prefix: "{VAR:", Hint: "Variável do jogo (item, nome, número)", Values: specialValues(tc, ztenc.ClassVar, lang)},
		{Tag: "CTR", Prefix: "{CTR:", Hint: "Contador", Values: specialValues(tc, ztenc.ClassCtr, lang)},
		{Tag: "KEY", Prefix: "{KEY:", Hint: "Item-chave", Values: specialValues(tc, ztenc.ClassKey, lang)},
		{Tag: "PAGE", Prefix: "{PAGE:", Hint: "Quebra de página", Values: specialValues(tc, ztenc.ClassPage, lang)},
		{Tag: "NL", Prefix: "{NL:", Hint: "Quebra de linha", Values: specialValues(tc, ztenc.ClassNL, lang)},
		{Tag: "SPACE", Prefix: "{SPACE:", Hint: spaceHint(lang), Values: specialValues(tc, ztenc.ClassSpace, lang)},
	}
	return tagCatalog{Version: part, Lang: lang, Tags: entries}, nil
}

// colorValues: o par de bytes vem do idioma; nome e swatch são display.
func colorValues(tc *ztenc.TextCodec) []tagSuggestion {
	out := make([]tagSuggestion, 0, len(tc.Colors))
	for _, t := range tc.Colors {
		s := tagSuggestion{
			Tag:   t.Text(),
			Label: t.Name,
			Key:   fmt.Sprintf("%02X:%02X", t.B1, t.B2),
			Hex:   t.Swatch,
		}
		// Ex* são semitransparentes e ficam num grupo secundário no popover.
		if strings.HasPrefix(t.Name, "Ex") {
			s.Group = "ex"
			s.SemiTransparent = true
		}
		if strings.EqualFold(t.Name, "White") {
			s.Reset = true
		}
		out = append(out, s)
	}
	return out
}

// pairValues: ICON/BTN — hex autoritativo, nome é display.
func pairValues(list []ztenc.Tag) []tagSuggestion {
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

// specialValues: as classes de par da tabela special (VAR/CTR/KEY/PAGE/NL/SPACE).
func specialValues(tc *ztenc.TextCodec, class ztenc.TagClass, lang string) []tagSuggestion {
	out := make([]tagSuggestion, 0, 4)
	for _, t := range tc.Specials {
		if t.Class != class {
			continue
		}
		key := fmt.Sprintf("%02X:%02X", t.B1, t.B2)
		label := key
		if class == ztenc.ClassSpace {
			label = spaceLabel(lang)
		}
		out = append(out, tagSuggestion{Tag: t.Text(), Label: label, Key: key})
	}
	return out
}

// spaceHint é o texto do autocomplete para o espaço sem quebra. O glifo do
// jogo é U+00A0; no francês ele cumpre o papel do narrow no-break (U+202F).
func spaceHint(lang string) string {
	if ztenc.IsFrench(lang) {
		return "Espaço estreito sem quebra (U+202F)"
	}
	return "Espaço sem quebra (U+00A0)"
}

func spaceLabel(lang string) string {
	if ztenc.IsFrench(lang) {
		return "NNBSP"
	}
	return "NBSP"
}
