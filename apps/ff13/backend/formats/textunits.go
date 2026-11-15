package formats

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

	"ffresources/apps/ff13/backend/parts"
	"ffresources/internal/families/white"
	"ffresources/internal/families/white/codecs/whitebin"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/languages"
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

// UnitLangs e o catalogo de idiomas da trilogia, vindo da lista exclusiva do
// FF13 (internal/families/white/languages). O sufixo "_XX" do nome do arquivo
// so conta quando e um destes codigos; codigo fora do catalogo (ex.: um sufixo
// desconhecido) e tratado como arquivo SEM idioma, com a base incluindo o
// sufixo.
var UnitLangs = languages.FF13Codes()

// IsUnitLang diz se o codigo e um idioma do catalogo.
func IsUnitLang(code string) bool {
	for _, l := range UnitLangs {
		if l == code {
			return true
		}
	}
	return false
}

// LangRank ordena os idiomas na ordem do catalogo (us primeiro).
func LangRank(code string) int {
	for i, l := range UnitLangs {
		if l == code {
			return i
		}
	}
	return len(UnitLangs)
}

// ParseUnitFileName separa o nome-base do idioma: "txtres_us.ztr" ->
// ("txtres", "us"); "data_backup.bin" -> ("data_backup", "").
func ParseUnitFileName(name string) (base, lang string) {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if i := strings.LastIndex(stem, "_"); i > 0 {
		candidate := strings.ToLower(stem[i+1:])
		if IsUnitLang(candidate) {
			return stem[:i], candidate
		}
	}
	return stem, ""
}

// UnitFile e um arquivo que pertence a uma unidade.
type UnitFile struct {
	// Rel é o caminho relativo a data/partN.
	Rel string
	// Container é "ztr" (texto solto) ou "wpd" (container).
	Container string
	// Source diz de onde os bytes vêm (data/partN ou white bin).
	Source UnitSource
}

// TextUnit e a unidade de texto resolvida.
type TextUnit struct {
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
	// Sources guarda o container e a fonte de CADA arquivo (o primario pode
	// ser .ztr e uma referencia .bin — o mesmo base em dois containers).
	Sources map[string]UnitFile
}

// UnitID monta o id da unidade ("<dir>/<base>"; so o base na raiz).
func UnitID(dir, base string) string {
	if dir == "" {
		return base
	}
	return dir + "/" + base
}

// SplitRel separa um caminho relativo (com "/" mesmo no Windows) em
// (diretorio, arquivo); o diretorio nao tem barra final ("" na raiz).
func SplitRel(rel string) (dir, file string) {
	dir, file = path.Split(rel)
	return strings.TrimSuffix(dir, "/"), file
}

