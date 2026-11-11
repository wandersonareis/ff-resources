// Package strings implementa o formato Strings (java-like) de exportação/
// importação dos textos de localização. É um formato texto-para-humanos
// que vive ao lado do JSON: os builders montam o mesmo DTO
// (backend/dto) e cada formatter o serializa à sua maneira.
//
// Gramática (v2 — o que o app ESCREVE):
//
//	<entry-block> := <header> <line>+
//	<header>      := "/*key=" <key> " row_count=" <n> " version=2 source=us target=br*/"
//	<line>        := <keyfields> "║" "$" <hash16> " " "=" " " <value>
//	<keyfields>   := [<prefix> ":"]... <id> [":" <name>] ":" <index> ":" <lang>
//	<value>       := <literal> | "$" <hash16>          // def | ref
//
// O par CAT (source/target) não existe mais como campo na linha: cada um
// virou a PRÓPRIA LINHA do seu idioma, e o mapa vivo no header diz qual é
// qual. A linha do idioma FONTE (source=us) carrega o pristine de data/ no
// próprio valor; a linha do idioma ALVO (target=br) carrega a tradução de
// mods/ — o ÚNICO texto que o import aplica. Os demais idiomas da unidade
// continuam saindo como linhas de referência.
//
// Regras da linha alvo: sai sempre que há tradução (target != ""), é SEMPRE
// literal (nunca colapsa em ref "$hash" — tradução repetida não vira ponteiro)
// e acompanha a linha do idioma fonte: com filtro de idiomas que exclui o
// fonte, a tradução não é exportada.
//
// ORDEM DAS LINHAS de uma row: os idiomas de jogo (incluindo o fonte) em ordem
// alfabética e, por último, a linha do idioma alvo. A ordem alfabética ajuda a
// busca visual nos artefatos grandes; a âncora do target no fim separa o texto
// aplicável das referências.
//
// row_count do header conta ROWS, não linhas: uma row pode ter duas linhas
// (fonte + alvo). O parser casa as linhas por index:name, como sempre.
//
// A versão do FORMATO não vai na linha: vive só no header /*key=*/ (a
// versão do JOGO, essa, segue na metadata.key, como nos demais formatos) —
// a linha começa no arquivo (id) em diante. Âncora semântica à direita:
// último campo = lang, penúltimo = index; campos NOVOS (ex.: path/region)
// crescem pela esquerda sem quebrar o parser — versão no primeiro campo é
// erro de formato, nunca campo tolerado.
// "║" liga a chave ao hash de identidade; "=" liga ao valor com split no
// primeiro ("=" dentro do texto passa intacto; a chave nunca tem espaço).
// def: literal cujo xxh64 == hash da chave (verificação grátis na leitura).
// ref: "$hash" resolvido pela tabela hash→texto do próprio arquivo — só a
// linha do idioma FONTE colapsa (dedup de conteúdo do pristine).
// Name é omitido quando vazio (events/help); objects/macro emitem
// <id>:<name>:<index>:<lang>.
//
// v1 (legado — o parser LÊ, ninguém escreve mais): header sem version e sem
// mapa, e o par source/target como campos na linha do idioma default
// ("<valor>║source = <esc>║target = <esc>"). Artefatos antigos continuam
// importando normalmente.
package strings

import (
	"sort"
	"strconv"
	stdstrings "strings"

	"ffresources/internal/families/ffx/profile/common"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/hash"
)

// extensionStrings é a extensão produzida pelo formatter Strings.
const extensionStrings = ".strings"

// formatVersion é a versão do formato que o app ESCREVE. Vivendo no header,
// o parser escolhe a gramática: ausente = 1 (legado), 2 = par por linha.
const formatVersion = 2

// sourceLang/targetLang são os códigos do par CAT no formato v2 — declarados
// no header de cada bloco e usados pelo parser para saber qual linha é o
// pristine (fonte) e qual é a tradução (alvo).
var (
	sourceLang = common.DefaultLocalization
	targetLang = common.DefaultTargetLocalization
)

// keyValueSep liga a chave ao hash de identidade (U+2566).
const keyValueSep = "║"

