package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"ffresources/apps/ff13/parts"
	"ffresources/internal/families/white"
	"ffresources/internal/families/white/codecs/whitebin"
	"ffresources/internal/families/white/codecs/wpd"
	loggingService "ffresources/internal/shared/logging"
)

// UNIDADES DE TEXTO do FF13.
//
// A arvore de /data nao lista ARQUIVOS: lista UNIDADES. Uma unidade e o
// conjunto de arquivos do mesmo diretorio que compartilham o nome-base
// (txtres_us.ztr + txtres_fr.ztr = a unidade "txtres"), e o arquivo "_us" e
// o unico editavel — os demais idiomas sao REFERENCIAS somente-leitura.
//
// Um arquivo de texto e um .ztr solto OU um container .bin WPD (magic "WPD\0")
// que tenha um registro .ztr dentro. Tudo o mais (imgb, xgr, wdb solto...) e
// ignorado pela varredura.

// unitLangs e o catalogo de idiomas da trilogia (o mesmo de
// internal/families/white/titles/ff13/part1.go). O sufixo "_XX" do nome do
// arquivo so conta quando e um destes codigos: "txtres_ck.ztr" (codigo fora
// do catalogo) e um arquivo SEM idioma, com base "txtres_ck".
var unitLangs = []string{"us", "jp", "fr", "de", "it", "sp", "ch", "kr"}

// isUnitLang diz se o codigo e um idioma do catalogo.
func isUnitLang(code string) bool {
	for _, l := range unitLangs {
		if l == code {
			return true
		}
	}
	return false
}

// langRank ordena os idiomas na ordem do catalogo (us primeiro).
func langRank(code string) int {
	for i, l := range unitLangs {
		if l == code {
			return i
		}
	}
	return len(unitLangs)
}

// parseUnitFileName separa o nome-base do idioma: "txtres_us.ztr" ->
// ("txtres", "us"); "data_backup.bin" -> ("data_backup", "").
func parseUnitFileName(name string) (base, lang string) {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if i := strings.LastIndex(stem, "_"); i > 0 {
		candidate := strings.ToLower(stem[i+1:])
		if isUnitLang(candidate) {
			return stem[:i], candidate
		}
	}
	return stem, ""
}

// unitFile e um arquivo que pertence a uma unidade.
type unitFile struct {
	rel       string     // caminho relativo a data/partN
	container string     // "ztr" (texto solto) | "wpd" (container)
	src       unitSource // de onde os bytes vem (data/partN ou white bin)
}

// textUnit e a unidade de texto resolvida.
type textUnit struct {
	ID        string            // "<dir>/<base>"
	Dir       string            // diretorio interno ("txtres/resident/book")
	Base      string            // nome-base ("txtres")
	Label     string            // rotulo (o base, sem o sufixo de idioma)
	Type      string            // categoria de conteudo (icone)
	Kind      string            // familia do texto (ac/resident/zone/event/scenario)
	Langs     []string          // idiomas presentes (o "" nao e idioma)
	Files     map[string]string // idioma -> arquivo (relativo a data/partN)
	Primary   string            // arquivo editavel (relativo a data/partN)
	Container string            // tipo do primario: "ztr" | "wpd"
	// PrimaryLang e o idioma do primario ("us"): o "_us" da unidade.
	PrimaryLang string
	// files guarda o container e a fonte de CADA arquivo (o primario pode
	// ser .ztr e uma referencia .bin — o mesmo base em dois containers).
	files map[string]unitFile
}

// unitID monta o id da unidade ("<dir>/<base>"; so o base na raiz).
func unitID(dir, base string) string {
	if dir == "" {
		return base
	}
	return dir + "/" + base
}

// splitRel separa um caminho relativo (com "/" mesmo no Windows) em
// (diretorio, arquivo); o diretorio nao tem barra final ("" na raiz).
func splitRel(rel string) (dir, file string) {
	dir, file = path.Split(rel)
	return strings.TrimSuffix(dir, "/"), file
}

// detectWpdZtr confere o magic WPD e devolve o PRIMEIRO registro com extensao
// "ztr" (o texto dentro do container) e os bytes dele.
func detectWpdZtr(raw []byte) (wpd.Record, []byte, bool) {
	f, err := wpd.Read(raw)
	if err != nil {
		return wpd.Record{}, nil, false
	}
	for _, r := range f.Header.Records {
		if !strings.EqualFold(r.Extension, "ztr") {
			continue
		}
		data, ok := f.ExtractRecord(r.Name, r.Extension)
		if !ok {
			return wpd.Record{}, nil, false
		}
		return r, data, true
	}
	return wpd.Record{}, nil, false
}

