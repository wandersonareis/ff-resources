package service

// INSERÇÃO DE MODS NO FILELIST (append + reapontamento).
//
// O formato filelist não cria entradas novas: só reaponta o endereço de um
// caminho que já existe. A inserção, portanto:
//
//  1. acha TODOS os índices que declaram o caminho;
//  2. para cada um, grava o blob no FIM do white bin que ele descreve,
//     alinhado a 2048 bytes, e atualiza os tamanhos/posição das ocorrências;
//  3. reconstrói o índice e, se ele for EMBUTIDO, grava a nova versão no
//     white bin do pai e reaponta a entrada do pai — subindo a cadeia até o
//     filelist de disco;
//  4. grava os filelists de disco de forma atômica, por último.
//
// Ordem de gravação: primeiro os white bins (append), depois os filelists.
// Um crash entre os dois deixa dados órfãos no fim do bin, mas os ponteiros
// antigos continuam válidos — o jogo não quebra.
//
// O manifesto em mods/partN/filelist-inserts.json guarda o tamanho ORIGINAL
// de cada bin e uma cópia de cada filelist antes da primeira alteração. É o
// que permite "reverter inserções". Nada em data/ é tocado.

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	fl "ffresources/internal/families/white/codecs/filelist"
	"ffresources/internal/families/white/codecs/whitebin"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/dto"
	logging "ffresources/internal/shared/logging"
)

// InsertItem é UM arquivo a inserir no filelist: o caminho interno (relativo
// a data/partN, o mesmo de mods/) e os bytes do binário já convertido.
type InsertItem struct {
	RelPath string `json:"rel_path"`
	Data    []byte `json:"data"`
}

// InsertOutcome descreve o que aconteceu com um InsertItem.
type InsertOutcome struct {
	RelPath    string   `json:"rel_path"`
	Inserted   bool     `json:"inserted"`
	Reason     string   `json:"reason,omitempty"`
	Targets    int      `json:"targets"`
	Entries    int      `json:"entries"`
	Binaries   []string `json:"binaries,omitempty"`
	PosSectors uint64   `json:"pos_sectors,omitempty"`
	UncmpSize  uint32   `json:"uncmp_size,omitempty"`
	CmpSize    uint32   `json:"cmp_size,omitempty"`
}

// insertTargetRef é um índice que declara o caminho + o caminho ORIGINAL
// (com o casing do arquivo) que deve ser reapontado.
type insertTargetRef struct {
	idx      *FilelistIndex
	origPath string
}

// insertAppendOp é uma gravação planejada.
type insertAppendOp struct {
	bin     string
	offset  int64
	payload []byte
	uncmp   uint32
	cmp     uint32
	verify  bool
	source  []byte
}

// insertPlan acumula o lote inteiro antes de tocar no disco.
type insertPlan struct {
	lens      map[string]int64
	snapshots map[string]int64
	appends   []insertAppendOp
	writes    map[string][]byte
	order     []string
}

func newInsertPlan() *insertPlan {
	return &insertPlan{
		lens:      map[string]int64{},
		snapshots: map[string]int64{},
		writes:    map[string][]byte{},
	}
}

// containerLen devolve o tamanho ATUAL do bin e guarda o snapshot original
// (o valor usado pelo rollback).
func (p *insertPlan) containerLen(bin string) (int64, error) {
	if n, ok := p.lens[bin]; ok {
		return n, nil
	}
	st, err := os.Stat(bin)
	if err != nil {
		return 0, err
	}
	p.lens[bin] = st.Size()
	p.snapshots[bin] = st.Size()
	return st.Size(), nil
}

// appendBlob planeja um append: calcula o alinhamento, prepara a compressão
// e atualiza o tamanho virtual do container.
func (p *insertPlan) appendBlob(bin string, data []byte, compress, verify bool) (posSectors uint64, uncmp, cmp uint32, err error) {
	cur, err := p.containerLen(bin)
	if err != nil {
		return 0, 0, 0, err
	}
	aligned, pos := fl.PlanAppend(cur)
	prepared, err := whitebin.PrepareBlob(data, compress)
	if err != nil {
		return 0, 0, 0, err
	}
	op := insertAppendOp{
		bin:     bin,
		offset:  aligned,
		payload: prepared.Payload,
		uncmp:   prepared.UncmpSize,
		cmp:     prepared.CmpSize,
		verify:  verify,
	}
	if verify {
		op.source = data
	}
	p.appends = append(p.appends, op)
	p.lens[bin] = aligned + int64(len(prepared.Payload))
	return pos, prepared.UncmpSize, prepared.CmpSize, nil
}