// Marshal organiza a Collection pronta no formato Strings (todos os idiomas).
// Recebe apenas DTO pronto; não toca no domínio nem muta a entrada.
func Marshal(c dto.Collection) ([]byte, error) {
	return MarshalLangs(c, nil)
}

// MarshalNoDedup serializa SEM o dedup de conteúdo: todo texto sai literal,
// inclusive os que casariam como repetição (não viram ref "$hash"). É a
// direção "dupe/resolve" do menu/CLI: expandir os hashes de conteúdo de
// volta ao texto completo. O campo de identidade (hash da row) continua
// presente, como no Marshal.
func MarshalNoDedup(c dto.Collection) ([]byte, error) {
	return marshalLangsOpt(c, nil, false)
}

// MarshalNoDedupLangs é MarshalNoDedup com o filtro de idiomas
// (nil/vazio = todos).
func MarshalNoDedupLangs(c dto.Collection, langs []string) ([]byte, error) {
	return marshalLangsOpt(c, langs, false)
}

// MarshalLangs organiza a Collection pronta no formato Strings contendo
// só os idiomas pedidos (nil/vazio = todos). Uma linha por (row, idioma)
// não-vazio MAIS a linha do idioma alvo quando há tradução, com chaves e
// rows ordenadas (saída determinística).
func MarshalLangs(c dto.Collection, langs []string) ([]byte, error) {
	return marshalLangsOpt(c, langs, true)
}

// marshalLangsOpt é o motor único do Marshal/MarshalNoDedup: dedup=false
// desliga só o colapso do TEXTO repetido em ref "$hash" — o hash de
// identidade da linha permanece em qualquer modo. A saída é sempre v2.
func marshalLangsOpt(c dto.Collection, langs []string, dedup bool) ([]byte, error) {
	filter := langSet(langs)
	var out stdstrings.Builder
	seen := make(map[string]string) // hashHex bare -> texto (só idioma fonte)
	for _, k := range c.SortedKeys() {
		entry := c[k]
		out.WriteString("/*key=")
		out.WriteString(entry.Metadata.Key)
		out.WriteString(" row_count=")
		out.WriteString(strconv.Itoa(len(entry.Rows)))
		// O par CAT vive no header: version=2 + o mapa fonte/alvo. O parser
		// escolhe a gramática por ele — sem estes campos, o arquivo é v1.
		out.WriteString(" version=")
		out.WriteString(strconv.Itoa(formatVersion))
		out.WriteString(" source=")
		out.WriteString(sourceLang)
		out.WriteString(" target=")
		out.WriteString(targetLang)
		out.WriteString("*/\n")
		rows := make([]dto.TextRow, len(entry.Rows))
		copy(rows, entry.Rows)
		dto.SortRows(rows)
		id := lineID(entry.Metadata, k)
		for _, row := range rows {
			sourceAllowed := filter == nil || filter[sourceLang]
			source := row.Text[sourceLang]
			if source == "" {
				source = row.Source
			}

			// Linhas dos idiomas de jogo em ORDEM ALFABÉTICA: o próprio
			// idioma fonte entra na posição dele (o texto pode vir de
			// row.Source quando o mapa Text não o traz).
			langs := rowLangs(row, filter)
			if sourceAllowed && source != "" {
				langs = append(langs, sourceLang)
				sort.Strings(langs)
			}

			for _, lang := range langs {
				if lang == sourceLang {
					// Linha do idioma FONTE: o pristine de data/ no próprio
					// valor — é o "source" do par. Só esta linha colapsa em
					// ref "$hash" (dedup de conteúdo do pristine).
					h := row.Hash[lang]
					if h == "" {
						h = hash.Sum64Hex(source)
					}
					writeLine(&out, id, row, lang, hash.Prefix(h), dedupValue(dedup, seen, h, source, row))
					continue
				}
				text := row.Text[lang]
				if text == "" {
					continue
				}
				h := row.Hash[lang]
				if h == "" {
					h = hash.Sum64Hex(text)
				}
				writeLine(&out, id, row, lang, hash.Prefix(h), escapeValue(text))
			}

			// Linha do idioma ALVO: a tradução de mods/ — o único texto que
			// o import aplica. Sempre a ÚLTIMA linha da row, sempre literal
			// (nunca ref) e sempre que existe (target != ""), mesmo igual à
			// fonte. Acompanha a linha do fonte: com filtro que exclui o
			// fonte, a tradução não é exportada.
			if sourceAllowed && row.Target != "" {
				th := hash.Sum64Hex(row.Target)
				writeLine(&out, id, row, targetLang, hash.Prefix(th), escapeValue(row.Target))
			}
		}
	}
	return []byte(out.String()), nil
}

