package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	formatters "ffresources/internal/shared/artifact"
	_ "ffresources/internal/shared/artifact/csv"
	_ "ffresources/internal/shared/artifact/json"
	_ "ffresources/internal/shared/artifact/po"
	_ "ffresources/internal/shared/artifact/strings"
	_ "ffresources/internal/shared/artifact/xlsx"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/hash"
	"ffresources/internal/shared/progress"

	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
)

// EXPORT do FF13: o artefato de texto das units de data/partN.
//
// O export monta as rows DELE MESMO — nunca reaproveita as rows de
// exibição (GetTextEntry). Modelo por row (mesmo contrato do FFX):
//
//	Text[lang] = o pristine de data/ nesse idioma — TODOS os idiomas da
//	             unidade vão como idiomas de /data, o default ("us") incluso;
//	Source     = cópia do idioma default (== Text["us"], o pristine);
//	Target     = o texto de mods/ (o que o import aplica no slot "us").
//
// O hash é o do TEXTO EXPORTADO (hash.Texts sobre os Text), então hash e
// texto ficam COERENTES entre si — o dedup dos formatters agrupa pelo hash
// do texto exportado. O dedup colapsa só o TEXTO repetido no slot default;
// o Target vira linha própria no artefato (idioma alvo "br" no formato
// .strings v2), então o import não perde a tradução.
//
// O export é INDEPENDENTE da flag enableMods: Text/Source vêm sempre de
// data/ e só o Target lê mods/ direto.

// exportKindLabel é o rótulo do step no modal de progresso.
const exportKindLabel = "Textos"

// ExportPlanItem é UMA ação do lote: o kind e os ids dele (vazio = tudo do
// kind; com seleção, os ids marcados na árvore).
//
// Complete marca a seleção TOTAL do kind (checkbox da raiz na árvore). No
// FF13 o escopo total de um kind é o MESMO escopo de "tudo": o artefato é o
// bulk daquele kind.
type ExportPlanItem struct {
	Kind     string   `json:"kind"`
	IDs      []string `json:"ids"`
	Complete bool     `json:"complete"`
}

// exportKindLabelOf devolve o rótulo do kind no modal de progresso (o nome
// da família do texto: Habilidades, Residentes, Zonas, Eventos, Cenários).
func exportKindLabelOf(kind string) string {
	switch kind {
	case parts.KindAC:
		return "Habilidades"
	case parts.KindResident:
		return "Residentes"
	case parts.KindZone:
		return "Zonas"
	case parts.KindEvent:
		return "Eventos"
	case parts.KindScenario:
		return "Cenários"
	default:
		return "Textos"
	}
}

// validExportKind confere se o kind é um kind de TEXTO da part. O kind
// legado "ztr" (o único que existia antes da divisão por pasta) continua
// aceito e significa "todos os kinds da part".
func validExportKind(p parts.Part, kind string) bool {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "ztr" {
		return true
	}
	return p.HasTextKind(kind)
}

// ExportArtifactBatch exporta o lote como UMA operação agrupada: um step
// por kind no modal de progresso (eventos ProgressStart/Update/End). items:
// [{kind, ids}] — ids vazio = tudo do kind; format: "json" | "strings" |
// "csv" | "xlsx" | "po". Os caminhos escritos são devolvidos.
func (s *Service) ExportArtifactBatch(items []ExportPlanItem, part string, langs []string, format string) ([]string, error) {
	// Validação ANTES do batch: erro daqui ainda é toast do frontend.
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "json", "strings", "csv", "xlsx", "po":
	default:
		return nil, fmt.Errorf("formato de export desconhecido: %s", format)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("nada para exportar: lote vazio")
	}
	part = config.NormalizePart(part)
	p := parts.Get(part)
	for _, it := range items {
		if !validExportKind(p, it.Kind) {
			return nil, fmt.Errorf("kind desconhecido: %s", it.Kind)
		}
	}

	batch := s.progress.NewBatch(fmt.Sprintf("Exportando textos (%s)…", exportFormatLabel(format)))
	// UM step por kind: o rótulo de cada família (Residentes, Eventos…).
	steps := batch.Steps(kindSteps(items)...)
	// Sem emitter (testes/CLI): batch nil — o step vira no-op.
	if steps == nil {
		steps = make([]*progress.Step, len(items))
	}

	var all []string
	for i, it := range items {
		kind := strings.ToLower(strings.TrimSpace(it.Kind))
		paths, err := s.exportText(steps[i], part, kind, it.IDs, langs, format)
		all = append(all, paths...)
		if err != nil {
			steps[i].Fail(err.Error())
			batch.End()
			return all, err
		}
	}
	batch.SetSummary(len(all), exportSummaryDestination(all))
	batch.End()
	return all, nil
}

// kindSteps devolve os rótulos dos steps — um por item do lote, na mesma
// ordem. O rótulo é o nome da família do texto, não o nome do kind.
func kindSteps(items []ExportPlanItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, exportKindLabelOf(strings.ToLower(strings.TrimSpace(it.Kind))))
	}
	return out
}