// addWrite guarda os bytes finais de um filelist de disco.
func (p *insertPlan) addWrite(path string, data []byte) {
	if _, ok := p.writes[path]; !ok {
		p.order = append(p.order, path)
	}
	p.writes[path] = data
}

// normalizeInsertRel normaliza um caminho interno para casar com os índices:
// minúsculas, barras "/" e sem barra inicial. Mantém o caminho original nos
// registros; só a comparação é normalizada.
func normalizeInsertRel(rel string) string {
	rel = strings.ReplaceAll(strings.TrimSpace(rel), "\\", "/")
	rel = strings.TrimPrefix(rel, "/")
	return strings.ToLower(rel)
}

// findInsertTargets devolve os índices que declaram o caminho. O mesmo
// caminho pode aparecer em mais de um índice (global e de zona) e repetido
// dentro do mesmo índice — o reapontamento cobre todas as ocorrências.
func findInsertTargets(p parts.Part, rel string) []insertTargetRef {
	norm := normalizeInsertRel(rel)
	if norm == "" {
		return nil
	}
	disc := DiscoverPart(p)
	out := make([]insertTargetRef, 0, 2)
	for _, idx := range disc.Indexes {
		if idx == nil || idx.Coll == nil || idx.Container == "" {
			continue
		}
		// Índice embutido sem a cadeia do pai não tem como ser gravado.
		if idx.Level > 0 && (idx.ParentIndex == nil || idx.IndexBin == "") {
			continue
		}
		zone := ZoneOfIndex(idx)
	entries:
		for ci := range idx.Coll.Chunks {
			for ei := range idx.Coll.Chunks[ci].Entries {
				entryPath := fl.EntryPath(idx.Coll.Chunks[ci].Entries[ei].FileInfo)
				if normalizeInsertRel(FilelistDataPath(zone, entryPath)) != norm {
					continue
				}
				out = append(out, insertTargetRef{idx: idx, origPath: entryPath})
				break entries
			}
		}
	}
	return out
}

var (
	partWriteMu    sync.Mutex
	partWriteLocks = map[string]*sync.Mutex{}
)

