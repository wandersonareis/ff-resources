package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ffresources/internal/families/white"
	"ffresources/internal/families/white/languages"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/dto"
	"ffresources/internal/shared/hash"
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
func (Service) ListTextUnits(part string) ([]white.TextUnitSummary, error) {
	part = config.NormalizePart(part)
	dataDir, err := config.PartRoot("data", part)
	if err != nil {
		return nil, err
	}
	return ScanTextUnits(dataDir, parts.Get(part))
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
//
// root/indexPath identificam a ORIGEM quando a folha veio do FILELIST. Nesse
// caso a unidade é resolvida no próprio índice/white bin (fonte validada) e
// NÃO passa pelo cache da coleção de data/ — as fontes não se misturam. root
// vazio mantém o caminho de /data.
func (Service) GetTextEntry(part, id, lang, root, indexPath string) (dto.FileEntry, error) {
	part = config.NormalizePart(part)
	id = config.NormalizeInternalPath(id)
	if id == "" {
		return dto.FileEntry{}, fmt.Errorf("id vazio")
	}
	lang = normalizeLang(lang)
	var unit *TextUnit
	var content *UnitContent
	if strings.TrimSpace(root) != "" {
		var err error
		unit, err = ResolveFilelistTextUnit(part, root, indexPath, id)
		if err != nil {
			return dto.FileEntry{}, err
		}
		content, err = ReadUnitContent(part, unit)
		if err != nil {
			return dto.FileEntry{}, err
		}
	} else {
		coll, err := collectionFor(part)
		if err != nil {
			return dto.FileEntry{}, err
		}
		unit, err = collectionUnit(coll, part, id)
		if err != nil {
			return dto.FileEntry{}, err
		}
		content, err = unitContentOf(coll, part, unit)
		if err != nil {
			return dto.FileEntry{}, err
		}
	}

	// A flag de mods decide o que a TABELA mostra: ligada = a traducao de
	// mods/, desligada = o pristine. Source/Target (o par de export) nao
	// mudam com ela.
	modsOn := config.EnableModsEnabled()
	rows := make([]dto.TextRow, len(content.IDs))
	for i, key := range content.IDs {
		origText := content.Orig[key]
		cur := content.Cur[key]
		text := cur
		if !modsOn {
			text = origText
		}
		row := dto.TextRow{
			Index:    i,
			Key:      key,
			Name:     key,
			Text:     map[string]string{"us": text},
			Original: map[string]string{"us": origText},
			Source:   origText,
			Target:   cur,
			Hash:     hash.Texts(map[string]string{"us": origText}),
		}
		for refLang, m := range content.Refs {
			if txt, ok := m[key]; ok {
				row.Original[refLang] = txt
			}
		}
		if lang != "" && lang != unit.PrimaryLang {
			if txt, ok := content.Refs[lang][key]; ok {
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

// ApplyTextCollection grava as edicoes em mods/partN. O original de data/
// nunca e alterado.
//
// A chave da collection e o ID DA UNIDADE ("<dir>/<base>"). Unidade com
// primario .ztr segue o fluxo de sempre (o ZTR e reescrito no modo
// comprimido e com TieMinLast). Unidade com primario .bin (container WPD) e
// reempacotada: o registro .ztr e reescrito e os demais vao byte a byte.
func (Service) ApplyTextCollection(part string, c dto.Collection) error {
	part = config.NormalizePart(part)
	dataDir, err := config.PartRoot("data", part)
	if err != nil {
		return err
	}
	modsDir, err := config.PartRoot("mods", part)
	if err != nil {
		return err
	}
	ks, err := TextCodec(part)
	if err != nil {
		return err
	}
	for id, entry := range c {
		id = config.NormalizeInternalPath(id)
		if id == "" {
			continue
		}
		// O layout system exige o bloco de chaves original para a ordenação
		// de itens do menu. Até o codec ztrlayout existir, salvar em system
		// com o writer padrão pode quebrar o jogo — bloqueia com erro claro.
		if strings.Contains(strings.ToLower(id), "/system/") || strings.Contains(strings.ToLower(id), "_system") {
			return fmt.Errorf("%s: layout system ainda não suportado para salvar (preserva o bloco de chaves)", id)
		}
		unit, err := ResolveUnit(dataDir, parts.Get(part), id)
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
			blob, err = PackWpdText(modsDir, unit, ks, byKey)
		} else {
			blob, err = WriteZtrText(unit, ks, byKey)
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

// ConfigSnapshot alimenta o store de config da UI. Os caminhos saem as raizes
// data/ e mods/; a part ativa e a que a aba principal mandou (SetActivePart).
func (Service) ConfigSnapshot() dto.AppConfigSnapshot {
	part := config.ActivePart()
	data, err := config.DataRoot("data")
	if err != nil {
		return dto.AppConfigSnapshot{EnableMods: config.EnableModsEnabled(), GameVersion: part}
	}
	mods, _ := config.DataRoot("mods")
	return dto.AppConfigSnapshot{
		GameFilesLocation: data,
		TranslateLocation: mods,
		EnableMods:        config.EnableModsEnabled(),
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
// entram como referencias somente-leitura no dialogo do editor. A lista e a
// mesma do codec/profile (internal/families/white/languages): inclui as
// traducoes comunitarias ck/uk/ru.
func (Service) ListLanguages() []Language {
	out := make([]Language, 0, len(languages.FF13))
	for _, l := range languages.FF13 {
		out = append(out, Language{Code: l.Code, Name: l.Name})
	}
	return out
}
