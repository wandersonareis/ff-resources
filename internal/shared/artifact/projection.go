package formatters

import (
	"ffresources/internal/shared/dto"
)

// SourceTarget devolve o par source/target de uma linha (row, idioma) para os
// formatos tabulares/.strings: só o idioma fonte carrega o par — é ali que
// vive a tradução em mods/. O `target` é o ÚNICO texto que o import aplica de
// volta no binário.
func SourceTarget(r dto.TextRow, lang string, opts Options) (source, target string) {
	if lang == opts.Source() {
		return r.Source, r.Target
	}
	return "", ""
}

// HasSourceTarget informa se a row pede a linha do idioma fonte mesmo com o
// texto original vazio: tradução em mods/ sem referência (ex.: original "-")
// não pode sumir do artefato.
func HasSourceTarget(r dto.TextRow, lang string, opts Options) bool {
	return lang == opts.Source() && (r.Source != "" || r.Target != "")
}