// lockPartWrite serializa as inserções de uma part (mesmo que a UI dispare
// duas importações ao mesmo tempo).
func lockPartWrite(part string) func() {
	partWriteMu.Lock()
	mu := partWriteLocks[part]
	if mu == nil {
		mu = &sync.Mutex{}
		partWriteLocks[part] = mu
	}
	partWriteMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// InsertBlobs insere um LOTE no filelist. Os itens são agrupados por índice,
// os appends e os filelists são planejados antes de qualquer gravação, e os
// filelists de disco são gravados por último.
//
// Um item que não existe em nenhum índice volta com Inserted=false e Reason
// (o chamador decide se é aviso ou erro). Erro de I/O aborta o lote.
func (s *Service) InsertBlobs(part string, items []InsertItem) ([]InsertOutcome, error) {
	part = config.NormalizePart(part)
	out := make([]InsertOutcome, len(items))
	for i := range out {
		out[i].RelPath = items[i].RelPath
	}
	if len(items) == 0 {
		return out, nil
	}

	unlock := lockPartWrite(part)
	defer unlock()

	// Só marca Inserted=true no retorno de SUCESSO: falha de planejamento,
	// commit ou filelist desfaz o lote (bins truncados/filelists restaurados)
	// e o chamador não pode achar que o item entrou.
	committedOK := false
	defer func() {
		if committedOK {
			return
		}
		for i := range out {
			out[i].Inserted = false
		}
	}()

	p := parts.Get(part)
	if config.GameRoot(p.ID()) == "" {
		return out, fmt.Errorf("filelist: instalação do jogo não configurada para %s", part)
	}

	type work struct {
		item int
		ref  insertTargetRef
	}
	var works []work
	for i, it := range items {
		if normalizeInsertRel(it.RelPath) == "" {
			out[i].Reason = "caminho inválido"
			continue
		}
		if uint64(len(it.Data)) > 0xFFFFFFFF {
			out[i].Reason = "arquivo grande demais para o filelist"
			continue
		}
		targets := findInsertTargets(p, it.RelPath)
		if len(targets) == 0 {
			out[i].Reason = "não indexado em nenhum filelist"
			continue
		}
		for _, t := range targets {
			works = append(works, work{item: i, ref: t})
		}
	}
	if len(works) == 0 {
		return out, nil
	}

	plan := newInsertPlan()
	touched := map[*FilelistIndex]bool{}

	// 1) appends dos DADOS, por índice (cada índice tem o seu container).
	for _, w := range works {
		it := items[w.item]
		idx := w.ref.idx
		found, compressed := idx.Coll.PathCompressionFold(w.ref.origPath)
		if !found {
			out[w.item].Reason = "caminho não encontrado no índice"
			continue
		}
		pos, uncmp, cmp, err := plan.appendBlob(idx.Container, it.Data, compressed, true)
		if err != nil {
			return out, fmt.Errorf("planejando %s: %w", it.RelPath, err)
		}
		n, err := idx.Coll.SetAddressByPathFold(w.ref.origPath, pos, uncmp, cmp)
		if err != nil {
			return out, fmt.Errorf("reapontando %s: %w", it.RelPath, err)
		}
		out[w.item].Inserted = true
		out[w.item].Targets++
		out[w.item].Entries += n
		out[w.item].PosSectors = pos
		out[w.item].UncmpSize = uncmp
		out[w.item].CmpSize = cmp
		out[w.item].Binaries = appendUnique(out[w.item].Binaries, idx.Container)
		touched[idx] = true
	}

	if len(touched) == 0 {
		return out, nil
	}

	// 2) cadeia de índices: do mais profundo para o de disco. Cada índice é
	// reconstruído UMA vez, já com todas as atualizações dos filhos.
	all := map[*FilelistIndex]bool{}
	for idx := range touched {
		for cur := idx; cur != nil; cur = cur.ParentIndex {
			all[cur] = true
		}
	}
	list := make([]*FilelistIndex, 0, len(all))
	for idx := range all {
		list = append(list, idx)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Level != list[j].Level {
			return list[i].Level > list[j].Level
		}
		return list[i].Path < list[j].Path
	})

	for _, idx := range list {
		data, err := idx.Coll.Build()
		if err != nil {
			return out, fmt.Errorf("reconstruindo %s: %w", IndexNameOf(idx), err)
		}
		// Sanidade: o índice precisa voltar a ser parseável.
		if _, err := fl.Parse(data, p.GameCode()); err != nil {
			return out, fmt.Errorf("índice reconstruído inválido (%s): %w", IndexNameOf(idx), err)
		}
		if idx.Level == 0 || idx.IndexBin == "" {
			plan.addWrite(idx.Path, data)
			continue
		}
		parent := idx.ParentIndex
		if parent == nil {
			return out, fmt.Errorf("índice embutido sem pai: %s", idx.Path)
		}
		_, compressed := parent.Coll.PathCompressionFold(idx.Virtual)
		pos, uncmp, cmp, err := plan.appendBlob(idx.IndexBin, data, compressed, false)
		if err != nil {
			return out, fmt.Errorf("gravando índice %s no pai: %w", IndexNameOf(idx), err)
		}
		if _, err := parent.Coll.SetAddressByPathFold(idx.Virtual, pos, uncmp, cmp); err != nil {
			return out, fmt.Errorf("reapontando índice %s no pai: %w", IndexNameOf(idx), err)
		}
	}

	// 3) confere que ninguém mexeu nos bins desde o planejamento.
	for bin, orig := range plan.snapshots {
		st, err := os.Stat(bin)
		if err != nil {
			return out, fmt.Errorf("white bin %s: %w", filepath.Base(bin), err)
		}
		if st.Size() != orig {
			return out, fmt.Errorf(
				"white bin %s mudou durante a importação (tamanho %d, esperado %d) — tente de novo",
				filepath.Base(bin), st.Size(), orig)
		}
	}

	// 4) grava os bins (append), verificando o que os dados apontam.
	committed := map[string]bool{}
	for _, op := range plan.appends {
		if err := whitebin.AppendPayload(op.bin, op.offset, op.payload); err != nil {
			rollbackBins(plan.snapshots, committed)
			return out, fmt.Errorf("gravando em %s: %w", filepath.Base(op.bin), err)
		}
		committed[op.bin] = true
	}
	for _, op := range plan.appends {
		if !op.verify {
			continue
		}
		got, err := whitebin.ExtractAt(op.bin, uint64(op.offset/2048), op.uncmp, op.cmp)
		if err != nil {
			rollbackBins(plan.snapshots, committed)
			return out, fmt.Errorf("verificando gravação em %s: %w", filepath.Base(op.bin), err)
		}
		if !bytes.Equal(got, op.source) {
			rollbackBins(plan.snapshots, committed)
			return out, fmt.Errorf("verificação pós-gravação falhou em %s", filepath.Base(op.bin))
		}
	}

	// 5) grava os filelists de disco, com backup da versão anterior.
	var writtenFiles []writtenFilelist
	for _, path := range plan.order {
		orig, err := os.ReadFile(path)
		if err != nil {
			restoreWritten(writtenFiles)
			rollbackBins(plan.snapshots, committed)
			return out, fmt.Errorf("lendo filelist %s: %w", filepath.Base(path), err)
		}
		backup, err := ensureFilelistBackup(part, path, orig)
		if err != nil {
			restoreWritten(writtenFiles)
			rollbackBins(plan.snapshots, committed)
			return out, err
		}
		if err := writeFileAtomic(path, plan.writes[path]); err != nil {
			restoreWritten(writtenFiles)
			rollbackBins(plan.snapshots, committed)
			return out, fmt.Errorf("gravando filelist %s: %w", filepath.Base(path), err)
		}
		writtenFiles = append(writtenFiles, writtenFilelist{
			path:     path,
			backup:   backup,
			original: orig,
		})
	}

	// 6) journal de rollback. Falha aqui não desfaz a inserção (os dados já
	// estão no lugar); só registra que o "reverter" não terá a trilha.
	recordInsertManifest(part, plan, out, writtenFiles)

	committedOK = true
	invalidateAfterFilelistWrite(part)
	return out, nil
}

