// Package csv implementa o formato CSV (RFC 4180) para exportação/importação
// dos arquivos de localização. É um formato tabular amplamente suportado
// por planilhas (Excel, LibreOffice, Google Sheets).
package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	"ffresources/internal/shared/artifact"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/hash"
)

// extensionCSV é a extensão produzida pelo formatter CSV.
const extensionCSV = ".csv"

// ExtensionCSV retorna a extensão produzida pelo formatter CSV.
func ExtensionCSV() string {
	return extensionCSV
}

// CSVFormatter formata a Collection textual para CSV.
type CSVFormatter struct{}

// NewCSVFormatter constrói o formatter CSV.
func NewCSVFormatter() CSVFormatter {
	return CSVFormatter{}
}

// Extension retorna a extensão produzida por este formatter.
func (CSVFormatter) Extension() string {
	return extensionCSV
}

// csvHeader define as colunas do CSV. A coluna key é a metadata.key
// COMPLETA (version/pattern/arquivo) — a identidade da entrada no app: é
// dela que saem o kind, a versão e o casamento por basename, então um
// artefato com VÁRIAS entradas (events_all, objects_all) volta inteiro no
// import. id é o metadata.ID (ex.: chunk_07 do dicionário, que não vive
// no nome do arquivo).
//
// source/target são a projeção de export para CAT (ver dto.TextRow): source
// é a referência `us` de data//.vbf e target é a tradução de mods/ — o
// ÚNICO texto que o import aplica. As duas colunas só vêm preenchidas nas
// linhas do idioma `us`.
var csvHeader = []string{
	"key",
	"id",
	"name",
	"index",
	"lang",
	"hash",
	"text",
	"source",
	"target",
	"divergent",
}

// Índices das colunas (csvHeader).
const (
	colKey       = 0
	colID        = 1
	colName      = 2
	colIndex     = 3
	colLang      = 4
	colHash      = 5
	colText      = 6
	colSource    = 7
	colTarget    = 8
	colDivergent = 9
)

// Marshal organiza a Collection pronta no formato CSV (todos os idiomas).
// Recebe apenas DTO pronto; não toca no domínio nem muta a entrada.
func (CSVFormatter) Marshal(c dto.Collection) ([]byte, error) {
	return (CSVFormatter{}).MarshalOptions(c, formatters.DefaultOptions())
}

// MarshalLangs organiza a Collection pronta contendo só os idiomas pedidos
// (nil/vazio = todos).
func (CSVFormatter) MarshalLangs(c dto.Collection, langs []string) ([]byte, error) {
	opts := formatters.DefaultOptions()
	opts.Languages = langs
	return (CSVFormatter{}).MarshalOptions(c, opts)
}

// MarshalOptions serializa com o perfil neutro (idiomas, identidade, dedup).
func (CSVFormatter) MarshalOptions(c dto.Collection, opts formatters.Options) ([]byte, error) {
	return marshalOptions(c, opts, opts.DedupEnabled())
}

