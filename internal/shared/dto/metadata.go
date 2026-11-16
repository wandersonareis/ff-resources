package dto

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Metadata é o contrato mínimo de localização de um texto exportado.
//
//	key      - chave canônica do store (version/pattern, com `/`): ex
//	           ffx/event/obj_ps3/az/azit0000/azit0000.bin
//	row_count - número de rows exportadas (conferência; importer ignora)
//	id       - identificador universal = chave da Collection:
//	           events = eventID, objects = basename sem extensão,
//	           macro = chunk_NN
//	is_dir   - omitido quando arquivo; true quando diretório
//
// O formato da key é responsabilidade da família (ver ffx/profile/metadata).
type Metadata struct {
	Key      string `json:"key"`
	RowCount int    `json:"row_count,omitempty"`
	ID       string `json:"id,omitempty"`
	IsDir    bool   `json:"is_dir,omitempty"`
	// Files mapeia idioma -> caminho do arquivo (FF13: a unidade de texto
	// tem um arquivo por idioma; o "_us" e o editavel).
	Files map[string]string `json:"files,omitempty"`
}

// WithRowCount preenche a contagem de rows (len(entry.Rows) no export).
func (m Metadata) WithRowCount(n int) Metadata {
	m.RowCount = n
	return m
}

// MacroChunkID é o id universal de um chunk: chunk_NN.
func MacroChunkID(chunk int) string {
	return fmt.Sprintf("chunk_%02d", chunk)
}

// ChunkIndexFromID extrai o índice de ids chunk_NN.
func ChunkIndexFromID(id string) (int, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(id), "chunk_")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// ParsedKey é a decomposição de uma key canônica.
type ParsedKey struct {
	Version             string // primeiro segmento (ffx, ffx2, lastmiss)
	DirPattern          string // segmentos intermediários (event/obj_ps3, battle/kernel, menu)
	FileName            string // basename (azit0000.bin)
	Stem                string // basename sem extensão (azit0000)
	LocalizationPattern string // key sem o primeiro segmento
}

// ParseKey decompõe version/pattern em partes. Exige ao menos 2 segmentos.
func ParseKey(key string) (ParsedKey, bool) {
	clean := strings.Trim(filepath.ToSlash(strings.TrimSpace(key)), "/")
	if clean == "" {
		return ParsedKey{}, false
	}
	segs := strings.Split(clean, "/")
	if len(segs) < 2 {
		return ParsedKey{}, false
	}
	fileName := segs[len(segs)-1]
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return ParsedKey{
		Version:             segs[0],
		DirPattern:          strings.Join(segs[1:len(segs)-1], "/"),
		FileName:            fileName,
		Stem:                stem,
		LocalizationPattern: strings.Join(segs[1:], "/"),
	}, true
}
