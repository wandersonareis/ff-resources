package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffresources/apps/ff13/parts"
	"ffresources/internal/families/white"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	ztenc "ffresources/internal/families/white/codecs/ztr/encoding"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/hash"
	loggingService "ffresources/internal/shared/logging"
)

// TextService e a ponte Wails minima da UI FF13 copiada do FFX. Ela le os
// arquivos de data/partN e as traducoes de mods/partN, sem conhecer o
// filelist/white bin: o importador (scripts/ff13import) ja deixou as duas
// arvores no caminho interno.
//
// A UI do FFX usa a interface de TextService (kind/version); o adaptador do
// frontend traduz kind/version para part/id e chama estes metodos.

// TextEntrySummary espelha white.TextEntrySummary para o binding do Wails.
type TextEntrySummary = white.TextEntrySummary

// TextUnitSummary espelha white.TextUnitSummary para o binding do Wails.
type TextUnitSummary = white.TextUnitSummary

// ListTextUnits lista as UNIDADES de TEXTO de data/partN.
//
// A lista nao traz arquivos: traz unidades (o conjunto de idiomas que
// compartilham o nome-base na mesma pasta). Isso e o que permite a arvore
// hibrida (a pasta com UMA unidade vira o no clicavel) e as referencias
// multi-idioma no dialogo do editor.
//
// Só entram unidades de TEXTO: arquivos .ztr soltos e containers .bin WPD
// (magic "WPD\0") que tenham um registro .ztr dentro. Os demais arquivos
// extraidos (imgb, xgr, wdb solto...) nao sao texto e ficam de fora. A pasta
// filelist/ fica de fora: os indices copies moram no grupo "Filelist".
func (FF13Service) ListTextUnits(part string) ([]white.TextUnitSummary, error) {
	part = normalizePart(part)
	dataDir, err := partRoot("data", part)
	if err != nil {
		return nil, err
	}
	return scanTextUnits(dataDir, parts.Get(part))
}

