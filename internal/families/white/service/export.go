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
	logging "ffresources/internal/shared/logging"
	"ffresources/internal/shared/progress"

	"ffresources/internal/families/white/codecs/whitebin"
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
//
// Unidade cujo idioma fonte ("us") não tem NENHUMA linha (ztr vazio) fica
// FORA do artefato: sem fonte não há row, e as referências dos outros
// idiomas não são exportadas sozinhas. Nada de bloco "row_count=0".

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
	// Source força a origem: "filelist" (validada) ou "data". Vazio = auto
	// (filelist quando o jogo está configurado, senão data).
	Source string `json:"source,omitempty"`
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
func (s *Service) ExportArtifactBatch(items []ExportPlanItem, part string, langs []string, format string, options dto.ExportOptions) ([]string, error) {
	// Validação ANTES do batch: erro daqui ainda é toast do frontend.
	options = normalizeExportOptions(options)
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
	// Os totais vão junto no ProgressStart (StepsWithTotals): o modal abre
	// já com a largura da contagem reservada ("301/301") e não reflui.
	steps := batch.StepsWithTotals(exportStepSpecs(part, items)...)
	// Sem emitter (testes/CLI): batch nil — o step vira no-op.
	if steps == nil {
		steps = make([]*progress.Step, len(items))
	}

	var all []string
	for i, it := range items {
		kind := strings.ToLower(strings.TrimSpace(it.Kind))
		paths, err := s.exportText(steps[i], part, kind, it.IDs, langs, format, it.Source, options)
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

// exportStepSpecs devolve os steps do lote — um por item, na mesma ordem —
// com o rótulo da família e o total JÁ contado. Contar antes do
// ProgressStart é o que permite ao modal reservar a largura da contagem; a
// contagem usa o MESMO exportUnits do exportText, então o número declarado
// aqui é exatamente o que a barra vai percorrer.
//
// Erro na contagem (coleção indisponível, id desconhecido) deixa o total em
// 0: o step continua válido e o exportText/Fail relatam o erro de verdade.
func exportStepSpecs(part string, items []ExportPlanItem) []progress.StepSpec {
	specs := make([]progress.StepSpec, len(items))
	for i, it := range items {
		kind := strings.ToLower(strings.TrimSpace(it.Kind))
		specs[i].Label = exportKindLabelOf(kind)
		units, _, uerr := exportUnitsSource(part, kind, it.IDs, it.Source)
		if uerr != nil {
			continue
		}
		specs[i].Total = len(units)
	}
	return specs
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
func (s *Service) exportText(r *progress.Step, part, kind string, ids []string, langs []string, format, source string, options dto.ExportOptions) ([]string, error) {
	units, filelistSource, err := exportUnitsSource(part, kind, ids, source)
	if err != nil {
		return nil, err
	}
	r.Begin(exportKindLabelOf(kind), len(units))
	r.SetTotal(len(units))

	var paths []string
	c := dto.Collection{}
	for _, unit := range units {
		if options.Decode {
			entry, derr := exportEntryRows(part, unit)
			if derr != nil {
				r.Issue(unit.ID, derr.Error())
			} else if len(entry.Rows) > 0 {
				// Unidade sem NENHUMA linha (o primário "_us" é um ztr vazio: o
				// jogo tem vários) NÃO vira bloco "row_count=0" no artefato.
				c[unit.ID] = entry
			}
		}
		if options.Extract {
			if !filelistSource {
				logging.Warn("export: Extrair ignorado para %s — a fonte não é o filelist", unit.ID)
			} else {
				done, skipped, failed := extractTextUnit(part, unit, "", false)
				paths = append(paths, done...)
				for _, d := range skipped {
					r.Issue(unit.ID, "já existe: "+filepath.Base(d))
				}
				for _, f := range failed {
					r.Issue(unit.ID, f)
				}
			}
		}
		r.Step(unit.ID)
	}

	if options.Decode && len(c) > 0 {
		artifactLangs := langs
		if filelistSource {
			// Do filelist o artefato leva TODOS os idiomas da unidade; o
			// seletor do header só limita a extração por kind em data/.
			artifactLangs = nil
		}
		written, werr := writeArtifact(part, kind, ids, c, format, artifactLangs)
		if werr != nil {
			return paths, werr
		}
		paths = append(paths, written)
	}
	return paths, nil
}

// exportEntryPair exporta UMA unidade nos DOIS formatos do menu de contexto
// (json + strings): a Collection é a mesma, muda só a extensão.
//
// Unidade sem linhas no idioma fonte (ztr vazio) não gera artefato: devolve
// vazio e nenhum erro — o chamador mostra "Nada para exportar".
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
	if len(entry.Rows) == 0 {
		return nil, nil
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

// normalizeExportOptions mantém o contrato antigo: nenhuma opção = Decode.
func normalizeExportOptions(o dto.ExportOptions) dto.ExportOptions {
	if !o.Decode && !o.Extract {
		o.Decode = true
	}
	return o
}

// exportUnitsSource resolve as unidades pela origem pedida:
//
//	"filelist" → índice/white bin (validado); erro se não houver jogo.
//	"data"     → coleção de data/partN.
//	""         → filelist quando disponível, senão data.
//
// O segundo retorno informa se a origem vencedora foi o filelist (o artefato
// leva todos os idiomas; o seletor do header não filtra).
func exportUnitsSource(part, kind string, ids []string, source string) ([]*TextUnit, bool, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "filelist" {
		if units, ok := filelistTextUnits(part, kind, ids); ok {
			return units, true, nil
		}
		// Header export sem o jogo instalado: cai em data/ (a extração do
		// binário fica indisponível e o chamador avisa).
		logging.Warn("export: filelist indisponível para %s — usando data/", part)
	}
	coll, err := collectionFor(part)
	if err != nil {
		return nil, false, err
	}
	units, err := exportUnits(coll, part, kind, ids)
	return units, false, err
}

// filelistTextUnits monta as unidades a partir do índice de fontes validado
// (white bins). IDs vazio = todas as unidades do kind.
func filelistTextUnits(part, kind string, ids []string) ([]*TextUnit, bool) {
	if config.GameRoot(part) == "" {
		return nil, false
	}
	p := parts.Get(part)
	src := PartSources(p)
	if len(src.Units) == 0 {
		return nil, false
	}
	build := func(id string) *TextUnit {
		sources := src.UnitsOf(id)
		if len(sources) == 0 {
			return nil
		}
		dir, base, err := UnitScope(id)
		if err != nil {
			return nil
		}
		files := make(map[string]UnitFile, len(sources))
		for lang, s := range sources {
			files[lang] = UnitFile{
				Rel:       s.Rel,
				Container: ContainerOfExt(s.Rel),
				Source:    s,
			}
		}
		u := buildUnit(dir, base, files, "", p.TextKind(id))
		if u.Primary == "" {
			return nil
		}
		return u
	}
	matchesKind := func(id string) bool {
		k := strings.ToLower(strings.TrimSpace(kind))
		return k == "" || k == "ztr" || p.TextKind(id) == k
	}
	out := make([]*TextUnit, 0, len(src.Units))
	if len(ids) == 0 {
		for id := range src.Units {
			if !matchesKind(id) {
				continue
			}
			if u := build(id); u != nil {
				out = append(out, u)
			}
		}
	} else {
		for _, raw := range ids {
			id := config.NormalizeInternalPath(raw)
			if id == "" || !matchesKind(id) {
				continue
			}
			if u := build(id); u != nil {
				out = append(out, u)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, true
}

// extractTextUnit grava TODOS os arquivos da unidade (todos os idiomas) em
// destRoot/<rel>, descomprimindo o zlib do white bin. Existente = skipped,
// salvo force. destRoot vazio = data/partN.
func extractTextUnit(part string, unit *TextUnit, destRoot string, force bool) (done, skipped, failed []string) {
	dest := strings.TrimSpace(destRoot)
	if dest == "" {
		d, err := config.PartRoot("data", part)
		if err != nil {
			return nil, nil, []string{err.Error()}
		}
		dest = d
	}
	langs := make([]string, 0, len(unit.Sources))
	for lang := range unit.Sources {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	for _, lang := range langs {
		f := unit.Sources[lang]
		src := f.Source
		if src.Bin == "" || src.Coll == nil {
			continue
		}
		dst := filepath.Join(dest, filepath.FromSlash(f.Rel))
		if !force {
			if _, err := os.Stat(dst); err == nil {
				skipped = append(skipped, dst)
				continue
			}
		}
		raw, err := whitebin.ExtractBytes(src.Bin, src.VPath, src.Coll)
		if err != nil {
			failed = append(failed, f.Rel+": "+err.Error())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			failed = append(failed, f.Rel+": "+err.Error())
			continue
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			failed = append(failed, f.Rel+": "+err.Error())
			continue
		}
		done = append(done, dst)
	}
	return done, skipped, failed
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
			Key:    key,
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

// artifactOptions devolve o perfil de serialização do White: identidade por
// chave (o ID do ZTR), ordem não semântica, fonte e alvo "us" e dedup ligado.
func artifactOptions(langs []string) formatters.Options {
	return formatters.Options{
		SourceLang:         "us",
		TargetLang:         "br",
		TargetLanguageName: "pt-BR",
		Languages:          langs,
		Identity:           dto.IdentityKey,
		Ordered:            false,
	}
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
	return formatters.NewWriter(f, c, formatters.SwapExt(f, path), artifactOptions(langs)).Write()
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