// RestoreFilelistInserts desfaz as inserções da part: restaura os filelists
// do backup e trunca os white bins de volta ao tamanho original.
//
// A restauração dos filelists vem PRIMEIRO: se o truncamento falhar, o índice
// novo continua apontando para dados que ainda existem. Se truncar primeiro,
// um filelist novo apontaria para regiões inexistentes.
func (s *Service) RestoreFilelistInserts(part string) (dto.FilelistRestoreResult, error) {
	part = config.NormalizePart(part)
	res := dto.FilelistRestoreResult{Warnings: []string{}}

	unlock := lockPartWrite(part)
	defer unlock()

	man, path, err := loadInsertManifest(part)
	if err != nil {
		return res, err
	}
	if man == nil {
		return res, nil
	}

	// 1) filelists de volta ao conteúdo original.
	paths := make([]string, 0, len(man.Filelists))
	for p := range man.Filelists {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	filelistFailed := false
	for _, p := range paths {
		backup := man.Filelists[p]
		data, rerr := os.ReadFile(backup)
		if rerr != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("backup ausente para %s: %v", filepath.Base(p), rerr))
			filelistFailed = true
			continue
		}
		if werr := writeFileAtomic(p, data); werr != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("restaurando %s: %v", filepath.Base(p), werr))
			filelistFailed = true
			continue
		}
		res.Filelists++
	}

	// 2) bins de volta ao tamanho original. Se QUALQUER filelist não voltou,
	// o truncamento é CANCELADO: um índice novo apontando para um setor que
	// deixou de existir quebraria o jogo.
	if filelistFailed {
		res.Warnings = append(res.Warnings,
			"white bins NÃO foram truncados porque um ou mais filelists não puderam ser restaurados")
	} else {
		bins := make([]string, 0, len(man.Bins))
		for b := range man.Bins {
			bins = append(bins, b)
		}
		sort.Strings(bins)
		for _, bin := range bins {
			orig := man.Bins[bin]
			st, serr := os.Stat(bin)
			if serr != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", filepath.Base(bin), serr))
				continue
			}
			switch {
			case st.Size() == orig:
				// já está no tamanho original
			case st.Size() < orig:
				res.Warnings = append(res.Warnings, fmt.Sprintf(
					"%s está menor que o original (%d < %d) — não truncado",
					filepath.Base(bin), st.Size(), orig))
			default:
				if terr := os.Truncate(bin, orig); terr != nil {
					res.Warnings = append(res.Warnings, fmt.Sprintf("truncando %s: %v", filepath.Base(bin), terr))
					continue
				}
				res.Bins++
				res.Bytes += st.Size() - orig
			}
		}
	}

	// 3) limpa o journal e os backups SÓ quando tudo foi revertido. Com
	// avisos (ex.: bin menor que o original), o manifesto fica para uma nova
	// tentativa.
	if len(res.Warnings) == 0 {
		for _, backup := range man.Filelists {
			_ = os.Remove(backup)
		}
		if rerr := os.Remove(path); rerr != nil && !os.IsNotExist(rerr) {
			res.Warnings = append(res.Warnings, fmt.Sprintf("removendo manifesto: %v", rerr))
		}
	}
	res.Applied = res.Filelists + res.Bins

	invalidateAfterFilelistWrite(part)
	return res, nil
}