// GetTextEntry decodifica a UNIDADE de texto (kind ztr) de data/partN.
//
// O arquivo "_us" e o primario (o que a tabela mostra): o texto atual vem de
// mods/ quando existe, o pristine de data/. Os DEMAIS idiomas da unidade
// sao REFERENCIAS somente-leitura: entram em Original[lang], alinhados pelo
// ID do ZTR (o mesmo ID = a mesma linha).
//
// lang e o idioma que a FOLHA clicada representa (o filelist sabe o idioma
// do arquivo): o slot editavel ("us") vem SEMPRE — a base do rascunho precisa
// dele — e o idioma pedido entra em Text[lang] (o pristine dele, referencia
// somente-leitura).
func (FF13Service) GetTextEntry(part, id, lang string) (dto.FileEntry, error) {
	part = normalizePart(part)
	id = normalizeInternalPath(id)
	if id == "" {
		return dto.FileEntry{}, fmt.Errorf("id vazio")
	}
	lang = normalizeLang(lang)
	coll, err := collectionFor(part)
	if err != nil {
		return dto.FileEntry{}, err
	}
	unit, err := collectionUnit(coll, part, id)
	if err != nil {
		return dto.FileEntry{}, err
	}
	content, err := unitContentOf(coll, part, unit)
	if err != nil {
		return dto.FileEntry{}, err
	}

	// A flag de mods decide o que a TABELA mostra: ligada = a traducao de
	// mods/, desligada = o pristine. Source/Target (o par de export) nao
	// mudam com ela.
	modsOn := enableModsEnabled()
	rows := make([]dto.TextRow, len(content.ids))
	for i, key := range content.ids {
		origText := content.orig[key]
		cur := content.cur[key]
		text := cur
		if !modsOn {
			text = origText
		}
		row := dto.TextRow{
			Index:    i,
			Name:     key,
			Text:     map[string]string{"us": text},
			Original: map[string]string{"us": origText},
			Source:   origText,
			Target:   cur,
			Hash:     hash.Texts(map[string]string{"us": origText}),
		}
		for refLang, m := range content.refs {
			if txt, ok := m[key]; ok {
				row.Original[refLang] = txt
			}
		}
		if lang != "" && lang != unit.PrimaryLang {
			if txt, ok := content.refs[lang][key]; ok {
				row.Text[lang] = txt
			}
		}
		rows[i] = row
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

// unitContent e a VISAO DEFAULT (o slot "us") de uma unidade: o pristine, o
// texto atual (mods-first) e as referencias dos demais idiomas.
//
// Os textos sao indexados pelo ID do ZTR (o mesmo ID = a mesma linha): e o
// que alinha a traducao com o pristine mesmo quando a ordem das chaves muda.
type unitContent struct {
	ids  []string
	orig map[string]string            // id -> pristine do idioma default
	cur  map[string]string            // id -> texto atual (mods-first)
	refs map[string]map[string]string // idioma -> id -> pristine
}

// readUnitContent decodifica a unidade: o pristine do primario, a traducao
// de mods/ e as referencias dos demais idiomas.
func readUnitContent(part string, unit *textUnit) (*unitContent, error) {
	ks, err := keySet(part)
	if err != nil {
		return nil, err
	}
	// Pristine: o arquivo da unidade em data/ — ou no white bin, quando ele
	// nao foi extraido para data/.
	origRaw, err := readUnitFile(unit, unit.PrimaryLang)
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
	content := &unitContent{
		ids:  orig.IDs,
		orig: make(map[string]string, len(orig.IDs)),
		cur:  make(map[string]string, len(orig.IDs)),
		refs: map[string]map[string]string{},
	}
	for i, key := range orig.IDs {
		text := ""
		if i < len(origTexts) {
			text = origTexts[i]
		}
		content.orig[key] = text
		content.cur[key] = text
	}

	// Traducao: o arquivo de mods pode ter as mesmas chaves em outra ordem.
	modsDir, err := partRoot("mods", part)
	if err != nil {
		return nil, err
	}
	if b, rerr := os.ReadFile(filepath.Join(modsDir, filepath.FromSlash(unit.Primary))); rerr == nil {
		if raw, ok := unitTextBytes(b, unit.Container); ok {
			if mf, zerr := ztr.Read(raw); zerr == nil {
				if mt, derr := ks.DecodeLines(mf.Lines); derr == nil {
					for i, k := range mf.IDs {
						if i < len(mt) {
							content.cur[k] = mt[i]
						}
					}
				} else {
					loggingService.Warn("tradução %s ilegível (decode): %v", unit.Primary, derr)
				}
			} else {
				loggingService.Warn("tradução %s ilegível (ZTR): %v", unit.Primary, zerr)
			}
		}
	}

	// Referencias: os outros idiomas da unidade, lidos de data/ (ou do
	// white bin) e alinhados pelo ID do ZTR. ID ausente no idioma = sem
	// referencia na row.
	for _, lang := range unit.Langs {
		if lang == unit.PrimaryLang {
			continue
		}
		rel, ok := unit.Files[lang]
		if !ok {
			continue
		}
		raw, err := readUnitFile(unit, lang)
		if err != nil {
			loggingService.Warn("referência %s ilegível: %v", rel, err)
			continue
		}
		zf, err := ztr.Read(raw)
		if err != nil {
			loggingService.Warn("referência %s ilegível: %v", rel, err)
			continue
		}
		texts, err := ks.DecodeLines(zf.Lines)
		if err != nil {
			loggingService.Warn("referência %s ilegível: %v", rel, err)
			continue
		}
		m := make(map[string]string, len(zf.IDs))
		for i, k := range zf.IDs {
			if i < len(texts) {
				m[k] = texts[i]
			}
		}
		content.refs[lang] = m
	}
	return content, nil
}

// unitTextBytes devolve os bytes do TEXTO de um arquivo da unidade: o .ztr
// solto ou o registro .ztr dentro do container WPD.
func unitTextBytes(raw []byte, container string) ([]byte, bool) {
	if container != "wpd" {
		return raw, true
	}
	_, data, ok := detectWpdZtr(raw)
	return data, ok
}

// ApplyTextCollection grava as edicoes em mods/partN. O original de data/
// nunca e alterado.
//
// A chave da collection e o ID DA UNIDADE ("<dir>/<base>"). Unidade com
// primario .ztr segue o fluxo de sempre (o ZTR e reescrito no modo
// comprimido e com TieMinLast). Unidade com primario .bin (container WPD) e
// reempacotada: o registro .ztr e reescrito e os demais vao byte a byte.
func (FF13Service) ApplyTextCollection(part string, c dto.Collection) error {
	part = normalizePart(part)
	dataDir, err := partRoot("data", part)
	if err != nil {
		return err
	}
	modsDir, err := partRoot("mods", part)
	if err != nil {
		return err
	}
	ks, err := keySet(part)
	if err != nil {
		return err
	}
	for id, entry := range c {
		id = normalizeInternalPath(id)
		if id == "" {
			continue
		}
		// O layout system exige o bloco de chaves original para a ordenação
		// de itens do menu. Até o codec ztrlayout existir, salvar em system
		// com o writer padrão pode quebrar o jogo — bloqueia com erro claro.
		if strings.Contains(strings.ToLower(id), "/system/") || strings.Contains(strings.ToLower(id), "_system") {
			return fmt.Errorf("%s: layout system ainda não suportado para salvar (preserva o bloco de chaves)", id)
		}
		unit, err := resolveUnit(dataDir, parts.Get(part), id)
		if err != nil {
			return err
		}
		byKey := make(map[string]string, len(entry.Rows))
		for _, r := range entry.Rows {
			if r.Name == "" {
				continue
			}
			t := r.Text["us"]
			if t == "" {
				t = r.Target
			}
			byKey[r.Name] = t
		}
		var blob []byte
		if unit.Container == "wpd" {
			blob, err = packWpdText(modsDir, unit, ks, byKey)
		} else {
			blob, err = writeZtrText(unit, ks, byKey)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		dst := filepath.Join(modsDir, filepath.FromSlash(unit.Primary))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, blob, 0o644); err != nil {
			return err
		}
	}
	// O mods/ mudou: a visão default cacheada (a tradução) está velha.
	invalidateCollection(part)
	return nil
}

// ztrSaveFormat é o formato da GRAVAÇÃO de texto. O padrão continua o BE do
// jogo; o LE (par/dicionário do Pulse) fica disponível para uma escolha
// posterior (config/UI). A leitura é sempre automática.
var ztrSaveFormat = ztr.FormatBigEndianCompressed

// writeZtrFile centraliza a gravação: todo save passa por aqui para que a
// escolha de formato seja um ponto único.
func writeZtrFile(f *ztr.File) ([]byte, error) {
	return ztr.WriteFormat(f, ztrSaveFormat, ztr.Compressed, ztr.TieMinLast)
}

// writeZtrText reescreve o .ztr solto da unidade (pristine da unidade + os
// textos editados, por ID) e devolve o blob pronto para mods/.
//
// O pristine vem de data/ ou do white bin (quando o arquivo não foi
// extraído para data/).
func writeZtrText(unit *textUnit, ks *ztenc.KeySet, byKey map[string]string) ([]byte, error) {
	raw, err := readUnitFile(unit, unit.PrimaryLang)
	if err != nil {
		return nil, err
	}
	zf, err := ztr.Read(raw)
	if err != nil {
		return nil, err
	}
	lines, err := applyTexts(ks, zf, byKey)
	if err != nil {
		return nil, err
	}
	return writeZtrFile(&ztr.File{IDs: zf.IDs, Lines: lines})
}

// packWpdText reempacota o container WPD: o registro .ztr e reescrito com os
// textos editados e os demais registros vao byte a byte (o .wdb e o resto).
//
// O container base e o de mods/ quando existe (o que o usuario ja salvou),
// senao o pristine da unidade (data/ ou white bin).
func packWpdText(modsDir string, unit *textUnit, ks *ztenc.KeySet, byKey map[string]string) ([]byte, error) {
	var raw []byte
	if b, err := os.ReadFile(filepath.Join(modsDir, filepath.FromSlash(unit.Primary))); err == nil {
		raw = b
	} else {
		var rerr error
		raw, rerr = readSource(unit.files[unit.PrimaryLang].src)
		if rerr != nil {
			return nil, rerr
		}
	}
	f, err := wpd.Read(raw)
	if err != nil {
		return nil, err
	}
	ztrIndex := -1
	for i, r := range f.Header.Records {
		if strings.EqualFold(r.Extension, "ztr") {
			ztrIndex = i
			break
		}
	}
	if ztrIndex < 0 {
		return nil, fmt.Errorf("container sem registro .ztr")
	}
	rec := f.Header.Records[ztrIndex]
	zraw, ok := f.ExtractRecord(rec.Name, rec.Extension)
	if !ok {
		return nil, fmt.Errorf("registro %s.%s não encontrado", rec.Name, rec.Extension)
	}
	zf, err := ztr.Read(zraw)
	if err != nil {
		return nil, err
	}
	lines, err := applyTexts(ks, zf, byKey)
	if err != nil {
		return nil, err
	}
	blob, err := writeZtrFile(&ztr.File{IDs: zf.IDs, Lines: lines})
	if err != nil {
		return nil, err
	}
	parts := make([]wpd.Part, 0, len(f.Header.Records))
	for i, r := range f.Header.Records {
		data, ok := f.ExtractRecord(r.Name, r.Extension)
		if !ok {
			return nil, fmt.Errorf("registro %s.%s não encontrado", r.Name, r.Extension)
		}
		if i == ztrIndex {
			data = blob
		}
		parts = append(parts, wpd.Part{Name: r.Name, Extension: r.Extension, Data: data})
	}
	return wpd.WriteFromParts(parts)
}

// applyTexts reescreve as linhas do ZTR com os textos editados (por ID). As
// linhas sem edicao mantem o texto pristine.
func applyTexts(ks *ztenc.KeySet, zf *ztr.File, byKey map[string]string) ([][]byte, error) {
	texts := make([]string, len(zf.IDs))
	for i, key := range zf.IDs {
		if t, ok := byKey[key]; ok {
			texts[i] = t
			continue
		}
		if i >= len(zf.Lines) {
			return nil, fmt.Errorf("linha %d ausente no arquivo", i)
		}
		origText, derr := ks.DecodeLine(zf.Lines[i])
		if derr != nil {
			return nil, fmt.Errorf("linha %d: %w", i, derr)
		}
		texts[i] = origText
	}
	return ks.EncodeLines(texts)
}

// ConfigSnapshot alimenta o store de config da UI. Os caminhos saem as raizes
// data/ e mods/; a part ativa e a que a aba principal mandou (SetActivePart).
func (FF13Service) ConfigSnapshot() dto.AppConfigSnapshot {
	part := currentActivePart()
	data, err := dataRoot("data")
	if err != nil {
		return dto.AppConfigSnapshot{EnableMods: enableModsEnabled(), GameVersion: part}
	}
	mods, _ := dataRoot("mods")
	return dto.AppConfigSnapshot{
		GameFilesLocation: data,
		TranslateLocation: mods,
		EnableMods:        enableModsEnabled(),
		GameVersion:       part,
	}
}

// Language e o par codigo/nome que a UI exibe no seletor de idiomas.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ListLanguages devolve o catalogo COMPLETO de idiomas da trilogia. O FF13
// usa o esquema de sufixo txt_XX: o "_us" e o texto editavel e os demais
// entram como referencias somente-leitura no dialogo do editor.
func (FF13Service) ListLanguages() []Language {
	return []Language{
		{Code: "us", Name: "English"},
		{Code: "jp", Name: "Japanese"},
		{Code: "fr", Name: "French"},
		{Code: "de", Name: "German"},
		{Code: "it", Name: "Italian"},
		{Code: "sp", Name: "Spanish"},
		{Code: "ch", Name: "Chinese"},
		{Code: "kr", Name: "Korean"},
	}
}

func normalizePart(part string) string {
	p := strings.ToLower(strings.TrimSpace(part))
	switch p {
	case "part1", "part2", "part3":
		return p
	default:
		return "part1"
	}
}

func normalizeInternalPath(id string) string {
	id = filepath.ToSlash(strings.TrimSpace(id))
	id = strings.TrimPrefix(id, "/")
	if id == "" || strings.Contains(id, "..") {
		return ""
	}
	return id
}

func keySet(part string) (*ztenc.KeySet, error) {
	var game ztenc.GameCode
	switch normalizePart(part) {
	case "part2":
		game = ztenc.FF132
	case "part3":
		game = ztenc.FF133
	default:
		game = ztenc.FF131
	}
	if err := ztenc.PrepareKeySet(game, ztenc.LatinJapanese); err != nil {
		return nil, fmt.Errorf("charset %s: %w", game, err)
	}
	return ztenc.KeySetFor(game, ztenc.LatinJapanese)
}

// findRepoRoot sobe a partir do executável/diretório atual até achar o
// go.mod do monorepo. É o que permite achar data/ e mods/ tanto no build
// (apps/ff13/bin) quanto no dev (apps/ff13).
//
// O ambiente vence: FF_RESOURCES_DATA / FF_RESOURCES_MODS apontam as raízes
// quando o app roda de outro lugar (cópia de release, pasta do usuário).
// dataRoot devolve a raiz de dados do FF13, RELATIVA AO EXECUTÁVEL — a mesma
// convenção do projeto FFX antigo (bin/data):
//
//	<bin>/data/partN        originais (filelist/white bin)
//	<bin>/data/mods/partN   traduções (7z)
//
// Overrides: FF_RESOURCES_DATA (raiz dos originais) e FF_RESOURCES_MODS
// (raiz das traduções) para uma cópia de release em outra pasta.
func dataRoot(kind string) (string, error) {
	if kind == "data" {
		if env := strings.TrimSpace(os.Getenv("FF_RESOURCES_DATA")); env != "" {
			return env, nil
		}
	} else if kind == "mods" {
		if env := strings.TrimSpace(os.Getenv("FF_RESOURCES_MODS")); env != "" {
			return env, nil
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("localizando o executável: %w", err)
	}
	data := filepath.Join(filepath.Dir(exe), "data")
	if kind == "mods" {
		return filepath.Join(data, "mods"), nil
	}
	return data, nil
}

// partRoot garante que <data>/partN e <data>/mods/partN existem e devolve o
// diretório.
func partRoot(kind, part string) (string, error) {
	base, err := dataRoot(kind)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, normalizePart(part))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