// writeLine emite UMA linha do formato:
// <id>[:<name>]:<index>:<lang>║$<hash> = <valor>
// O valor vem pronto (escapado, com ou sem ref) — quem decide o escape é o
// chamador.
func writeLine(out *stdstrings.Builder, id string, row dto.TextRow, lang, hashPrefix, value string) {
	out.WriteString(id)
	if row.Name != "" {
		out.WriteString(":")
		out.WriteString(row.Name)
	}
	out.WriteString(":")
	out.WriteString(strconv.Itoa(row.Index))
	out.WriteString(":")
	out.WriteString(lang)
	out.WriteString(keyValueSep)
	out.WriteString(hashPrefix)
	out.WriteString(" = ")
	out.WriteString(value)
	out.WriteString("\n")
}

// dedupValue decide o valor da linha do idioma fonte: literal na primeira
// ocorrência do texto, ref "$hash" nas repetições (dedup ligado). Row
// divergente (edição só nesta cópia) nunca colapsa.
func dedupValue(dedup bool, seen map[string]string, h, text string, row dto.TextRow) string {
	value := escapeValue(text)
	if !dedup || row.Divergent || !hash.IsDedupEligible(text) {
		return value
	}
	if _, dup := seen[h]; dup {
		return hash.Prefix(h)
	}
	seen[h] = text
	return value
}

// lineID resolve o id da linha a partir da metadata (fallback: chave da
// Collection). A versão do FORMATO não entra na linha — vive só no header
// /*key=*/.
func lineID(m dto.Metadata, mapKey string) string {
	if id := m.ID; id != "" {
		return id
	}
	return mapKey
}

// langSet normaliza o filtro de idiomas (nil = todos).
func langSet(langs []string) map[string]bool {
	if len(langs) == 0 {
		return nil
	}
	set := make(map[string]bool, len(langs))
	for _, l := range langs {
		if l = stdstrings.TrimSpace(l); l != "" {
			set[l] = true
		}
	}
	return set
}

// rowLangs devolve os idiomas de REFERÊNCIA da row com texto, ordenados e
// filtrados. O idioma fonte NÃO entra: o chamador o insere na posição
// alfabética dele e escreve a linha do alvo por último. A tradução do alvo
// não é um idioma de jogo e por isso não vive no mapa Text.
func rowLangs(row dto.TextRow, filter map[string]bool) []string {
	langs := make([]string, 0, len(row.Text))
	for lang, text := range row.Text {
		if lang == sourceLang || text == "" {
			continue
		}
		if filter != nil && !filter[lang] {
			continue
		}
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// escapeValue escapa valor literal para uma linha: backslash, CR, LF e ║.
func escapeValue(s string) string {
	s = stdstrings.ReplaceAll(s, `\`, `\\`)
	s = stdstrings.ReplaceAll(s, "\r", `\r`)
	s = stdstrings.ReplaceAll(s, "\n", `\n`)
	s = stdstrings.ReplaceAll(s, keyValueSep, `\║`)
	return s
}

// unescapeValue desfaz escapeValue (leniente: sequência desconhecida
// mantém backslash + char).
func unescapeValue(s string) string {
	var out stdstrings.Builder
	out.Grow(len(s))
	esc := false
	for _, r := range s {
		if esc {
			switch r {
			case 'n':
				out.WriteRune('\n')
			case 'r':
				out.WriteRune('\r')
			case '\\':
				out.WriteRune('\\')
			case '║':
				out.WriteString(keyValueSep)
			default:
				out.WriteRune('\\')
				out.WriteRune(r)
			}
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		out.WriteRune(r)
	}
	if esc {
		out.WriteRune('\\')
	}
	return out.String()
}
