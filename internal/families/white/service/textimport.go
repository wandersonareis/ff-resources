package service

// IMPORT DE TEXTO DO FF13.
//
// O artefato (JSON/Strings/CSV/XLSX/PO) traz, por unidade, as rows do ZTR:
//
//	Name   = ID da linha no ZTR (a chave estável do FF13)
//	Source = o pristine (us) no momento do export
//	Target = a tradução de mods/ — o ÚNICO texto que o import aplica
//
// Diferente do FFX, a identidade é a CHAVE, não o índice: a ordem do ZTR pode
// mudar sem invalidar a linha. Por isso o import é PARCIAL por natureza — uma
// linha inexistente na unidade é pulada, as demais entram; o arquivo nunca é
// rejeitado por causa de uma row. Um Source divergente só gera aviso (a
// tradução continua sendo aplicada pela chave).
//
// A fonte do PRISTINE é a validada: filelist/white bin primeiro (o dado real
// do jogo), com data/partN só como fallback quando a unidade não está
// indexada. O /data pode estar incompleto/velho; o filelist não.
//
// Destinos independentes (TextImportOptions):
//
//	WriteMods      → reconstrói .ztr/WPD e grava em mods/partN
//	InsertFilelist → insere o binário no filelist (InsertBlobs)
//
// Nada em data/ é alterado.

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
	"ffresources/internal/shared/progress"

	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
)

// translatedTextSubdir é a pasta, dentro de mods/, dos artefatos de texto.
const translatedTextSubdir = "translated/text"

// TranslatedTextRoot devolve a raiz de TEXTOS traduzidos da part:
// <mods>/translated/text/<part> (o mesmo padrão de translated/images/<part>).
func TranslatedTextRoot(part string) string {
	part = config.NormalizePart(part)
	mods, err := config.DataRoot("mods")
	if err != nil {
		return ""
	}
	return filepath.Join(mods, filepath.FromSlash(translatedTextSubdir), part)
}

// textArtifactExts são os formatos aceitos no lote e no seletor.
var textArtifactExts = map[string]bool{
	".json":    true,
	".strings": true,
	".csv":     true,
	".xlsx":    true,
	".po":      true,
}

// textImportAccumulator agrupa as rows por UNIDADE antes de montar o plano:
// o mesmo unit ID pode aparecer em mais de um arquivo, e a unidade é gravada
// UMA vez.
type textImportAccumulator struct {
	units    map[string]*textImportUnit
	files    map[string]bool
	warnings []string
}

type textImportUnit struct {
	ID    string
	Kind  string
	Rows  map[string]textImportRow
	Files map[string]bool
}

type textImportRow struct {
	target string
	source string
	index  int
}

// textImportUnitPlan é a unidade já validada contra o pristine atual.
type textImportUnitPlan struct {
	unitID  string
	kind    string
	unit    *TextUnit
	byKey   map[string]string
	rows    int
	changed int
}

type textImportPlan struct {
	units   []*textImportUnitPlan
	batch   dto.ImportBatch
	summary dto.ImportSummary
	format  string
}

// textArtifactOptions é o MESMO perfil do export do FF13: identidade por
// chave, par us/br e sem ordenação posicional.
func textArtifactOptions() formatters.Options {
	return formatters.Options{
		SourceLang:         "us",
		TargetLang:         "br",
		TargetLanguageName: "pt-BR",
		Identity:           dto.IdentityKey,
		Ordered:            false,
	}
}