// rollbackBins trunca os bins já gravados de volta ao tamanho original.
func rollbackBins(snapshots map[string]int64, committed map[string]bool) {
	for bin := range committed {
		orig, ok := snapshots[bin]
		if !ok {
			continue
		}
		if err := os.Truncate(bin, orig); err != nil {
			logging.Error("filelist: rollback de %s falhou: %v", bin, err)
		}
	}
}

// writtenFilelist é um filelist já gravado nesta operação + o backup pristine
// e o conteúdo imediatamente anterior (para o rollback desta operação não
// desfazer inserções de execuções anteriores).
type writtenFilelist struct {
	path     string
	backup   string
	original []byte
}

// restoreWritten devolve os filelists já gravados ao conteúdo anterior DESTA
// operação.
func restoreWritten(files []writtenFilelist) {
	for _, f := range files {
		if f.original == nil {
			continue
		}
		_ = writeFileAtomic(f.path, f.original)
	}
}

// appendUnique adiciona s em out se ainda não estiver lá.
func appendUnique(out []string, s string) []string {
	for _, v := range out {
		if strings.EqualFold(v, s) {
			return out
		}
	}
	return append(out, s)
}

// writeFileAtomic escreve tmp + rename no mesmo diretório.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".fflist-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ensureFilelistBackup cria (uma vez) o backup do filelist antes de
// sobrescrevê-lo; se o backup já existe, devolve o caminho sem tocar nele
// (o primeiro backup é o pristine, o que interessa para o reverter).
func ensureFilelistBackup(part, path string, orig []byte) (string, error) {
	mods, err := config.PartRoot("mods", part)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(mods, insertBackupDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("criando pasta de backup do filelist: %w", err)
	}
	sum := sha1.Sum([]byte(strings.ToLower(filepath.Clean(path))))
	name := hex.EncodeToString(sum[:6]) + "-" + filepath.Base(path) + ".bak"
	backup := filepath.Join(dir, name)
	if _, err := os.Stat(backup); err == nil {
		return backup, nil
	}
	if err := writeFileAtomic(backup, orig); err != nil {
		return "", fmt.Errorf("fazendo backup de %s: %w", filepath.Base(path), err)
	}
	return backup, nil
}