// DetectWpdZtr confere o magic WPD e devolve o PRIMEIRO registro com extensao
// "ztr" (o texto dentro do container) e os bytes dele.
func DetectWpdZtr(raw []byte) (wpd.Record, []byte, bool) {
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

// DetectContainer diz o tipo do arquivo de texto: "ztr" (texto solto) ou
// "wpd" (container com registro .ztr). found=false quando o arquivo nao e
// texto (extensao outra, ou .bin sem container WPD).
func DetectContainer(path, rel string) (string, bool) {
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
func buildUnit(dir, base string, files map[string]UnitFile, typ, kind string) *TextUnit {
	u := &TextUnit{
		ID:      UnitID(dir, base),
		Dir:     dir,
		Base:    base,
		Label:   base,
		Type:    typ,
		Kind:    kind,
		Files:   map[string]string{},
		Sources: files,
	}
	for lang, f := range files {
		u.Files[lang] = f.Rel
		if lang != "" {
			u.Langs = append(u.Langs, lang)
		}
	}
	sort.Slice(u.Langs, func(i, j int) bool {
		ri, rj := LangRank(u.Langs[i]), LangRank(u.Langs[j])
		if ri != rj {
			return ri < rj
		}
		return u.Langs[i] < u.Langs[j]
	})
	if primary, ok := files["us"]; ok {
		u.Primary = primary.Rel
		u.Container = primary.Container
		u.PrimaryLang = "us"
	}
	return u
}

// summary converte a unidade no sumario que vai para a UI.
func (u *TextUnit) summary(partID string) white.TextUnitSummary {
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

// ScanTextUnits varre data/partN e devolve as UNIDADES de texto: arquivos
// .ztr soltos e containers .bin WPD que tenham um registro .ztr. Os arquivos
// do mesmo diretorio com o mesmo nome-base formam uma unidade.
//
// A pasta filelist/ fica de fora: os indices copies moram no grupo
// "Filelist" da arvore, nao em /data.
func ScanTextUnits(dataDir string, part parts.Part) ([]white.TextUnitSummary, error) {
	units, _, err := ScanDataDir(dataDir, part)
	if err != nil {
		return nil, err
	}
	out := make([]white.TextUnitSummary, 0, len(units))
	for _, u := range units {
		out = append(out, u.summary(part.ID()))
	}
	return out, nil
}

// ScanDataDir varre data/partN UMA vez e devolve as UNIDADES de texto e os
// ARQUIVOS QUE NAO ENTRARAM EM NENHUMA UNIDADE (os "fora do escopo": imagem,
// script, indice... — o app nao os decodifica).
//
// A regra de unidade mora em um lugar so (ParseUnitFileName + buildUnit +
// DetectContainer): quem quiser o inventario completo (a colecao de data)
// reaproveita esta varredura em vez de reescreve-la.
//
// Um arquivo de TEXTO sem primario "_us" (o "_ck" fora do catalogo) tambem
// fica fora do escopo: ele existe, mas nao e editavel — nao ha o que mostrar.
func ScanDataDir(dataDir string, part parts.Part) ([]*TextUnit, []string, error) {
	// dir -> base -> idioma -> arquivo
	byDir := map[string]map[string]map[string]UnitFile{}
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
		dir, file := SplitRel(rel)
		base, lang := ParseUnitFileName(file)
		container, ok := DetectContainer(path, rel)
		if !ok {
			return nil
		}
		bases := byDir[dir]
		if bases == nil {
			bases = map[string]map[string]UnitFile{}
			byDir[dir] = bases
		}
		langs := bases[base]
		if langs == nil {
			langs = map[string]UnitFile{}
			bases[base] = langs
		}
		// Desempate: dois arquivos do mesmo idioma (o .ztr solto e o .bin
		// WPD) — o .ztr solto ganha.
		if prev, exists := langs[lang]; exists && prev.Container == "ztr" {
			return nil
		}
		langs[lang] = UnitFile{
			Rel:       rel,
			Container: container,
			Source:    UnitSource{Root: dataDir, Rel: rel},
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	units := make([]*TextUnit, 0, len(byDir))
	inUnit := make(map[string]bool, len(allRels))
	for dir, bases := range byDir {
		for base, files := range bases {
			id := UnitID(dir, base)
			u := buildUnit(dir, base, files, part.EntryType(id), part.TextKind(id))
			if u.Primary == "" {
				loggingService.Warn("unidade de texto sem arquivo _us ignorada: %s", u.ID)
				continue
			}
			for _, f := range files {
				inUnit[f.Rel] = true
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

// ResolveTextUnit resolve "<dir>/<base>" listando a pasta e casando
// "<base>_<lang>.ztr|.bin" + "<base>.ztr|.bin" — sem varrer a arvore
// inteira (e o que GetTextEntry/ApplyTextCollection usam).
//
// Só data/partN: os arquivos que o importador extraiu.
func ResolveTextUnit(dataDir string, p parts.Part, unitID string) (*TextUnit, error) {
	dir, base, err := UnitScope(unitID)
	if err != nil {
		return nil, err
	}
	return buildResolvedUnit(unitID, dir, base, collectUnitFiles(dataDir, dir, base), p)
}

// ResolveUnit resolve "<dir>/<base>" com o WHITE BIN como reserva.
//
// Os arquivos vêm de data/partN quando estão lá (o caso normal: o
// importador extraiu os .ztr e os arquivos traduzidos) e do white bin
// quando não estão — o filelist tem TODOS os arquivos do jogo, data/partN
// tem só os extraídos.
func ResolveUnit(dataDir string, p parts.Part, id string) (*TextUnit, error) {
	return ResolveUnitLazy(dataDir, p, id, func() *SourceIndex { return PartSources(p) })
}

// ResolveUnitWithSources é o ResolveUnit com o índice de fontes injetado
// (o teste monta o seu).
func ResolveUnitWithSources(dataDir string, p parts.Part, src *SourceIndex, id string) (*TextUnit, error) {
	return ResolveUnitLazy(dataDir, p, id, func() *SourceIndex { return src })
}

// ResolveUnitLazy resolve a unidade: primeiro data/partN e, só se ela não
// estiver completa ali, o WHITE BIN.
//
// A fonte do white bin é preguiçosa: a árvore /data (que tem tudo em
// data/partN) não paga a montagem do índice de fontes.
func ResolveUnitLazy(dataDir string, p parts.Part, id string, sources func() *SourceIndex) (*TextUnit, error) {
	dir, base, err := UnitScope(id)
	if err != nil {
		return nil, err
	}
	files := collectUnitFiles(dataDir, dir, base)
	if u, err := buildResolvedUnit(id, dir, base, files, p); err == nil {
		return u, nil
	}
	// A unidade não está completa em data/ (ou não existe): o white bin
	// completa — o filelist tem TODOS os arquivos do jogo.
	for lang, source := range sources().UnitsOf(UnitID(dir, base)) {
		if _, exists := files[lang]; exists {
			continue
		}
		files[lang] = UnitFile{
			Rel:       source.Rel,
			Container: ContainerOfExt(source.Rel),
			Source:    source,
		}
	}
	return buildResolvedUnit(id, dir, base, files, p)
}

// UnitScope separa o id da unidade em (diretorio, nome-base).
//
// O último segmento do id É o nome-base (o id é sempre "<dir>/<base>"): ele
// não é reprocessado, senão um nome com duas extensões ("x.win32.bin") não
// voltaria para a mesma base.
func UnitScope(id string) (dir, base string, err error) {
	id = strings.Trim(strings.TrimSpace(id), "/")
	if id == "" {
		return "", "", fmt.Errorf("id vazio")
	}
	dir, base = SplitRel(id)
	if base == "" {
		return "", "", fmt.Errorf("id vazio")
	}
	return dir, base, nil
}

// collectUnitFiles coleta os arquivos da unidade que estão em data/partN.
func collectUnitFiles(dataDir, dir, base string) map[string]UnitFile {
	files := map[string]UnitFile{}
	absDir := filepath.Join(dataDir, filepath.FromSlash(dir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fileBase, lang := ParseUnitFileName(e.Name())
		if fileBase != base {
			continue
		}
		rel := path.Join(dir, e.Name())
		container, ok := DetectContainer(filepath.Join(absDir, e.Name()), rel)
		if !ok {
			continue
		}
		// Desempate: dois arquivos do mesmo idioma (o .ztr solto e o .bin
		// WPD) — o .ztr solto ganha.
		if prev, exists := files[lang]; exists && prev.Container == "ztr" {
			continue
		}
		files[lang] = UnitFile{
			Rel:       rel,
			Container: container,
			Source:    UnitSource{Root: dataDir, Rel: rel},
		}
	}
	return files
}

// buildResolvedUnit monta a unidade a partir dos arquivos coletados e
// valida o primario (sem "_us" a unidade não é editável).
func buildResolvedUnit(id, dir, base string, files map[string]UnitFile, p parts.Part) (*TextUnit, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("unidade de texto não encontrada: %s", id)
	}
	u := buildUnit(dir, base, files, "", p.TextKind(id))
	if u.Primary == "" {
		return nil, fmt.Errorf("unidade de texto sem arquivo _us: %s", id)
	}
	return u, nil
}

// ReadUnitFile le o arquivo de uma unidade e devolve os BYTES DO TEXTO: o
// .ztr solto ou o registro .ztr dentro do container WPD.
func ReadUnitFile(u *TextUnit, lang string) ([]byte, error) {
	f, ok := u.Sources[lang]
	if !ok {
		return nil, fmt.Errorf("arquivo ausente na unidade %s: %s", u.ID, lang)
	}
	raw, err := ReadSource(f.Source)
	if err != nil {
		return nil, err
	}
	if f.Container != "wpd" {
		return raw, nil
	}
	_, data, ok := DetectWpdZtr(raw)
	if !ok {
		return nil, fmt.Errorf("%s: container WPD sem registro .ztr", f.Rel)
	}
	return data, nil
}

// ReadSource le os bytes de um arquivo de unidade: em data/partN ou no
// white bin (quando o arquivo não foi extraído para data/).
func ReadSource(src UnitSource) ([]byte, error) {
	if src.IsDisk() {
		return os.ReadFile(filepath.Join(src.Root, filepath.FromSlash(src.Rel)))
	}
	if src.Bin != "" {
		return whitebin.ExtractBytes(src.Bin, src.VPath, src.Coll)
	}
	return nil, fmt.Errorf("sem fonte para %s", src.Rel)
}