// walkTextArtifacts devolve os candidatos relativos à raiz, em ordem estável.
// Raiz inexistente = lote vazio (a pasta pode ainda não ter sido criada).
func walkTextArtifacts(root string) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if !textArtifactExts[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// parseTextArtifact lê UM artefato pelo registry de formatters.
func parseTextArtifact(path string) (dto.Collection, string, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	f, ok := formatters.Get(ext)
	if !ok {
		return nil, ext, fmt.Errorf("formato não suportado: %s (use .json, .strings, .csv, .xlsx ou .po)", ext)
	}
	c, err := formatters.NewReader(f, path, textArtifactOptions()).Read()
	if err != nil {
		return nil, ext, err
	}
	return c, ext, nil
}

// normalizedTextImportOptions garante um destino: a UI manda os flags
// explícitos; chamadas antigas continuam gravando em mods/.
func normalizedTextImportOptions(opts dto.TextImportOptions) dto.TextImportOptions {
	if !opts.WriteMods && !opts.InsertFilelist {
		opts.WriteMods = true
	}
	return opts
}

// textKindSet normaliza o escopo de kinds do lote.
func textKindSet(kinds []string) map[string]bool {
	out := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		if k = strings.ToLower(strings.TrimSpace(k)); k != "" {
			out[k] = true
		}
	}
	return out
}

// isSystemTextUnit reproduz a trava do editor: o layout system preserva um
// bloco de chaves que o writer padrão ainda não reproduz.
func isSystemTextUnit(id string) bool {
	l := strings.ToLower(id)
	return strings.Contains(l, "/system/") || strings.Contains(l, "_system")
}

// addCollection acumula as entradas de UM artefato. Devolve true quando pelo
// menos uma unidade/row entrou no escopo.
func (a *textImportAccumulator) addCollection(part string, p parts.Part, c dto.Collection, file string, wants map[string]bool) bool {
	added := false
	for _, key := range c.SortedKeys() {
		entry := c[key]
		unitID := strings.TrimSpace(entry.Metadata.ID)
		if unitID == "" {
			unitID = strings.TrimSpace(key)
		}
		// A metadata.key canônica é "<part>/<unit-id>". Se ela aponta para
		// OUTRA part, a unidade não pertence à aba ativa.
		if mk := strings.TrimSpace(entry.Metadata.Key); mk != "" {
			if i := strings.IndexByte(mk, '/'); i > 0 {
				prefix := strings.ToLower(mk[:i])
				if strings.HasPrefix(prefix, "part") {
					if prefix != part {
						a.warnings = append(a.warnings,
							fmt.Sprintf("%s: artefato da %s, ignorado na %s", file, prefix, part))
						continue
					}
					if rest := strings.TrimSpace(mk[i+1:]); rest != "" {
						unitID = rest
					}
				}
			}
		}
		if unitID == "" {
			a.warnings = append(a.warnings, fmt.Sprintf("%s: entrada sem id de unidade", file))
			continue
		}
		kind := p.TextKind(unitID)
		if len(wants) > 0 && !wants[kind] {
			continue
		}
		u := a.units[unitID]
		if u == nil {
			u = &textImportUnit{
				ID:    unitID,
				Kind:  kind,
				Rows:  map[string]textImportRow{},
				Files: map[string]bool{},
			}
			a.units[unitID] = u
		}
		u.Files[file] = true
		added = true
		a.files[file] = true
		for _, row := range entry.Rows {
			name := strings.TrimSpace(row.Name)
			if name == "" {
				name = strings.TrimSpace(row.Key)
			}
			if name == "" {
				a.warnings = append(a.warnings,
					fmt.Sprintf("%s: linha sem chave em %s", file, unitID))
				continue
			}
			if row.Target == "" {
				continue // sem tradução: nada a aplicar
			}
			u.Rows[name] = textImportRow{
				target: row.Target,
				source: row.Source,
				index:  row.Index,
			}
		}
	}
	return added
}

// resolveValidatedTextUnit resolve a unidade pela fonte VALIDADA (filelist/
// white bin) e só cai em data/ quando ela não está indexada (jogo não
// configurado ou arquivo só extraído).
func resolveValidatedTextUnit(part, id string) (*TextUnit, error) {
	p := parts.Get(part)
	dir, base, err := UnitScope(id)
	if err != nil {
		return nil, err
	}
	if src := PartSources(p).UnitsOf(id); len(src) > 0 {
		files := make(map[string]UnitFile, len(src))
		for lang, s := range src {
			files[lang] = UnitFile{
				Rel:       s.Rel,
				Container: ContainerOfExt(s.Rel),
				Source:    s,
			}
		}
		if u, uerr := buildResolvedUnit(id, dir, base, files, p); uerr == nil {
			return u, nil
		}
	}
	dataDir, err := config.PartRoot("data", part)
	if err != nil {
		return nil, err
	}
	return ResolveUnit(dataDir, p, id)
}