// ExportEntry exporta UMA unidade de texto: o mesmo caminho do lote, com
// o escopo de uma entrada (o par json + strings, como o export do menu de
// contexto do FFX). langs vazio = todos os idiomas.
func (s *Service) ExportEntry(part, id string, langs []string) ([]string, error) {
	part = config.NormalizePart(part)
	id = config.NormalizeInternalPath(id)
	if id == "" {
		return nil, fmt.Errorf("id vazio")
	}
	batch := s.progress.NewBatch(fmt.Sprintf("Exportando %s…", id))
	step := batch.Step(exportKindLabel)
	if step == nil {
		step = &progress.Step{}
	}
	paths, err := s.exportEntryPair(part, id, langs)
	if err != nil {
		step.Fail(err.Error())
		batch.End()
		return nil, err
	}
	batch.SetSummary(len(paths), exportSummaryDestination(paths))
	batch.End()
	return paths, nil
}

// exportText monta as rows do escopo pedido e escreve o artefato no formato.
//
// kind filtra as unidades pela FAMILIA (a pasta): "ac", "resident", "zone",
// "event", "scenario". O kind legado "ztr" (ou vazio) significa TODOS os
// kinds da part.
func (s *Service) exportText(r *progress.Step, part, kind string, ids []string, langs []string, format string) ([]string, error) {
	coll, err := collectionFor(part)
	if err != nil {
		return nil, err
	}
	units, err := exportUnits(coll, part, kind, ids)
	if err != nil {
		return nil, err
	}
	r.Begin(exportKindLabelOf(kind), len(units))
	r.SetTotal(len(units))
	c := dto.Collection{}
	for _, unit := range units {
		entry, err := exportEntryRows(part, unit)
		if err != nil {
			r.Issue(unit.ID, err.Error())
			continue
		}
		c[unit.ID] = entry
		r.Step(unit.ID)
	}
	if len(c) == 0 {
		return nil, fmt.Errorf("nenhuma unidade exportável em %s", part)
	}
	written, err := writeArtifact(part, kind, ids, c, format, langs)
	if err != nil {
		return nil, err
	}
	return []string{written}, nil
}

// exportEntryPair exporta UMA unidade nos DOIS formatos do menu de contexto
// (json + strings): a Collection é a mesma, muda só a extensão.
func (s *Service) exportEntryPair(part, id string, langs []string) ([]string, error) {
	coll, err := collectionFor(part)
	if err != nil {
		return nil, err
	}
	unit, err := collectionUnit(coll, part, id)
	if err != nil {
		return nil, err
	}
	entry, err := exportEntryRows(part, unit)
	if err != nil {
		return nil, err
	}
	c := dto.Collection{unit.ID: entry}
	jsonPath, err := writeArtifact(part, unit.Kind, []string{id}, c, "json", langs)
	if err != nil {
		return nil, err
	}
	stringsPath, err := writeArtifact(part, unit.Kind, []string{id}, c, "strings", langs)
	if err != nil {
		return []string{jsonPath}, err
	}
	return []string{jsonPath, stringsPath}, nil
}

// exportUnits resolve as unidades do escopo: ids vazio = TODAS as unidades
// de texto do KIND na part (o inventário da coleção); com ids, as unidades
// pedidas (resolvidas pelo white bin quando não foram extraídas para data/).
//
// kind vazio ou "ztr" = todos os kinds da part.
func exportUnits(coll *dataCollection, part, kind string, ids []string) ([]*TextUnit, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	filterKind := kind != "" && kind != "ztr"
	if len(ids) == 0 {
		coll.mu.Lock()
		units := make([]*TextUnit, 0, len(coll.units))
		for _, u := range coll.units {
			if filterKind && u.Kind != kind {
				continue
			}
			units = append(units, u)
		}
		coll.mu.Unlock()
		sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })
		return units, nil
	}
	units := make([]*TextUnit, 0, len(ids))
	for _, raw := range ids {
		id := config.NormalizeInternalPath(raw)
		if id == "" {
			continue
		}
		unit, err := collectionUnit(coll, part, id)
		if err != nil {
			return nil, err
		}
		if filterKind && unit.Kind != kind {
			continue
		}
		units = append(units, unit)
	}
	return units, nil
}

