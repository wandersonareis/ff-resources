package formatters

import (
	"strings"

	"ffresources/internal/shared/dto"
)

// Options são os parâmetros de UMA chamada de serialização/importação. O
// shared não conhece nenhum jogo: a família diz quais idiomas existem, qual é
// o slot fonte/alvo, qual é a identidade da row e como validar o destino.
type Options struct {
	// SourceLang é o idioma cujo texto é a fonte (e o slot deduplicado).
	SourceLang string
	// TargetLang é o rótulo do slot de tradução no formato (ex.: FFX "br").
	TargetLang string
	// TargetLanguageName é o nome humano do alvo em metadados (ex.: "pt-BR").
	TargetLanguageName string
	// Languages filtra os idiomas exportados (nil/vazio = todos os da row).
	Languages []string
	// Identity diz como casar a row no import (Index ou Key).
	Identity dto.Identity
	// Ordered indica que a ordem de Index é semântica (FFX). Falso = ordem
	// por chave (FF13), que também é a saída determinística.
	Ordered bool
	// DisableDedup desliga o colapso de texto repetido em "$hash". Default
	// (false) = dedup ligado.
	DisableDedup bool
	// ReservedKeyPrefixes são prefixos que NÃO podem abrir a chave da row no
	// formato .strings (ex.: versões do FFX). O shared não conhece a lista;
	// quem chama fornece.
	ReservedKeyPrefixes []string
	// CheckWrite valida o destino antes da gravação (ex.: trava do FFX).
	// Nil = sem validação.
	CheckWrite func(path string) error
}

// DefaultOptions mantém o comportamento dos helpers antigos: fonte "us",
// alvo "br"/pt-BR (rótulo do slot de tradução no artefato), identidade
// posicional e dedup ligado. As famílias passam o perfil explícito; estes
// defaults existem para os testes/CLI legados.
func DefaultOptions() Options {
	return Options{
		SourceLang:          "us",
		TargetLang:          "br",
		TargetLanguageName:  "pt-BR",
		Identity:            dto.IdentityIndex,
		Ordered:             true,
		ReservedKeyPrefixes: []string{"ffx", "ffx2", "lastmiss", "eternalcalm"},
	}
}

// Source devolve o idioma fonte normalizado.
func (o Options) Source() string {
	if s := strings.TrimSpace(o.SourceLang); s != "" {
		return s
	}
	return "us"
}

// Target devolve o rótulo do slot alvo (default: o próprio fonte).
func (o Options) Target() string {
	if t := strings.TrimSpace(o.TargetLang); t != "" {
		return t
	}
	return o.Source()
}

// TargetLanguage devolve o nome humano do alvo (default: o código).
func (o Options) TargetLanguage() string {
	if n := strings.TrimSpace(o.TargetLanguageName); n != "" {
		return n
	}
	return o.Target()
}

// DedupEnabled informa se o colapso de repetições está ligado.
func (o Options) DedupEnabled() bool { return !o.DisableDedup }

// Includes diz se o idioma entra no filtro (nil/vazio = todos).
func (o Options) Includes(lang string) bool {
	if len(o.Languages) == 0 {
		return true
	}
	for _, l := range o.Languages {
		if strings.TrimSpace(l) == lang {
			return true
		}
	}
	return false
}

// IdentityOf devolve a identidade configurada (default posicional).
func (o Options) IdentityOf() dto.Identity {
	if o.Identity == dto.IdentityKey {
		return dto.IdentityKey
	}
	return dto.IdentityIndex
}

// SortedRows devolve uma cópia ordenada conforme a identidade/ordem.
func (o Options) SortedRows(rows []dto.TextRow) []dto.TextRow {
	out := make([]dto.TextRow, len(rows))
	copy(out, rows)
	if o.IdentityOf() == dto.IdentityKey || !o.Ordered {
		dto.SortRowsBy(out, dto.IdentityKey)
		return out
	}
	dto.SortRowsBy(out, dto.IdentityIndex)
	return out
}

// RowIdentityName devolve o valor da coluna "name" conforme a identidade:
// chave estável (Key) ou qualificador de segmento (Name).
func RowIdentityName(r dto.TextRow, opts Options) string {
	if opts.IdentityOf() == dto.IdentityKey && r.Key != "" {
		return r.Key
	}
	return r.Name
}

// RowMatches diz se a row existente é a mesma, conforme a identidade.
func RowMatches(r dto.TextRow, index int, name string, opts Options) bool {
	if opts.IdentityOf() == dto.IdentityKey {
		return r.KeyFor(dto.IdentityKey) == name || r.Name == name
	}
	return r.Index == index && r.Name == name
}