// readWpdTable le o cabecalho (16 bytes) e a tabela de registros (32*n bytes)
// de um .bin WPD: DUAS leituras, nunca o arquivo inteiro. E o que permite
// varrer thousands de containers sem le-los por completo.
func readWpdTable(path string) (*wpd.Header, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var head [wpd.HeaderOffset]byte
	if _, err := io.ReadFull(f, head[:]); err != nil {
		return nil, fmt.Errorf("wpd: cabecalho curto: %w", err)
	}
	magic := binary.BigEndian.Uint32(head[0:4])
	if magic != wpd.Magic {
		return nil, fmt.Errorf("wpd: magic %#x, esperado %#x", magic, wpd.Magic)
	}
	total := binary.BigEndian.Uint32(head[4:8])
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	tableLen := int64(total) * wpd.RecordSize
	if tableLen > info.Size() {
		return nil, fmt.Errorf("wpd: tabela de %d registros excede o arquivo", total)
	}
	table := make([]byte, tableLen)
	if _, err := io.ReadFull(f, table); err != nil {
		return nil, fmt.Errorf("wpd: tabela de registros truncada: %w", err)
	}
	return wpd.ReadTable(append(head[:], table...))
}

// wpdHasZtr diz se o container tem registro de texto.
func wpdHasZtr(h *wpd.Header) bool {
	for _, r := range h.Records {
		if strings.EqualFold(r.Extension, "ztr") {
			return true
		}
	}
	return false
}

// detectContainer diz o tipo do arquivo de texto: "ztr" (texto solto) ou
// "wpd" (container com registro .ztr). found=false quando o arquivo nao e
// texto (extensao outra, ou .bin sem container WPD).
func detectContainer(path, rel string) (string, bool) {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".ztr":
		return "ztr", true
	case ".bin":
		h, err := readWpdTable(path)
		if err != nil || !wpdHasZtr(h) {
			return "", false
		}
		return "wpd", true
	}
	return "", false
}

// buildUnit monta a unidade a partir dos arquivos coletados no diretorio.
// Sem arquivo "_us" nao ha primario: a unidade nao e editavel (e nao entra
// na arvore).
func buildUnit(dir, base string, files map[string]unitFile, typ, kind string) *textUnit {
	u := &textUnit{
		ID:    unitID(dir, base),
		Dir:   dir,
		Base:  base,
		Label: base,
		Type:  typ,
		Kind:  kind,
		Files: map[string]string{},
		files: files,
	}
	for lang, f := range files {
		u.Files[lang] = f.rel
		if lang != "" {
			u.Langs = append(u.Langs, lang)
		}
	}
	sort.Slice(u.Langs, func(i, j int) bool {
		ri, rj := langRank(u.Langs[i]), langRank(u.Langs[j])
		if ri != rj {
			return ri < rj
		}
		return u.Langs[i] < u.Langs[j]
	})
	if primary, ok := files["us"]; ok {
		u.Primary = primary.rel
		u.Container = primary.container
		u.PrimaryLang = "us"
	}
	return u
}

// summary converte a unidade no sumario que vai para a UI.
func (u *textUnit) summary(partID string) white.TextUnitSummary {
	langs := u.Langs
	if langs == nil {
		langs = []string{}
	}
	return white.TextUnitSummary{
		ID:        u.ID,
		Key:       partID + "/" + u.ID,
		Label:     u.Label,
		Type:      u.Type,
		Kind:      u.Kind,
		Langs:     langs,
		Primary:   u.Primary,
		Files:     u.Files,
		Container: u.Container,
	}
}

// scanTextUnits varre data/partN e devolve as UNIDADES de texto: arquivos
// .ztr soltos e containers .bin WPD que tenham um registro .ztr. Os arquivos
// do mesmo diretorio com o mesmo nome-base formam uma unidade.
//
// A pasta filelist/ fica de fora: os indices copies moram no grupo
// "Filelist" da arvore, nao em /data.
func scanTextUnits(dataDir string, part parts.Part) ([]white.TextUnitSummary, error) {
	units, _, err := scanDataDir(dataDir, part)
	if err != nil {
		return nil, err
	}
	out := make([]white.TextUnitSummary, 0, len(units))
	for _, u := range units {
		out = append(out, u.summary(part.ID()))
	}
	return out, nil
}

