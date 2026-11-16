package strings

import (
	"encoding/hex"
	"fmt"
	"strconv"
	stdstrings "strings"

	formatters "ffresources/internal/shared/artifact"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/hash"
	logging "ffresources/internal/shared/logging"
)

// parsedLine é uma linha de conteúdo decomposta.
type parsedLine struct {
	id     string
	name   string
	index  int
	lang   string
	hash   string // hex bare
	ref    bool
	value  string // literal (com escapes desfeitos) ou "" quando ref
	source string // referência us de data/ (v1: campo da linha do idioma default)
	target string // tradução de mods/ — o texto que o import aplica
}

// headerInfo é o cabeçalho de bloco /*key=… row_count=… version=… source=…
// target=…*/. A versão escolhe a gramática da linha: ausente (ou 1) = v1,
// com o par source/target como CAMPOS na linha do idioma default; 2 = v2,
// com o par como LINHAS próprias do idioma fonte/alvo do mapa.
type headerInfo struct {
	key      string
	rowCount int
	version  int
	source   string // idioma fonte do mapa (v2)
	target   string // idioma alvo do mapa (v2)
}

// Unmarshal parseia o formato Strings de volta para a Collection (DTO),
// com rows ordenadas por Index para reconstrução posicional do binário.
//
// O header (key/row_count) é informativo: a key ancora a metadata da
// entrada e row_count é conferido com LogVerbose (o import reconta).
// Como no JSON, o hash é ponteiro opaco no import: refs "$hash" são
// resolvidas pela tabela hash→texto do próprio arquivo, e defs com
// xxh64(literal) != hash da chave geram só LogVerbose (texto vence).
// Literal diferente sob o mesmo hash é marcado Divergent: um editor externo
// pode ter trocado o conteúdo $hash de uma cópia por texto válido.
//
// Os DOIS formatos são lidos aqui, escolhidos pelo header de cada bloco:
// v1 (artefatos antigos, sem version) mantém o par source/target como
// campos na linha do idioma default; v2 (o que o app escreve) tem uma
// linha própria para o idioma alvo — ela vira row.Target, o ÚNICO texto
// que o import aplica (a linha do idioma fonte é referência pristine).
func Unmarshal(data []byte) (dto.Collection, error) {
	return unmarshalOptions(data, formatters.DefaultOptions())
}