// exportEntryRows monta a entry DE EXPORTAÇÃO de uma unidade.
//
// O modelo (por row) — mesmo contrato do FFX (text/source de data/,
// target de mods/):
//
//	Text[lang] = o pristine de data/ nesse idioma: todos os idiomas da
//	             unidade vão como idiomas de /data, o default incluso
//	Source     = cópia do idioma default (== Text[PrimaryLang])
//	Target     = o texto de mods/ (mods-first) — o que o import aplica
//	Hash       = o hash do TEXTO EXPORTADO (hash.Texts sobre Text)
//	Original   = não preenchido (é exclusivo de exibição)
func exportEntryRows(part string, unit *TextUnit) (dto.FileEntry, error) {
	content, err := ReadUnitContent(part, unit)
	if err != nil {
		return dto.FileEntry{}, err
	}
	rows := make([]dto.TextRow, 0, len(content.IDs))
	for i, key := range content.IDs {
		orig := content.Orig[key]
		target := content.Cur[key]
		// O slot do idioma default traz o PRISTINE de data/ (como todos os
		// idiomas da unidade) — a tradução vai SOMENTE no Target.
		texts := map[string]string{unit.PrimaryLang: orig}
		for lang, m := range content.Refs {
			if txt, ok := m[key]; ok {
				texts[lang] = txt
			}
		}
		rows = append(rows, dto.TextRow{
			Index:  i,
			Name:   key,
			Text:   texts,
			Hash:   hash.Texts(texts),
			Source: orig,
			Target: target,
		})
	}
	return dto.FileEntry{
		Metadata: dto.Metadata{
			Key:      part + "/" + unit.ID,
			ID:       unit.ID,
			RowCount: len(rows),
			Files:    unit.Files,
		},
		Rows: rows,
	}, nil
}

// writeArtifact grava a Collection no caminho canônico do escopo, com a
// extensão do formato pedido.
func writeArtifact(part, kind string, ids []string, c dto.Collection, format string, langs []string) (string, error) {
	f, ok := formatters.Get(format)
	if !ok {
		return "", fmt.Errorf("formatter %s não registrado", format)
	}
	path, err := exportArtifactPath(part, kind, ids)
	if err != nil {
		return "", err
	}
	return formatters.NewWriter(f, c, formatters.SwapExt(f, path), langs).Write()
}

// exportArtifactPath resolve o caminho do artefato: a pasta edits/ fica na
// RAIZ dos mods (fora da árvore que o jogo lê) e a PART entra no nome —
// um subconjunto nunca pode sobrescrever o artefato completo do tradutor.
//
// O KIND entra no nome: cada família de texto tem o seu artefato, então
// "resident" e "event" nunca colidem no mesmo arquivo.
//
//	tudo do kind (ids vazio) → resident_all_localizations_<part>.<ext>
//	1 entrada                → resident_<id>_all_localizations_<part>.<ext>
//	N entradas               → resident_sel_<n>_<id1>_all_localizations_<part>.<ext>
//
// kind vazio ou "ztr" (o conjunto antigo, todos os kinds) usa o prefixo
// "text" — que é o nome que os artefatos antigos já têm.
func exportArtifactPath(part, kind string, ids []string) (string, error) {
	modsRoot, err := config.DataRoot("mods")
	if err != nil {
		return "", err
	}
	edits := filepath.Join(modsRoot, "edits")
	if err := os.MkdirAll(edits, 0o755); err != nil {
		return "", fmt.Errorf("criando pasta de edits: %w", err)
	}
	part = config.NormalizePart(part)
	prefix := sanitizeExportKind(kind)
	var name string
	switch len(ids) {
	case 0:
		name = fmt.Sprintf("%s_all_localizations_%s.json", prefix, part)
	case 1:
		name = fmt.Sprintf("%s_%s_all_localizations_%s.json", prefix, sanitizeExportID(ids[0]), part)
	default:
		name = fmt.Sprintf("%s_sel_%d_%s_all_localizations_%s.json", prefix, len(ids), sanitizeExportID(ids[0]), part)
	}
	return filepath.Join(edits, name), nil
}

// sanitizeExportKind normaliza o kind para o nome do arquivo: o kind novo
// entra literal ("resident", "event", "scenario"); vazio ou o kind legado
// "ztr" vira "text" (o nome dos artefatos antigos).
func sanitizeExportKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	if k == "" || k == "ztr" {
		return "text"
	}
	return sanitizeExportID(k)
}

// sanitizeExportID troca a "/" do id da unidade por "_": o id é
// "<dir>/<base>" e a "/" viraria uma pasta no nome do arquivo.
func sanitizeExportID(id string) string {
	return strings.ReplaceAll(strings.TrimSpace(id), "/", "_")
}

// exportFormatLabel é o nome do formato nos títulos.
func exportFormatLabel(format string) string {
	switch strings.ToUpper(strings.TrimSpace(format)) {
	case "JSON":
		return "JSON"
	case "STRINGS":
		return "Strings"
	case "CSV":
		return "CSV"
	case "XLSX":
		return "XLSX"
	case "PO":
		return "PO"
	}
	return format
}

// exportSummaryDestination devolve o destino para o estado final do modal:
// o caminho do único arquivo ou o diretório comum aos vários. Vazio quando
// os artefatos saíram para diretórios diferentes — a contagem já é exibida
// separadamente pelo frontend, então não repetimos "N arquivos" aqui.
func exportSummaryDestination(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return paths[0]
	}
	dir := filepath.Dir(paths[0])
	for _, p := range paths[1:] {
		if filepath.Dir(p) != dir {
			return ""
		}
	}
	return dir
}