// scanDataDir varre data/partN UMA vez e devolve as UNIDADES de texto e os
// ARQUIVOS QUE NAO ENTRARAM EM NENHUMA UNIDADE (os "fora do escopo": imagem,
// script, indice... — o app nao os decodifica).
//
// A regra de unidade mora em um lugar so (parseUnitFileName + buildUnit +
// detectContainer): quem quiser o inventario completo (a colecao de data)
// reaproveita esta varredura em vez de reescreve-la.
//
// Um arquivo de TEXTO sem primario "_us" (o "_ck" fora do catalogo) tambem
// fica fora do escopo: ele existe, mas nao e editavel — nao ha o que mostrar.
func scanDataDir(dataDir string, part parts.Part) ([]*textUnit, []string, error) {
	// dir -> base -> idioma -> arquivo
	byDir := map[string]map[string]map[string]unitFile{}
	// todos os arquivos vistos (a varredura inteira), para saber o que sobrou
	// fora das unidades.
	allRels := make([]string, 0, 256)
	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(dataDir, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "filelist" {
				return fs.SkipDir
			}
			return nil
		}
		allRels = append(allRels, rel)
		dir, file := splitRel(rel)
		base, lang := parseUnitFileName(file)
		container, ok := detectContainer(path, rel)
		if !ok {
			return nil
		}
		bases := byDir[dir]
		if bases == nil {
			bases = map[string]map[string]unitFile{}
			byDir[dir] = bases
		}
		langs := bases[base]
		if langs == nil {
			langs = map[string]unitFile{}
			bases[base] = langs
		}
		// Desempate: dois arquivos do mesmo idioma (o .ztr solto e o .bin
		// WPD) — o .ztr solto ganha.
		if prev, exists := langs[lang]; exists && prev.container == "ztr" {
			return nil
		}
		langs[lang] = unitFile{
			rel:       rel,
			container: container,
			src:       unitSource{root: dataDir, rel: rel},
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	units := make([]*textUnit, 0, len(byDir))
	inUnit := make(map[string]bool, len(allRels))
	for dir, bases := range byDir {
		for base, files := range bases {
			id := unitID(dir, base)
			u := buildUnit(dir, base, files, part.EntryType(id), part.TextKind(id))
			if u.Primary == "" {
				loggingService.Warn("unidade de texto sem arquivo _us ignorada: %s", u.ID)
				continue
			}
			for _, f := range files {
				inUnit[f.rel] = true
			}
			units = append(units, u)
		}
	}
	sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })

	others := make([]string, 0, 16)
	for _, rel := range allRels {
		if !inUnit[rel] {
			others = append(others, rel)
		}
	}
	sort.Strings(others)
	return units, others, nil
}

// resolveTextUnit resolve "<dir>/<base>" listando a pasta e casando
// "<base>_<lang>.ztr|.bin" + "<base>.ztr|.bin" — sem varrer a arvore
// inteira (e o que GetTextEntry/ApplyTextCollection usam).
//
// Só data/partN: os arquivos que o importador extraiu.
func resolveTextUnit(dataDir string, p parts.Part, unitID string) (*textUnit, error) {
	dir, base, err := unitScope(unitID)
	if err != nil {
		return nil, err
	}
	return buildResolvedUnit(unitID, dir, base, collectUnitFiles(dataDir, dir, base), p)
}

// resolveUnit resolve "<dir>/<base>" com o WHITE BIN como reserva.
//
// Os arquivos vêm de data/partN quando estão lá (o caso normal: o
// importador extraiu os .ztr e os arquivos traduzidos) e do white bin
// quando não estão — o filelist tem TODOS os arquivos do jogo, data/partN
// tem só os extraídos.
func resolveUnit(dataDir string, p parts.Part, id string) (*textUnit, error) {
	return resolveUnitLazy(dataDir, p, id, func() *sourceIndex { return partSources(p) })
}

// resolveUnitWithSources é o resolveUnit com o índice de fontes injetado
// (o teste monta o seu).
func resolveUnitWithSources(dataDir string, p parts.Part, src *sourceIndex, id string) (*textUnit, error) {
	return resolveUnitLazy(dataDir, p, id, func() *sourceIndex { return src })
}