// unmarshalOptions parseia respeitando o idioma fonte, a identidade e os
// prefixos reservados do documento.
func unmarshalOptions(data []byte, opts formatters.Options) (dto.Collection, error) {
	entries := make(map[string]*dto.FileEntry)
	rows := make(map[string]map[string]*dto.TextRow) // entryID -> "index\x00name" -> row
	type pendingRef struct {
		entryID string
		rkey    string
		lang    string
		hash    string
		target  bool // linha do idioma alvo: resolve em Target, não em Text
		source  bool // linha do idioma fonte (v2): o texto resolvido também é o Source
	}
	var refs []pendingRef
	table := make(map[string]string) // bare hash -> texto (defs do arquivo)
	blockKey := ""
	block := headerInfo{}
	expectedRows := make(map[string]int) // metadata.key -> row_count do header
	lineno := 0

	for _, raw := range stdstrings.Split(string(data), "\n") {
		lineno++
		line := stdstrings.TrimRight(raw, "\r")
		if stdstrings.TrimSpace(line) == "" {
			continue
		}
		h, ok, herr := parseHeader(line)
		if herr != nil {
			return nil, herr
		}
		if ok {
			blockKey = h.key
			block = h
			expectedRows[h.key] = h.rowCount
			continue
		}
		pl, err := parseLine(line, lineno, opts)
		if err != nil {
			return nil, err
		}
		entry := entries[pl.id]
		if entry == nil {
			entry = &dto.FileEntry{Metadata: dto.Metadata{Key: blockKey, ID: pl.id}}
			entries[pl.id] = entry
			rows[pl.id] = make(map[string]*dto.TextRow)
		} else if entry.Metadata.Key == "" && blockKey != "" {
			entry.Metadata.Key = blockKey
		}
		rkey := rowIdentityKey(pl, opts)
		row := rows[pl.id][rkey]
		if row == nil {
			newRow := dto.TextRow{Index: pl.index, Name: pl.name}
			if opts.IdentityOf() == dto.IdentityKey {
				newRow.Key = pl.name
			}
			entry.Rows = append(entry.Rows, newRow)
			row = &entry.Rows[len(entry.Rows)-1]
			rows[pl.id][rkey] = row
		}
		if row.Hash == nil {
			row.Hash = make(map[string]string)
		}
		if row.Text == nil {
			row.Text = make(map[string]string)
		}
		row.Hash[pl.lang] = pl.hash
		// v2: a linha do idioma ALVO é a tradução (Target) — o único texto
		// que o import aplica. Nunca entra em Text, que continua só com
		// idiomas de jogo. O mapa do header diz qual idioma é o alvo.
		if block.version >= formatVersion && pl.lang == block.target {
			if pl.ref {
				refs = append(refs, pendingRef{entryID: pl.id, rkey: rkey, lang: pl.lang, hash: pl.hash, target: true})
				continue
			}
			registerDef(table, pl.hash, pl.value)
			row.Target = pl.value
			continue
		}
		// v1: o par source/target vive como CAMPOS na linha do idioma
		// default (e vale inclusive para refs — o valor pode ser "$hash" e
		// a tradução continuar explícita no target).
		if block.version < formatVersion && pl.lang == opts.Source() {
			row.Source = pl.source
			row.Target = pl.target
		}
		// Linha do idioma-fonte: no v2 é a do mapa do header (o valor dela É
		// o source); no v1 é a do idioma default, onde o par vivia — é nela
		// que o dedup/divergente do pristine se apoia.
		isSourceLine := block.version >= formatVersion && pl.lang == block.source
		isPairLine := isSourceLine || (block.version < formatVersion && pl.lang == opts.Source())
		if pl.ref {
			refs = append(refs, pendingRef{
				entryID: pl.id, rkey: rkey, lang: pl.lang, hash: pl.hash, source: isSourceLine,
			})
			continue
		}
		if pl.hash != "" {
			if got := hash.Sum64Hex(pl.value); got != pl.hash {
				logging.Verbose("strings hash mismatch for %s[%d] lang %s: file %s vs text %s",
					pl.id, pl.index, pl.lang, pl.hash, got)
			}
			if pl.value != "" {
				if canonical, dup := table[pl.hash]; !dup {
					table[pl.hash] = pl.value
				} else if isPairLine && canonical != pl.value {
					// Literal diferente sob o mesmo hash: provavelmente um
					// editor externo substituiu $hash numa cópia. Mantê-la
					// separada do texto canônico ao aplicar o import.
					row.Divergent = true
				}
			}
		}
		row.Text[pl.lang] = pl.value
		// v2: a linha do idioma FONTE carrega o source no próprio valor —
		// preencher Source mantém a paridade com o import do v1 e com quem
		// relê o artefato pelo registry.
		if isSourceLine {
			row.Source = pl.value
		}
	}
	// Segunda passada: resolve refs pela tabela do próprio arquivo.
	for _, r := range refs {
		row := rows[r.entryID][r.rkey]
		if row == nil {
			continue
		}
		v, ok := table[r.hash]
		if !ok {
			logging.Verbose("strings orphan dedup reference $%s kept as-is", r.hash)
			v = hash.Prefix(r.hash)
		}
		if r.target {
			row.Target = v
			continue
		}
		row.Text[r.lang] = v
		if r.source {
			row.Source = v
		}
	}
	out := make(dto.Collection, len(entries))
	for id, entry := range entries {
		dto.SortRowsBy(entry.Rows, opts.IdentityOf())
		entry.Metadata = entry.Metadata.WithRowCount(len(entry.Rows))
		if want, ok := expectedRows[entry.Metadata.Key]; ok && len(entry.Rows) != want {
			logging.Verbose("strings row_count mismatch for %s: header %d vs %d rows",
				entry.Metadata.Key, want, len(entry.Rows))
		}
		out[id] = *entry
	}
	return out, nil
}

// registerDef grava um literal na tabela hash→texto (a mesa das refs). Sem
// efeito quando o hash já tem def — a primeira ocorrência é a canônica.
func registerDef(table map[string]string, h, value string) {
	if value == "" {
		return
	}
	if _, dup := table[h]; !dup {
		table[h] = value
	}
}