// ReadUnitPrimaryContent é o ReadUnitContent sem as REFERÊNCIAS: o import só
// precisa do pristine e do texto atual do slot default. Evita decodificar
// todos os idiomas (e não usa o cache da coleção de data/, que é de outra
// fonte).
func ReadUnitPrimaryContent(part string, unit *TextUnit) (*UnitContent, error) {
	ks, err := TextCodecForLang(part, unit.PrimaryLang)
	if err != nil {
		return nil, err
	}
	origRaw, err := ReadUnitFile(unit, unit.PrimaryLang)
	if err != nil {
		return nil, fmt.Errorf("lendo original %s: %w", unit.Primary, err)
	}
	orig, err := ztr.Read(origRaw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", unit.Primary, err)
	}
	origTexts, err := ks.DecodeLines(orig.Lines)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", unit.Primary, err)
	}
	content := &UnitContent{
		IDs:  orig.IDs,
		Orig: make(map[string]string, len(orig.IDs)),
		Cur:  make(map[string]string, len(orig.IDs)),
	}
	for i, key := range orig.IDs {
		text := ""
		if i < len(origTexts) {
			text = origTexts[i]
		}
		content.Orig[key] = text
		content.Cur[key] = text
	}

	modsDir, err := config.PartRoot("mods", part)
	if err != nil {
		return nil, err
	}
	if b, rerr := os.ReadFile(filepath.Join(modsDir, filepath.FromSlash(unit.Primary))); rerr == nil {
		if raw, ok := unitTextBytes(b, unit.Container); ok {
			if mf, zerr := ztr.Read(raw); zerr == nil {
				if mt, derr := ks.DecodeLines(mf.Lines); derr == nil {
					for i, k := range mf.IDs {
						if i < len(mt) {
							content.Cur[k] = mt[i]
						}
					}
				}
			}
		}
	}
	return content, nil
}