// resolveUnitLazy resolve a unidade: primeiro data/partN e, só se ela não
// estiver completa ali, o WHITE BIN.
//
// A fonte do white bin é preguiçosa: a árvore /data (que tem tudo em
// data/partN) não paga a montagem do índice de fontes.
func resolveUnitLazy(dataDir string, p parts.Part, id string, sources func() *sourceIndex) (*textUnit, error) {
	dir, base, err := unitScope(id)
	if err != nil {
		return nil, err
	}
	files := collectUnitFiles(dataDir, dir, base)
	if u, err := buildResolvedUnit(id, dir, base, files, p); err == nil {
		return u, nil
	}
	// A unidade não está completa em data/ (ou não existe): o white bin
	// completa — o filelist tem TODOS os arquivos do jogo.
	for lang, source := range sources().unitsOf(unitID(dir, base)) {
		if _, exists := files[lang]; exists {
			continue
		}
		files[lang] = unitFile{
			rel:       source.rel,
			container: containerOfExt(source.rel),
			src:       source,
		}
	}
	return buildResolvedUnit(id, dir, base, files, p)
}

// unitScope separa o id da unidade em (diretorio, nome-base).
//
// O último segmento do id É o nome-base (o id é sempre "<dir>/<base>"): ele
// não é reprocessado, senão um nome com duas extensões ("x.win32.bin") não
// voltaria para a mesma base.
func unitScope(id string) (dir, base string, err error) {
	id = strings.Trim(strings.TrimSpace(id), "/")
	if id == "" {
		return "", "", fmt.Errorf("id vazio")
	}
	dir, base = splitRel(id)
	if base == "" {
		return "", "", fmt.Errorf("id vazio")
	}
	return dir, base, nil
}

// collectUnitFiles coleta os arquivos da unidade que estão em data/partN.
func collectUnitFiles(dataDir, dir, base string) map[string]unitFile {
	files := map[string]unitFile{}
	absDir := filepath.Join(dataDir, filepath.FromSlash(dir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fileBase, lang := parseUnitFileName(e.Name())
		if fileBase != base {
			continue
		}
		rel := path.Join(dir, e.Name())
		container, ok := detectContainer(filepath.Join(absDir, e.Name()), rel)
		if !ok {
			continue
		}
		// Desempate: dois arquivos do mesmo idioma (o .ztr solto e o .bin
		// WPD) — o .ztr solto ganha.
		if prev, exists := files[lang]; exists && prev.container == "ztr" {
			continue
		}
		files[lang] = unitFile{
			rel:       rel,
			container: container,
			src:       unitSource{root: dataDir, rel: rel},
		}
	}
	return files
}

// buildResolvedUnit monta a unidade a partir dos arquivos coletados e
// valida o primario (sem "_us" a unidade não é editável).
func buildResolvedUnit(id, dir, base string, files map[string]unitFile, p parts.Part) (*textUnit, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("unidade de texto não encontrada: %s", id)
	}
	u := buildUnit(dir, base, files, "", p.TextKind(id))
	if u.Primary == "" {
		return nil, fmt.Errorf("unidade de texto sem arquivo _us: %s", id)
	}
	return u, nil
}

// readUnitFile le o arquivo de uma unidade e devolve os BYTES DO TEXTO: o
// .ztr solto ou o registro .ztr dentro do container WPD.
func readUnitFile(u *textUnit, lang string) ([]byte, error) {
	f, ok := u.files[lang]
	if !ok {
		return nil, fmt.Errorf("arquivo ausente na unidade %s: %s", u.ID, lang)
	}
	raw, err := readSource(f.src)
	if err != nil {
		return nil, err
	}
	if f.container != "wpd" {
		return raw, nil
	}
	_, data, ok := detectWpdZtr(raw)
	if !ok {
		return nil, fmt.Errorf("%s: container WPD sem registro .ztr", f.rel)
	}
	return data, nil
}

// readSource le os bytes de um arquivo de unidade: em data/partN ou no
// white bin (quando o arquivo não foi extraído para data/).
func readSource(src unitSource) ([]byte, error) {
	if src.isDisk() {
		return os.ReadFile(filepath.Join(src.root, filepath.FromSlash(src.rel)))
	}
	if src.bin != "" {
		return whitebin.ExtractBytes(src.bin, src.vpath, src.coll)
	}
	return nil, fmt.Errorf("sem fonte para %s", src.rel)
}