func marshalOptions(c dto.Collection, opts formatters.Options, dedup bool) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ','
	writer.UseCRLF = false // LF padrão; planilhas leem ambos

	// Escreve header
	if err := writer.Write(csvHeader); err != nil {
		return nil, err
	}

	seen := make(map[string]string) // hashHex bare -> texto (só idioma fonte)

	for _, k := range c.SortedKeys() {
		entry := c[k]
		rows := opts.SortedRows(entry.Rows)

		key, id := entryColumns(entry, k)

		for _, row := range rows {
			for _, lang := range rowLangs(row, opts) {
				text := row.Text[lang]
				if text == "" && !formatters.HasSourceTarget(row, lang, opts) {
					continue
				}
				h := row.Hash[lang]
				if h == "" {
					h = hash.Sum64Hex(text)
				}

				// Dedup: no idioma fonte, textos repetidos viram ref "$hash"
				value := text
				if dedup && lang == opts.Source() && !row.Divergent && hash.IsDedupEligible(text) {
					if _, dup := seen[h]; dup {
						value = hash.Prefix(h)
					} else {
						seen[h] = text
					}
				}

				divergent := "false"
				if row.Divergent {
					divergent = "true"
				}

				source, target := formatters.SourceTarget(row, lang, opts)
				record := []string{
					key,
					id,
					rowIdentityName(row, opts),
					fmt.Sprintf("%d", row.Index),
					lang,
					hash.Prefix(h),
					value,
					source,
					target,
					divergent,
				}
				if err := writer.Write(record); err != nil {
					return nil, err
				}
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// rowIdentityName devolve o valor da coluna "name" conforme a identidade:
// chave estável (Key) ou qualificador de segmento (Name).
func rowIdentityName(row dto.TextRow, opts formatters.Options) string {
	if opts.IdentityOf() == dto.IdentityKey {
		if row.Key != "" {
			return row.Key
		}
	}
	return row.Name
}

// rowMatches diz se a row existente é a mesma (Index+Name ou Key).
func rowMatches(r dto.TextRow, index int, name string, opts formatters.Options) bool {
	if opts.IdentityOf() == dto.IdentityKey {
		return r.KeyFor(dto.IdentityKey) == name || r.Name == name
	}
	return r.Index == index && r.Name == name
}

// rowLangs devolve os idiomas da row com texto, ordenados e filtrados.
// A linha do idioma fonte entra também quando só há source/target (row sem
// texto no original mas com tradução em mods) — o target precisa chegar ao
// import.
func rowLangs(row dto.TextRow, opts formatters.Options) []string {
	langs := make([]string, 0, len(row.Text))
	for lang, text := range row.Text {
		if text == "" {
			continue
		}
		if !opts.Includes(lang) {
			continue
		}
		langs = append(langs, lang)
	}
	if formatters.HasSourceTarget(row, opts.Source(), opts) && opts.Includes(opts.Source()) {
		hasSource := false
		for _, l := range langs {
			if l == opts.Source() {
				hasSource = true
				break
			}
		}
		if !hasSource {
			langs = append(langs, opts.Source())
		}
	}
	sort.Strings(langs)
	return langs
}

// Unmarshal parseia o CSV de volta para a Collection (DTO), no modo
// posicional padrão.
func (CSVFormatter) Unmarshal(data []byte) (dto.Collection, error) {
	return (CSVFormatter{}).UnmarshalOptions(data, formatters.DefaultOptions())
}

// UnmarshalOptions parseia respeitando a identidade e o idioma fonte.
func (CSVFormatter) UnmarshalOptions(data []byte, opts formatters.Options) (dto.Collection, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = ','
	reader.LazyQuotes = true // mais tolerante
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		return dto.Collection{}, nil
	}

	// Valida header
	if len(records[0]) < len(csvHeader) {
		return nil, fmt.Errorf("invalid CSV header: expected at least %d columns, got %d", len(csvHeader), len(records[0]))
	}
	for i, col := range records[0] {
		if i < len(csvHeader) && col != csvHeader[i] {
			return nil, fmt.Errorf("invalid CSV header column %d: expected %q, got %q", i, csvHeader[i], col)
		}
	}

	c := make(dto.Collection)
	table := make(map[string]string) // hashHex bare -> texto canonical (idioma fonte)

	// Primeira passada: coleta definições canônicas
	for _, rec := range records[1:] {
		if len(rec) < len(csvHeader) {
			continue // pula linha malformada
		}
		lang := rec[colLang]
		hashHex, _ := hash.Strip(rec[colHash])
		text := rec[colText]
		divergent := rec[colDivergent] == "true"

		// Só registra canônica no idioma fonte e não-divergent
		if lang == opts.Source() && !divergent && text != "" && !hash.IsRef(text) {
			if canonical, exists := table[hashHex]; !exists {
				table[hashHex] = text
			} else if canonical != text {
				// Divergência: editor externo mudou o texto sob o mesmo hash
			}
		}
	}

	// Segunda passada: constrói a Collection
	for _, rec := range records[1:] {
		if len(rec) < len(csvHeader) {
			continue
		}
		key := strings.TrimSpace(rec[colKey])
		if key == "" {
			// Sem a key completa não há kind/versão/casamento — o formato
			// exige uma key por linha (nada de reconstruir por id).
			return nil, fmt.Errorf("linha sem metadata.key (re-exporte o arquivo no formato atual)")
		}
		id := rec[colID]
		name := rec[colName]
		indexStr := rec[colIndex]
		lang := rec[colLang]
		// Polícia estrita do formato: todo hash de artefato tem prefixo $
		// (a mesma régua do .strings). Célula vazia é tolerada — artefato
		// feito à mão, o import recalcula pelo texto.
		hashHex, hasPrefix := hash.Strip(rec[colHash])
		if !hasPrefix && hashHex != "" {
			return nil, fmt.Errorf("hash %q sem prefixo $ (re-exporte o arquivo no formato atual)", rec[colHash])
		}
		text := rec[colText]
		source := rec[colSource]
		target := rec[colTarget]
		divergent := rec[colDivergent] == "true"

		// Row sem texto no original ainda entra quando carrega o par
		// source/target: o target é o que o import aplica.
		if text == "" && source == "" && target == "" {
			continue
		}

		// Resolve referências "$hash"
		if refHash, isRef := hash.Strip(text); isRef {
			if canonical, ok := table[refHash]; ok {
				text = canonical
			} else {
				// Órfão: mantém como está (JSON damaged)
			}
		}

		index, _ := parseInt(indexStr)

		entry, ok := c[id]
		if !ok {
			entry = dto.FileEntry{
				Metadata: dto.Metadata{
					Key: key,
					ID:  id,
				},
				Rows: make([]dto.TextRow, 0),
			}
		}

		// Adiciona/atualiza o hash
		hashMap := make(map[string]string)
		if entry.Rows != nil {
			// Procurar row existente para preservar outros hashes
			found := false
			for _, r := range entry.Rows {
				if rowMatches(r, index, name, opts) {
					hashMap = r.Hash
					found = true
					break
				}
			}
			if !found {
				hashMap = make(map[string]string)
			}
		} else {
			hashMap = make(map[string]string)
		}
		hashMap[lang] = hashHex

		textMap := make(map[string]string)
		if entry.Rows != nil {
			for _, r := range entry.Rows {
				if rowMatches(r, index, name, opts) {
					textMap = r.Text
					break
				}
			}
		}
		if textMap == nil {
			textMap = make(map[string]string)
		}
		textMap[lang] = text

		row := dto.TextRow{
			Index:     index,
			Name:      name,
			Hash:      hashMap,
			Text:      textMap,
			Divergent: divergent,
			Character: "",
			Kind:      "",
			// A projeção source/target só existe na linha do idioma fonte
			// (é lá que a tradução vive); o import lê apenas o target.
			Source: source,
			Target: target,
		}
		if opts.IdentityOf() == dto.IdentityKey {
			row.Key = name
		}

		// Remove row existente se houver
		newRows := make([]dto.TextRow, 0, len(entry.Rows)+1)
		for _, r := range entry.Rows {
			if !rowMatches(r, index, name, opts) {
				newRows = append(newRows, r)
			}
		}
		newRows = append(newRows, row)
		dto.SortRowsBy(newRows, opts.IdentityOf())

		entry.Rows = newRows
		// Chave do mapa = id (como no JSON/DTO); a key completa vive na
		// Metadata — é dela que saem kind/versão/casamento.
		c[id] = entry
	}

	// Ordena rows finais
	for k, entry := range c {
		dto.SortRowsBy(entry.Rows, opts.IdentityOf())
		c[k] = entry
	}

	return c, nil
}

// parseInt converte string para int (leniente).
func parseInt(s string) (int, error) {
	var n int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n, nil
}

func init() {
	formatters.Register("csv", NewCSVFormatter())
	formatters.Register("csv-no-dedup", CSVFormatterNoDedup{})
}

// CSVFormatterNoDedup serializa SEM dedup (todo texto literal).
type CSVFormatterNoDedup struct{}

func (CSVFormatterNoDedup) Extension() string {
	return extensionCSV
}

func (CSVFormatterNoDedup) Marshal(c dto.Collection) ([]byte, error) {
	return marshalOptions(c, formatters.DefaultOptions(), false)
}

func (CSVFormatterNoDedup) MarshalLangs(c dto.Collection, langs []string) ([]byte, error) {
	opts := formatters.DefaultOptions()
	opts.Languages = langs
	return marshalOptions(c, opts, false)
}

func (CSVFormatterNoDedup) MarshalOptions(c dto.Collection, opts formatters.Options) ([]byte, error) {
	return marshalOptions(c, opts, false)
}

func (CSVFormatterNoDedup) Unmarshal(data []byte) (dto.Collection, error) {
	return NewCSVFormatter().Unmarshal(data)
}

func (CSVFormatterNoDedup) UnmarshalOptions(data []byte, opts formatters.Options) (dto.Collection, error) {
	return NewCSVFormatter().UnmarshalOptions(data, opts)
}

// entryColumns devolve a coluna key (metadata.key completa) e a coluna id
// (metadata.ID, com fallback para a chave da Collection) de uma entrada.
func entryColumns(entry dto.FileEntry, mapKey string) (key, id string) {
	key = entry.Metadata.Key
	id = entry.Metadata.ID
	if id == "" {
		id = mapKey
	}
	return key, id
}