// parseHeader lê /*key=<key> row_count=<n> [version=<v> source=<lang> target=<lang>]*/.
// Campos novos crescem pela direita e são opcionais: sem version/mapa o
// bloco é v1 (gramática legada, par como campo na linha). version
// desconhecida — ou um bloco v2 sem o mapa source/target — é ERRO: parsear
// errado em silêncio é pior que falhar.
func parseHeader(line string) (h headerInfo, ok bool, err error) {
	t := stdstrings.TrimSpace(line)
	if !stdstrings.HasPrefix(t, "/*key=") || !stdstrings.HasSuffix(t, "*/") {
		return headerInfo{}, false, nil
	}
	inner := stdstrings.TrimSuffix(stdstrings.TrimPrefix(t, "/*key="), "*/")
	idx := stdstrings.LastIndex(inner, " row_count=")
	if idx < 0 {
		return headerInfo{}, false, nil
	}
	key := stdstrings.TrimSpace(inner[:idx])
	if key == "" {
		return headerInfo{}, false, nil
	}
	// Depois do row_count vêm os pares chave=valor do formato (version,
	// source, target). O row_count é o primeiro token; o resto são campos.
	rest := stdstrings.TrimSpace(inner[idx+len(" row_count="):])
	head, tail := rest, ""
	if sp := stdstrings.IndexByte(rest, ' '); sp >= 0 {
		head, tail = rest[:sp], rest[sp+1:]
	}
	n, aerr := strconv.Atoi(head)
	if aerr != nil {
		return headerInfo{}, false, nil
	}
	out := headerInfo{key: key, rowCount: n, version: 1}
	for _, field := range stdstrings.Fields(tail) {
		name, value, found := stdstrings.Cut(field, "=")
		if !found {
			return headerInfo{}, false, nil
		}
		switch name {
		case "version":
			v, verr := strconv.Atoi(value)
			if verr != nil || v < 1 {
				return headerInfo{}, false, nil
			}
			if v > formatVersion {
				return headerInfo{}, false, fmt.Errorf("strings: versão de formato %d não suportada (máximo %d)", v, formatVersion)
			}
			out.version = v
		case "source":
			out.source = value
		case "target":
			out.target = value
			// Campo futuro no header é ignorado (cresce pela direita).
		}
	}
	if out.version >= formatVersion {
		if out.source == "" || out.target == "" {
			return headerInfo{}, false, fmt.Errorf("strings: bloco v%d sem o mapa source=/target= do par CAT", out.version)
		}
		if out.source == out.target {
			return headerInfo{}, false, fmt.Errorf("strings: source e target do bloco são o mesmo idioma (%q)", out.source)
		}
	}
	return out, true, nil
}