// invalidateAfterFilelistWrite limpa TODOS os caches que dependem do
// filelist/white bin: árvore, índices embutidos, fontes, coleção de texto e
// índices de imagem. Sem isso a UI continuaria lendo as posições antigas.
func invalidateAfterFilelistWrite(part string) {
	filelistMu.Lock()
	filelistCache = map[string]*fl.Collection{}
	filelistMu.Unlock()

	embeddedMu.Lock()
	embeddedCache = map[string]*fl.Collection{}
	embeddedMu.Unlock()

	binsMu.Lock()
	for k := range binsCache {
		if strings.HasPrefix(k, part+"|") {
			delete(binsCache, k)
		}
	}
	binsMu.Unlock()

	sourcesMu.Lock()
	for k := range sourcesCache {
		if strings.HasPrefix(k, part+"|") {
			delete(sourcesCache, k)
		}
	}
	sourcesMu.Unlock()

	InvalidateImageContainerIndex(part)
	InvalidateImageFileIndex(part)
	invalidateCollection(part)
}

// ---- manifesto de rollback ----

const (
	insertManifestName  = "filelist-inserts.json"
	insertBackupDirName = ".filelist-backup"
)

type insertManifest struct {
	Version   int               `json:"version"`
	Bins      map[string]int64  `json:"bins"`
	Filelists map[string]string `json:"filelists"`
	Inserts   []insertRecord    `json:"inserts"`
}

type insertRecord struct {
	RelPath    string `json:"rel_path"`
	Bin        string `json:"bin"`
	PosSectors uint64 `json:"pos_sectors"`
	UncmpSize  uint32 `json:"uncmp_size"`
	CmpSize    uint32 `json:"cmp_size"`
	Entries    int    `json:"entries"`
	At         string `json:"at"`
}

func insertManifestPath(part string) (string, error) {
	mods, err := config.PartRoot("mods", part)
	if err != nil {
		return "", err
	}
	return filepath.Join(mods, insertManifestName), nil
}

func loadInsertManifest(part string) (*insertManifest, string, error) {
	path, err := insertManifestPath(part)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, nil
		}
		return nil, path, err
	}
	var man insertManifest
	if err := json.Unmarshal(data, &man); err != nil {
		return nil, path, fmt.Errorf("manifesto de inserções ilegível: %w", err)
	}
	if man.Bins == nil {
		man.Bins = map[string]int64{}
	}
	if man.Filelists == nil {
		man.Filelists = map[string]string{}
	}
	return &man, path, nil
}

// recordInsertManifest grava/atualiza o journal. O snapshot dos bins e o
// backup dos filelists são mantidos da PRIMEIRA inserção: "reverter" volta ao
// estado pristine, não ao estado da última importação.
func recordInsertManifest(part string, plan *insertPlan, out []InsertOutcome, files []writtenFilelist) {
	man, path, err := loadInsertManifest(part)
	if err != nil {
		logging.Error("filelist: manifesto de inserções: %v", err)
		return
	}
	if man == nil {
		man = &insertManifest{
			Version:   1,
			Bins:      map[string]int64{},
			Filelists: map[string]string{},
		}
	}
	for bin, orig := range plan.snapshots {
		if _, ok := man.Bins[bin]; !ok {
			man.Bins[bin] = orig
		}
	}
	for _, f := range files {
		if _, ok := man.Filelists[f.path]; !ok {
			man.Filelists[f.path] = f.backup
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, o := range out {
		if !o.Inserted {
			continue
		}
		bin := ""
		if len(o.Binaries) > 0 {
			bin = o.Binaries[0]
		}
		man.Inserts = append(man.Inserts, insertRecord{
			RelPath:    o.RelPath,
			Bin:        bin,
			PosSectors: o.PosSectors,
			UncmpSize:  o.UncmpSize,
			CmpSize:    o.CmpSize,
			Entries:    o.Entries,
			At:         now,
		})
	}
	data, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		logging.Error("filelist: serializando manifesto: %v", err)
		return
	}
	if err := writeFileAtomic(path, data); err != nil {
		logging.Error("filelist: gravando manifesto: %v", err)
	}
}