// buildTextImportPlan monta o plano sem aplicar. single=true é o fluxo de UM
// arquivo ("Importar de…"): erro de parse volta como error e não há filtro de
// kind. single=false é o lote de mods/translated/text/<part>.
func buildTextImportPlan(part, root string, files []string, wants map[string]bool, single bool, opts dto.TextImportOptions) (*textImportPlan, error) {
	p := parts.Get(part)
	acc := &textImportAccumulator{
		units: map[string]*textImportUnit{},
		files: map[string]bool{},
	}
	batch := dto.ImportBatch{
		Root:     root,
		Kinds:    []string{},
		Files:    []string{},
		Errors:   []string{},
		Warnings: []string{},
	}
	summary := dto.ImportSummary{
		Path:        "",
		Version:     part,
		Languages:   []string{},
		SavesBinary: opts.WriteMods,
		Entries:     []dto.ImportEntryInfo{},
		Usages:      []dto.ImportUsage{},
		Warnings:    []string{},
		Skipped:     []dto.ImportRowIssue{},
		Errors:      []string{},
	}
	langs := map[string]bool{}
	format := ""
	kindCounts := map[string]int{}

	for _, file := range files {
		path := file
		if !filepath.IsAbs(file) {
			path = filepath.Join(root, file)
		}
		c, ext, err := parseTextArtifact(path)
		if err != nil {
			if single {
				return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
			}
			batch.Errors = append(batch.Errors, fmt.Sprintf("%s: %v", file, err))
			continue
		}
		if format == "" {
			format = ext
		}
		for _, entry := range c {
			for _, row := range entry.Rows {
				for lang := range row.Text {
					langs[lang] = true
				}
			}
		}
		acc.addCollection(part, p, c, file, wants)
		if single {
			summary.Path = path
		}
	}
	if single && summary.Path == "" && len(files) == 1 {
		summary.Path = files[0]
	}

	summary.Format = format
	if single && len(acc.units) == 0 {
		return nil, fmt.Errorf("nenhuma unidade de texto do FF13 encontrada no arquivo %s", filepath.Base(summary.Path))
	}

	unitIDs := make([]string, 0, len(acc.units))
	for id := range acc.units {
		unitIDs = append(unitIDs, id)
	}
	sort.Strings(unitIDs)

	plan := &textImportPlan{
		batch:   batch,
		summary: summary,
		format:  format,
	}
	langList := make([]string, 0, len(langs))
	for lang := range langs {
		langList = append(langList, lang)
	}
	sort.Strings(langList)
	plan.summary.Languages = langList

	for _, unitID := range unitIDs {
		u := acc.units[unitID]
		kindCounts[u.Kind]++
		unit, err := resolveValidatedTextUnit(part, unitID)
		if err != nil {
			msg := fmt.Sprintf("%s: %v", unitID, err)
			plan.batch.Warnings = append(plan.batch.Warnings, msg)
			plan.summary.Warnings = append(plan.summary.Warnings, msg)
			plan.summary.Entries = append(plan.summary.Entries, dto.ImportEntryInfo{
				ID:    unitID,
				Key:   part + "/" + unitID,
				Error: err.Error(),
			})
			continue
		}
		content, err := ReadUnitPrimaryContent(part, unit)
		if err != nil {
			msg := fmt.Sprintf("%s: %v", unitID, err)
			plan.batch.Warnings = append(plan.batch.Warnings, msg)
			plan.summary.Warnings = append(plan.summary.Warnings, msg)
			plan.summary.Entries = append(plan.summary.Entries, dto.ImportEntryInfo{
				ID:    unitID,
				Key:   part + "/" + unitID,
				Error: err.Error(),
			})
			continue
		}
		byKey := make(map[string]string, len(content.Cur))
		for key, text := range content.Cur {
			byKey[key] = text
		}
		names := make([]string, 0, len(u.Rows))
		for name := range u.Rows {
			names = append(names, name)
		}
		sort.Strings(names)

		changed := 0
		stale := 0
		skipped := make([]dto.ImportRowIssue, 0)
		for _, name := range names {
			row := u.Rows[name]
			cur, ok := content.Cur[name]
			if !ok {
				skipped = append(skipped, dto.ImportRowIssue{
					ID:     unitID,
					Index:  row.index,
					Text:   truncateImportText(row.target),
					Reason: "linha não existe na unidade",
				})
				continue
			}
			if row.source != "" {
				if orig := content.Orig[name]; orig != "" && row.source != orig {
					stale++
				}
			}
			if row.target == cur {
				continue
			}
			byKey[name] = row.target
			changed++
		}

		info := dto.ImportEntryInfo{
			ID:              unitID,
			Key:             part + "/" + unitID,
			IndexCount:      len(u.Rows),
			StoreIndexCount: len(content.IDs),
			ChangedTexts:    changed,
			SkippedRows:     len(skipped),
		}
		plan.summary.Entries = append(plan.summary.Entries, info)
		plan.summary.Skipped = append(plan.summary.Skipped, skipped...)
		if len(skipped) > 0 {
			plan.batch.Warnings = append(plan.batch.Warnings,
				fmt.Sprintf("%s: %d linha(s) ignorada(s)", unitID, len(skipped)))
		}
		if stale > 0 {
			plan.batch.Warnings = append(plan.batch.Warnings,
				fmt.Sprintf("%s: %d linha(s) com origem divergente — revisar", unitID, stale))
		}
		plan.units = append(plan.units, &textImportUnitPlan{
			unitID:  unitID,
			kind:    u.Kind,
			unit:    unit,
			byKey:   byKey,
			rows:    len(u.Rows),
			changed: changed,
		})
		plan.batch.EntryCount++
		plan.batch.ChangedTexts += changed
		plan.summary.EntryCount++
		plan.summary.TotalIndices += len(u.Rows)
		plan.summary.ChangedTexts += changed
	}

	kinds := make([]string, 0, len(kindCounts))
	for kind := range kindCounts {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	plan.batch.Kinds = kinds
	if len(kinds) == 1 {
		plan.summary.Kind = kinds[0]
	} else if len(kinds) > 1 {
		plan.summary.Kind = parts.KindText
	} else {
		plan.summary.Kind = parts.KindText
	}

	for file := range acc.files {
		plan.batch.Files = append(plan.batch.Files, file)
	}
	sort.Strings(plan.batch.Files)
	plan.batch.Warnings = append(plan.batch.Warnings, acc.warnings...)
	plan.summary.Warnings = append(plan.summary.Warnings, acc.warnings...)
	plan.summary.Errors = plan.batch.Errors
	return plan, nil
}

// truncateImportText encurta o texto para o resumo de rows ignoradas.
func truncateImportText(s string) string {
	const max = 120
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// PreviewImportTranslated varre <mods>/translated/text/<part> e devolve o
// resumo do lote SEM aplicar.
func (s *Service) PreviewImportTranslated(part string, kinds []string) (dto.ImportBatch, error) {
	part = config.NormalizePart(part)
	wants := textKindSet(kinds)
	if len(wants) == 0 {
		return dto.ImportBatch{}, fmt.Errorf("nenhum tipo de texto selecionado")
	}
	root := TranslatedTextRoot(part)
	files, err := walkTextArtifacts(root)
	if err != nil {
		return dto.ImportBatch{}, err
	}
	plan, err := buildTextImportPlan(part, root, files, wants, false, dto.TextImportOptions{WriteMods: true})
	if err != nil {
		return dto.ImportBatch{}, err
	}
	return plan.batch, nil
}

// ImportTranslated aplica o lote de texto com os destinos escolhidos.
func (s *Service) ImportTranslated(part string, kinds []string, opts dto.TextImportOptions) (dto.ImportBatch, error) {
	part = config.NormalizePart(part)
	opts = normalizedTextImportOptions(opts)
	wants := textKindSet(kinds)
	if len(wants) == 0 {
		return dto.ImportBatch{}, fmt.Errorf("nenhum tipo de texto selecionado")
	}
	root := TranslatedTextRoot(part)
	files, err := walkTextArtifacts(root)
	if err != nil {
		return dto.ImportBatch{}, err
	}
	plan, err := buildTextImportPlan(part, root, files, wants, false, opts)
	if err != nil {
		return dto.ImportBatch{}, err
	}
	if err := s.applyTextImportPlan(part, plan, opts); err != nil {
		return plan.batch, err
	}
	return plan.batch, nil
}

// PreviewImport lê UM arquivo e devolve o resumo do modal "Importar de…".
func (s *Service) PreviewImport(part, path string, opts dto.TextImportOptions) (dto.ImportSummary, error) {
	part = config.NormalizePart(part)
	opts = normalizedTextImportOptions(opts)
	plan, err := buildTextImportPlan(part, "", []string{path}, nil, true, opts)
	if err != nil {
		return dto.ImportSummary{}, err
	}
	return plan.summary, nil
}

// ImportFile aplica UM arquivo e devolve quantos textos mudaram.
func (s *Service) ImportFile(part, path string, opts dto.TextImportOptions) (int, error) {
	part = config.NormalizePart(part)
	opts = normalizedTextImportOptions(opts)
	plan, err := buildTextImportPlan(part, "", []string{path}, nil, true, opts)
	if err != nil {
		return 0, err
	}
	if err := s.applyTextImportPlan(part, plan, opts); err != nil {
		return plan.summary.ChangedTexts, err
	}
	if plan.summary.ChangedTexts == 0 && len(plan.batch.Errors) > 0 {
		return 0, fmt.Errorf("%s", plan.batch.Errors[0])
	}
	return plan.summary.ChangedTexts, nil
}

// applyTextImportPlan reconstrói e entrega as unidades planejadas.
func (s *Service) applyTextImportPlan(part string, plan *textImportPlan, opts dto.TextImportOptions) error {
	if len(plan.units) == 0 {
		return nil
	}
	ks, err := TextCodec(part)
	if err != nil {
		return err
	}
	modsDir, err := config.PartRoot("mods", part)
	if err != nil {
		return err
	}

	batch := s.progress.NewBatch("Importando textos…")
	step := batch.Step("Textos")
	if step == nil {
		step = &progress.Step{}
	}
	step.Begin("Textos", len(plan.units))

	var insertItems []InsertItem
	byRel := make(map[string]*textImportUnitPlan, len(plan.units))
	subtract := func(up *textImportUnitPlan) {
		plan.batch.ChangedTexts -= up.changed
		if plan.batch.ChangedTexts < 0 {
			plan.batch.ChangedTexts = 0
		}
		plan.summary.ChangedTexts -= up.changed
		if plan.summary.ChangedTexts < 0 {
			plan.summary.ChangedTexts = 0
		}
	}

	for _, up := range plan.units {
		if isSystemTextUnit(up.unitID) {
			reason := "layout system ainda não suportado para salvar (preserva o bloco de chaves)"
			plan.batch.Warnings = append(plan.batch.Warnings, up.unitID+": "+reason)
			plan.summary.Warnings = append(plan.summary.Warnings, up.unitID+": "+reason)
			subtract(up)
			step.Issue(up.unitID, reason)
			continue
		}
		var blob []byte
		if up.unit.Container == "wpd" {
			blob, err = PackWpdText(modsDir, up.unit, ks, up.byKey)
		} else {
			blob, err = WriteZtrText(up.unit, ks, up.byKey)
		}
		if err != nil {
			msg := up.unitID + ": " + err.Error()
			plan.batch.Warnings = append(plan.batch.Warnings, msg)
			plan.summary.Warnings = append(plan.summary.Warnings, msg)
			subtract(up)
			step.Issue(up.unitID, err.Error())
			continue
		}
		wroteMods := false
		if opts.WriteMods {
			if werr := writeModsFile(part, up.unit.Primary, blob); werr != nil {
				msg := up.unitID + ": " + werr.Error()
				plan.batch.Warnings = append(plan.batch.Warnings, msg)
				plan.summary.Warnings = append(plan.summary.Warnings, msg)
				// Falha em mods não impede o destino independente Inserir.
			} else {
				wroteMods = true
			}
		}
		if opts.InsertFilelist {
			insertItems = append(insertItems, InsertItem{RelPath: up.unit.Primary, Data: blob})
			byRel[up.unit.Primary] = up
		}
		if !wroteMods && !opts.InsertFilelist {
			subtract(up)
			step.Issue(up.unitID, "falha ao gravar em mods/")
			continue
		}
		step.Step(up.unitID)
	}

	if opts.InsertFilelist && len(insertItems) > 0 {
		outcomes, ierr := s.InsertBlobs(part, insertItems)
		if ierr != nil {
			plan.batch.Errors = append(plan.batch.Errors, "filelist: "+ierr.Error())
			plan.summary.Errors = append(plan.summary.Errors, "filelist: "+ierr.Error())
		}
		for _, o := range outcomes {
			if o.Inserted {
				continue
			}
			up := byRel[o.RelPath]
			msg := o.RelPath + ": " + o.Reason
			if opts.WriteMods {
				plan.batch.Warnings = append(plan.batch.Warnings, msg)
				plan.summary.Warnings = append(plan.summary.Warnings, msg)
				continue
			}
			plan.batch.Errors = append(plan.batch.Errors, msg)
			plan.summary.Errors = append(plan.summary.Errors, msg)
			if up != nil {
				subtract(up)
			}
		}
	}

	if opts.WriteMods {
		invalidateCollection(part)
	}
	step.End()
	batch.SetSummary(len(plan.units), TranslatedTextRoot(part))
	batch.End()
	return nil
}