// parseLine decompõe uma linha de conteúdo com âncora à direita:
// último campo = lang, penúltimo = index, resto = prefixo opaco
// (campos futuros crescem pela esquerda) com id [:name]. A versão do
// FORMATO não é campo da linha: vive só no header /*key=*/ — no primeiro
// campo da chave ela é erro de formato, nunca absorvida como prefixo.
//
// O valor pode trazer a projeção do export no formato v1:
// `<valor>║source = <esc>` e `<valor>║target = <esc>` (o escapeValue
// garante que ║ literal não existe no texto), inclusive em linha de ref
// "$hash". No v2 não há campos: o par são linhas próprias.
func parseLine(line string, lineno int, opts formatters.Options) (parsedLine, error) {
	fail := func(format string, args ...any) (parsedLine, error) {
		return parsedLine{}, fmt.Errorf("strings line %d: %s", lineno, fmt.Sprintf(format, args...))
	}
	ki := stdstrings.Index(line, keyValueSep)
	if ki < 0 {
		return fail("missing %q separator", keyValueSep)
	}
	keyPart, rest := line[:ki], line[ki+len(keyValueSep):]
	eq := stdstrings.Index(rest, "=")
	if eq < 0 {
		return fail("missing = separator")
	}
	rawHash := stdstrings.TrimSpace(rest[:eq])
	bare, ok := hash.Strip(rawHash)
	if !ok || bare == "" {
		return fail("missing $hash before =")
	}
	if len(bare) != 16 {
		return fail("hash must be 16 hex chars, got %q", bare)
	}
	if _, err := hex.DecodeString(bare); err != nil {
		return fail("hash is not hex: %q", bare)
	}
	raw := stdstrings.TrimPrefix(rest[eq+1:], " ")
	value, source, target := splitProjection(raw)
	fields := stdstrings.Split(keyPart, ":")
	if len(fields) < 3 {
		return fail("key needs at least id:index:lang, got %q", keyPart)
	}
	lang := fields[len(fields)-1]
	if stdstrings.TrimSpace(lang) == "" {
		return fail("empty lang in %q", keyPart)
	}
	index, err := strconv.Atoi(fields[len(fields)-2])
	if err != nil || index < 0 {
		return fail("invalid index in %q", keyPart)
	}
	left := fields[:len(fields)-2]
	if len(left) > 0 {
		// A versão vive só no header /*key=*/ (ParseKey a recupera). No
		// primeiro campo da chave ela é erro de formato: sem isto, seria
		// absorvida como prefixo futuro e a linha parsearia ERRADO em
		// silêncio (id="ffx", name=<id>).
		if isReservedKeyPrefix(left[0], opts) {
			return fail("unexpected version prefix in key %q", keyPart)
		}
	}
	var id, name string
	switch len(left) {
	case 0:
		return fail("missing id in %q", keyPart)
	case 1:
		id = left[0]
	default:
		// Crescimento futuro entra pela esquerda: id = penúltimo,
		// name = último do prefixo.
		id = left[len(left)-2]
		name = left[len(left)-1]
	}
	if stdstrings.TrimSpace(id) == "" {
		return fail("empty id in %q", keyPart)
	}
	pl := parsedLine{id: id, name: name, index: index, lang: lang, hash: bare, source: source, target: target}
	if rv, isRef := hash.Strip(value); isRef && rv == bare {
		pl.ref = true
		return pl, nil
	}
	pl.value = unescapeValue(value)
	return pl, nil
}

// splitProjection separa o valor literal da projeção source/target gravada
// pelo export (`║source = …` e `║target = …`). Os textos vêm escapados.
func splitProjection(raw string) (value, source, target string) {
	srcMark := keyValueSep + "source = "
	trgMark := keyValueSep + "target = "
	if idx := indexUnescaped(raw, srcMark); idx >= 0 {
		value = raw[:idx]
		rest := raw[idx+len(srcMark):]
		if tIdx := indexUnescaped(rest, trgMark); tIdx >= 0 {
			source = unescapeValue(rest[:tIdx])
			target = unescapeValue(rest[tIdx+len(trgMark):])
		} else {
			source = unescapeValue(rest)
		}
		return value, source, target
	}
	if idx := indexUnescaped(raw, trgMark); idx >= 0 {
		value = raw[:idx]
		target = unescapeValue(raw[idx+len(trgMark):])
		return value, "", target
	}
	return raw, "", ""
}

// indexUnescaped devolve o índice da primeira ocorrência de marker que NÃO
// está escapada: escapeValue grava `\║` para um ║ literal, então o marcador
// só vale quando o ║ tem um número PAR de backslashes imediatamente antes.
func indexUnescaped(s, marker string) int {
	from := 0
	for from <= len(s) {
		idx := stdstrings.Index(s[from:], marker)
		if idx < 0 {
			return -1
		}
		idx += from
		backslashes := 0
		for j := idx - 1; j >= 0 && s[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return idx
		}
		from = idx + len(marker)
	}
	return -1
}

// rowIdentityKey devolve a chave de agrupamento da linha conforme a identidade.
func rowIdentityKey(pl parsedLine, opts formatters.Options) string {
	if opts.IdentityOf() == dto.IdentityKey {
		if pl.name != "" {
			return pl.name
		}
		return strconv.Itoa(pl.index)
	}
	return strconv.Itoa(pl.index) + "\x00" + pl.name
}

// isReservedKeyPrefix diz se o prefixo não pode abrir a chave da row.
func isReservedKeyPrefix(prefix string, opts formatters.Options) bool {
	for _, reserved := range opts.ReservedKeyPrefixes {
		if stdstrings.EqualFold(prefix, reserved) {
			return true
		}
	}
	return false
}
