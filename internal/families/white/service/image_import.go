package service

// IMPORT de imagens do FF13.
//
// Dois fluxos, ambos gravando SÓ em mods/partN:
//
//	LOTE — mods/translated/images/<part>/<container-rel>/<registro>.dds
//	       o diretório é o caminho do container relativo a data/partN (o
//	       MESMO que o export usa) e o arquivo tem o nome do registro como
//	       a árvore mostra (nome.ext no WPD, N.tipo no TRB). O lookup é
//	       exato no filelist — sem busca por basename e sem ambiguidade.
//
//	AVULSO — .dds escolhido em qualquer lugar do disco para a imagem ABERTA.
//	       O caminho/nome do arquivo é irrelevante: o alvo é a ImageRef da
//	       imagem em exibição.
//
// As duas rotas compartilham a validação (dimensões, formato DX e mipmaps
// iguais) e o patch: como o payload tem o mesmo tamanho, os mips novos são
// copiados para os offsets da tabela do .imgb — nenhum repack de container.

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	imgbcore "ffresources/internal/families/white/codecs/imgb/core"
	"ffresources/internal/families/white/imagery"
	"ffresources/internal/families/white/profile/common"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/dto"
	loggingService "ffresources/internal/shared/logging"
	"ffresources/internal/shared/progress"
)

// ddsHeaderSize é o tamanho do header DDS clássico (4 + 124).
const ddsHeaderSize = 128

// translatedImagesSubdir é a pasta, dentro de mods/, das imagens traduzidas.
const translatedImagesSubdir = "translated/images"

// TranslatedImagesRoot devolve a raiz de imagens traduzidas da part:
// <mods>/translated/images/<part>.
func TranslatedImagesRoot(part string) string {
	part = config.NormalizePart(part)
	mods, err := config.DataRoot("mods")
	if err != nil {
		return ""
	}
	return filepath.Join(mods, filepath.FromSlash(translatedImagesSubdir), part)
}

// imageContainerIndex mapeia o caminho canônico (minúsculo) do container para
// o caminho como o filelist o conhece. É o lookup do diretório do lote.
//
// O índice cobre TODOS os filelists da part — os em disco (inclusive a cópia
// extraída em data/partN/filelist) e os embutidos nos white bins. Sem ele o
// import não saberia que "vfx/.../veffs.us.win32.xfv" é um arquivo real do
// jogo: o mesmo basename aparece em dezenas de pastas diferentes.
var (
	imageContainersMu    sync.Mutex
	imageContainersCache = map[string]map[string]string{}
)

// ImageContainerIndexFor devolve o mapa caminho (minúsculo) → caminho canônico.
func ImageContainerIndexFor(part string) map[string]string {
	part = config.NormalizePart(part)
	dataRoot, _ := config.DataRoot("data")
	key := part + "|" + config.GameRoot(part) + "|" + dataRoot
	imageContainersMu.Lock()
	if m, ok := imageContainersCache[key]; ok {
		imageContainersMu.Unlock()
		return m
	}
	imageContainersMu.Unlock()

	m := buildImageContainerIndex(part)

	imageContainersMu.Lock()
	imageContainersCache[key] = m
	imageContainersMu.Unlock()
	return m
}

// InvalidateImageContainerIndex limpa o índice de containers da part.
func InvalidateImageContainerIndex(part string) {
	part = config.NormalizePart(part)
	imageContainersMu.Lock()
	for k := range imageContainersCache {
		if strings.HasPrefix(k, part+"|") {
			delete(imageContainersCache, k)
		}
	}
	imageContainersMu.Unlock()
}

func buildImageContainerIndex(part string) map[string]string {
	p := parts.Get(part)
	out := map[string]string{}
	add := func(zone, entryPath string) {
		rel := strings.TrimPrefix(filepath.ToSlash(FilelistDataPath(zone, entryPath)), "/")
		if rel == "" {
			return
		}
		kind, container := ImageDataKind(rel)
		if kind == "" || !container {
			return
		}
		key := strings.ToLower(rel)
		if _, exists := out[key]; !exists {
			out[key] = rel
		}
	}

	disc := DiscoverPart(p)
	indexed := map[string]bool{}
	for _, idx := range disc.Indexes {
		if idx == nil || idx.Coll == nil {
			continue
		}
		if idx.Path != "" {
			indexed[strings.ToLower(idx.Path)] = true
		}
		zone := ZoneOfIndex(idx)
		for _, e := range IndexEntries(p, idx, nil) {
			add(zone, e.Path)
		}
	}
	// Raízes que não entraram na cadeia de white bins (ex.: índice extraído
	// em data/partN/filelist sem o bin par): lê direto do disco.
	for _, root := range disc.Roots {
		if root.Path == "" || indexed[strings.ToLower(root.Path)] {
			continue
		}
		idx, err := EmbeddedRoot(p, root.Path)
		if err != nil || idx == nil || idx.Coll == nil {
			continue
		}
		indexed[strings.ToLower(root.Path)] = true
		zone := ZoneOfIndex(idx)
		for _, e := range IndexEntries(p, idx, nil) {
			add(zone, e.Path)
		}
	}
	return out
}

// imageImportTarget é o registro que receberá os bytes do DDS.
type imageImportTarget struct {
	ref          ImageRef
	containerRel string
	payloadRel   string
	payload      []byte
	gtex         *imgbcore.GTEX
	mips         []imgbcore.MipInfo
	platform     imgbcore.Platform
	swizzled     bool
}

// resolveContainerRel valida o diretório do lote contra o filelist.
func resolveContainerRel(part, dir string) (string, error) {
	dir = normalizeTranslatedRel(dir)
	if dir == "" || dir == "." {
		return "", fmt.Errorf("arquivo solto: o .dds precisa estar dentro da pasta do container")
	}
	idx := ImageContainerIndexFor(part)
	rel, ok := idx[strings.ToLower(dir)]
	if !ok {
		return "", fmt.Errorf(
			"container %q não encontrado no filelist da part %s (o diretório precisa ser o caminho do container relativo a data/%s, ex.: vfx/menu/resident/resident/veffs.us.win32.xfv)",
			dir, part, part)
	}
	return rel, nil
}

// imageNodeMatches diz se o nó do container é o registro pedido. O nome do
// arquivo é o nome exibido na árvore (nome.ext no WPD, N.tipo no TRB); para
// TRB também aceitamos só o índice N.
func imageNodeMatches(node common.ImageNode, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if strings.EqualFold(node.Name, name) || strings.EqualFold(sanitizeImageName(node.Name), name) {
		return true
	}
	if strings.HasPrefix(node.Path, "trb:") {
		idx := strings.TrimPrefix(node.Path, "trb:")
		if idx == name {
			return true
		}
		if strings.HasPrefix(strings.ToLower(node.Name), strings.ToLower(name)+".") {
			return true
		}
	}
	return false
}

// resolveImageImportTarget resolve o container (pelo diretório) e o registro
// (pelo nome do arquivo) e lê o header GTEX + a tabela de mips + o payload.
func resolveImageImportTarget(part, containerDir, recordName string) (*imageImportTarget, error) {
	containerRel, err := resolveContainerRel(part, containerDir)
	if err != nil {
		return nil, err
	}
	containerRef := EncodeImageRef(ImageRef{
		Source:   "data",
		Path:     containerRel,
		Platform: ImagePlatformOf(containerRel),
	})
	nodes, err := (Service{}).ListImageContainer(part, containerRef)
	if err != nil {
		return nil, err
	}
	var found *common.ImageNode
	for i := range nodes {
		if imageNodeMatches(nodes[i], recordName) {
			found = &nodes[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf(
			"a imagem %q não existe no container %s — o arquivo precisa ter o nome original do registro (nome.ext no WPD, N.tipo no TRB) seguido de .dds",
			recordName, containerRel)
	}
	ref, err := DecodeImageRef(found.ImageRef)
	if err != nil {
		return nil, err
	}
	return imageImportTargetFromRef(part, ref)
}

// imageImportTargetFromRef monta o alvo a partir de uma ImageRef que já
// aponta para o registro (usado pelo import avulso, onde a imagem aberta é o
// alvo e o nome do arquivo escolhido não importa).
func imageImportTargetFromRef(part string, ref ImageRef) (*imageImportTarget, error) {
	if ref.Entry == "" {
		return nil, fmt.Errorf("a referência não aponta para uma imagem dentro de container")
	}
	sec, table, err := ImageSection(part, ref)
	if err != nil {
		return nil, err
	}
	gtex, err := imgbcore.SectionGTEX(sec)
	if err != nil {
		return nil, fmt.Errorf("lendo GTEX: %w", err)
	}
	mips, err := imgbcore.ParseMipInfoTable(sec, gtex)
	if err != nil {
		return nil, fmt.Errorf("lendo mipmaps: %w", err)
	}
	if len(mips) == 0 {
		return nil, fmt.Errorf("textura sem mipmaps")
	}
	payloadFile, err := ResolveImagePayload(part, ref, table.Rel)
	if err != nil {
		return nil, err
	}
	payload, err := payloadFile.Read()
	if err != nil {
		return nil, err
	}
	platform := imgbcore.PlatformWin32
	switch strings.ToLower(ref.Platform) {
	case "ps3":
		platform = imgbcore.PlatformPS3
	case "x360":
		platform = imgbcore.PlatformX360
	}
	swizzled := platform == imgbcore.PlatformPS3 &&
		gtex.Format == imgbcore.GTEXFormatRGBA &&
		gtex.Type == imgbcore.GTEXTypeClassicConsole
	return &imageImportTarget{
		ref:          ref,
		containerRel: canonicalImageRel(part, ref),
		payloadRel:   payloadFile.Rel,
		payload:      payload,
		gtex:         gtex,
		mips:         mips,
		platform:     platform,
		swizzled:     swizzled,
	}, nil
}

// gtexDDSFormat traduz o formato GTEX para o vocabulário do header DDS.
func gtexDDSFormat(f imgbcore.GTEXFormat) imagery.DDSFormat {
	switch f {
	case imgbcore.GTEXFormatDXT1, imgbcore.GTEXFormatLegacyDXT1:
		return imagery.DDSFormatDXT1
	case imgbcore.GTEXFormatDXT3, imgbcore.GTEXFormatLegacyDXT3:
		return imagery.DDSFormatDXT3
	case imgbcore.GTEXFormatDXT5, imgbcore.GTEXFormatLegacyDXT5:
		return imagery.DDSFormatDXT5
	case imgbcore.GTEXFormatRGB:
		return imagery.DDSFormatRGB
	case imgbcore.GTEXFormatRGBA:
		return imagery.DDSFormatRGBA
	default:
		return imagery.DDSFormatUnknown
	}
}

// imageMismatchError formata as divergências encontradas.
func imageMismatchError(diffs []string) error {
	return fmt.Errorf("incompatível — %s", strings.Join(diffs, "; "))
}

// prepareImageMips valida o DDS contra o GTEX/tabela do container e devolve
// os bytes de cada mip no layout que o .imgb espera.
func prepareImageMips(target *imageImportTarget, dds []byte) ([][]byte, error) {
	info, err := imagery.ParseDDSHeader(dds)
	if err != nil {
		return nil, err
	}
	var diffs []string
	if info.Width != uint32(target.gtex.Width) || info.Height != uint32(target.gtex.Height) {
		diffs = append(diffs, fmt.Sprintf("dimensões: container %dx%d, arquivo %dx%d",
			target.gtex.Width, target.gtex.Height, info.Width, info.Height))
	}
	if want := gtexDDSFormat(target.gtex.Format); info.Format != want {
		diffs = append(diffs, fmt.Sprintf("formato: container %s, arquivo %s",
			imgbcore.FormatName(target.gtex.Format), info.Format))
	}
	wantMips := uint32(target.gtex.MipCount)
	mipsMatch := info.MipMapCount == wantMips
	// O export do app grava DwMipMapCount com o total de LAYERS do cubemap
	// (6 × mips por face); aceitamos também essa contagem para o round-trip.
	if !mipsMatch && target.gtex.Type.IsCubemap() && info.MipMapCount == wantMips*6 {
		mipsMatch = true
	}
	if !mipsMatch {
		diffs = append(diffs, fmt.Sprintf("mipmaps: container %d, arquivo %d",
			target.gtex.MipCount, info.MipMapCount))
	}
	ddsCubemap := info.Caps2&0x200 != 0
	if ddsCubemap != target.gtex.Type.IsCubemap() {
		ddsTipo := "2D"
		if ddsCubemap {
			ddsTipo = "cubemap"
		}
		contTipo := "2D"
		if target.gtex.Type.IsCubemap() {
			contTipo = "cubemap"
		}
		diffs = append(diffs, fmt.Sprintf("tipo: container %s, arquivo %s", contTipo, ddsTipo))
	}
	if len(diffs) > 0 {
		return nil, imageMismatchError(diffs)
	}
	if target.swizzled {
		return nil, fmt.Errorf("textura PS3 swizzled (A8R8G8B8 classic-console) não é suportada no import")
	}

	expected := 0
	for _, m := range target.mips {
		expected += int(m.Size)
	}
	payload := dds[ddsHeaderSize:]
	if len(payload) != expected {
		return nil, imageMismatchError([]string{fmt.Sprintf(
			"tamanho do payload: container %d bytes, arquivo %d bytes", expected, len(payload))})
	}
	chunks := make([][]byte, len(target.mips))
	acc := 0
	for i, m := range target.mips {
		size := int(m.Size)
		chunk := make([]byte, size)
		copy(chunk, payload[acc:acc+size])
		acc += size
		// BuildDDS aplica ColorAsBGRA nos formatos não comprimidos do PS3;
		// a involução desfaz exatamente na importação.
		if target.platform == imgbcore.PlatformPS3 && !target.gtex.Format.IsCompressed() {
			chunk = imgbcore.ColorAsBGRA(chunk)
		}
		chunks[i] = chunk
	}
	return chunks, nil
}

// payloadMatches diz se os bytes novos são idênticos ao payload efetivo
// (mods/, quando existe; senão o pristine).
func payloadMatches(target *imageImportTarget, chunks [][]byte) bool {
	for i, m := range target.mips {
		start := int(m.Offset)
		end := start + int(m.Size)
		if start < 0 || end > len(target.payload) {
			return false
		}
		if !bytes.Equal(target.payload[start:end], chunks[i]) {
			return false
		}
	}
	return true
}

// patchPayload devolve uma CÓPIA do payload com os mips novos nas posições
// da tabela. O container não é tocado: o header GTEX e a tabela de mips não
// mudam (mesmas dimensões, formato e mipmaps).
//
// base pode ser o payload pristine ou o acumulado do lote (várias imagens no
// MESMO .imgb): cada registro só sobrescreve os seus mips.
func patchPayload(base []byte, mips []imgbcore.MipInfo, chunks [][]byte) ([]byte, error) {
	patched := make([]byte, len(base))
	copy(patched, base)
	for i, m := range mips {
		start := int(m.Offset)
		end := start + int(m.Size)
		if i >= len(chunks) {
			return nil, fmt.Errorf("mip %d sem bytes para gravar", i)
		}
		if start < 0 || end > len(patched) {
			return nil, fmt.Errorf("mip %d fora do payload (offset %d, tamanho %d, payload %d)",
				i, start, m.Size, len(patched))
		}
		copy(patched[start:end], chunks[i])
	}
	return patched, nil
}

// normalizeImageImportOptions garante que uma importação sempre tem destino.
// A UI manda os dois flags explicitamente; chamadas antigas (sem opções)
// continuam gravando em mods/, o comportamento de antes.
func normalizeImageImportOptions(opts dto.ImageImportOptions) dto.ImageImportOptions {
	if !opts.WriteMods && !opts.InsertFilelist {
		opts.WriteMods = true
	}
	return opts
}

// pendingImage é o payload acumulado de um arquivo binário (o .imgb pode
// conter vários registros: um lote com duas imagens do MESMO container
// precisa aplicar as duas no mesmo buffer antes de gravar).
type pendingImage struct {
	rel      string
	data     []byte
	displays []string
}

// imageImportBatch acumula o que será entregue (mods e/ou filelist) no fim
// do lote.
type imageImportBatch struct {
	order []string
	byRel map[string]*pendingImage
}

func newImageImportBatch() *imageImportBatch {
	return &imageImportBatch{byRel: map[string]*pendingImage{}}
}

// addPatch aplica os mips de UM registro sobre o payload já acumulado do
// arquivo (ou sobre o pristine, na primeira vez).
func (b *imageImportBatch) addPatch(rel string, base []byte, mips []imgbcore.MipInfo, chunks [][]byte, display string) error {
	if rel == "" || strings.Contains(rel, "..") {
		return fmt.Errorf("caminho do payload inválido: %s", rel)
	}
	if prev, ok := b.byRel[rel]; ok {
		base = prev.data
	}
	patched, err := patchPayload(base, mips, chunks)
	if err != nil {
		return err
	}
	b.put(rel, patched, display)
	return nil
}

// addDirect registra um arquivo que já é o binário final (imagem direta).
func (b *imageImportBatch) addDirect(rel string, blob []byte, display string) error {
	if rel == "" || strings.Contains(rel, "..") {
		return fmt.Errorf("caminho da imagem inválido: %s", rel)
	}
	b.put(rel, blob, display)
	return nil
}

func (b *imageImportBatch) put(rel string, data []byte, display string) {
	p, ok := b.byRel[rel]
	if !ok {
		p = &pendingImage{rel: rel}
		b.order = append(b.order, rel)
		b.byRel[rel] = p
	}
	p.data = data
	if display != "" && !containsString(p.displays, display) {
		p.displays = append(p.displays, display)
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// deliver grava os payloads acumulados nos destinos escolhidos. Os dois
// destinos são independentes; um item que não existe no filelist vira aviso
// quando o mods também foi gravado (a importação não ficou sem destino) e
// erro quando "Inserir" era o único destino.
func (b *imageImportBatch) deliver(s *Service, part string, opts dto.ImageImportOptions) (updated, failed, warnings []string) {
	updated = []string{}
	failed = []string{}
	warnings = []string{}
	if len(b.order) == 0 {
		return updated, failed, warnings
	}

	okRels := map[string]bool{}
	wroteMods := map[string]bool{}
	var insertItems []InsertItem
	for _, rel := range b.order {
		p := b.byRel[rel]
		if opts.WriteMods {
			// Falha ao gravar em mods/ não impede o destino independente:
			// o "Inserir" ainda é tentado para este arquivo.
			if err := writeModsFile(part, rel, p.data); err != nil {
				failed = append(failed, rel+": "+err.Error())
			} else {
				okRels[rel] = true
				wroteMods[rel] = true
			}
		}
		if opts.InsertFilelist {
			insertItems = append(insertItems, InsertItem{RelPath: rel, Data: p.data})
		}
	}

	if opts.InsertFilelist && len(insertItems) > 0 {
		outcomes, err := s.InsertBlobs(part, insertItems)
		if err != nil {
			failed = append(failed, "filelist: "+err.Error())
		}
		for _, o := range outcomes {
			if o.Inserted {
				okRels[o.RelPath] = true
				continue
			}
			msg := o.RelPath + ": " + o.Reason
			if wroteMods[o.RelPath] {
				warnings = append(warnings, msg)
			} else {
				failed = append(failed, msg)
			}
		}
	}

	for _, rel := range b.order {
		if !okRels[rel] {
			continue
		}
		updated = append(updated, b.byRel[rel].displays...)
	}
	return updated, failed, warnings
}

// normalizeTranslatedRel normaliza o caminho relativo à raiz traduzida.
func normalizeTranslatedRel(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\\", "/")
	s = strings.TrimPrefix(s, "/")
	if s == "" || strings.Contains(s, "..") || filepath.IsAbs(filepath.FromSlash(s)) {
		return ""
	}
	return s
}

// walkTranslatedDDS devolve os .dds da raiz, relativos a ela (barras "/").
func walkTranslatedDDS(root string) ([]string, error) {
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
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".dds") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}

// buildImportPreviewEntry valida UM arquivo .dds da pasta traduzida.
func buildImportPreviewEntry(part, root, displayPath string) dto.ImageImportPreviewEntry {
	displayPath = normalizeTranslatedRel(displayPath)
	entry := dto.ImageImportPreviewEntry{ID: displayPath, DisplayPath: displayPath}
	if displayPath == "" {
		entry.ValidationError = "caminho inválido"
		return entry
	}
	dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(displayPath)))
	base := filepath.Base(displayPath)
	record := strings.TrimSuffix(base, filepath.Ext(base))
	target, err := resolveImageImportTarget(part, dir, record)
	if err != nil {
		entry.ValidationError = err.Error()
		return entry
	}
	dds, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(displayPath)))
	if err != nil {
		entry.ValidationError = fmt.Sprintf("erro de leitura (IO): %v", err)
		return entry
	}
	chunks, err := prepareImageMips(target, dds)
	if err != nil {
		entry.ValidationError = err.Error()
		return entry
	}
	entry.Differs = !payloadMatches(target, chunks)
	entry.Selected = entry.Differs
	return entry
}

// PreviewImportImages varre <mods>/translated/images/<part> e devolve o
// resumo do import: uma entrada por .dds, com erro de validação (container
// fora do filelist, registro inexistente, incompatibilidade) já embutido.
func (s *Service) PreviewImportImages(part string) (dto.ImageImportPreview, error) {
	part = config.NormalizePart(part)
	root := TranslatedImagesRoot(part)
	preview := dto.ImageImportPreview{
		Root:    root,
		Entries: []dto.ImageImportPreviewEntry{},
		Errors:  []string{},
	}
	files, err := walkTranslatedDDS(root)
	if err != nil {
		return preview, err
	}
	preview.Max = len(files)
	for _, rel := range files {
		entry := buildImportPreviewEntry(part, root, rel)
		if entry.ValidationError != "" {
			preview.Errors = append(preview.Errors,
				fmt.Sprintf("%s: %s", entry.DisplayPath, entry.ValidationError))
		} else if entry.Differs {
			preview.NewCount++
		} else {
			preview.IdenticalCount++
		}
		preview.Entries = append(preview.Entries, entry)
	}
	return preview, nil
}

// ImportImages aplica a seleção do modal: só as entradas marcadas são
// gravadas. Cada arquivo é revalidado na hora da gravação (o disco pode ter
// mudado desde o preview).
func (s *Service) ImportImages(part string, selection dto.ImageImportSelection) (dto.ImageImportResult, error) {
	part = config.NormalizePart(part)
	root := TranslatedImagesRoot(part)
	opts := normalizeImageImportOptions(dto.ImageImportOptions{
		WriteMods:      selection.WriteMods,
		InsertFilelist: selection.InsertFilelist,
	})
	res := dto.ImageImportResult{Updated: []string{}, Failed: []string{}, Warnings: []string{}}
	for _, e := range selection.Entries {
		if e.ID != "" && e.Image {
			res.Total++
		}
	}
	if res.Total == 0 {
		return res, nil
	}

	batch := s.progress.NewBatch("Importando imagens traduzidas…")
	step := batch.Step("Imagens")
	if step == nil {
		step = &progress.Step{}
	}
	step.Begin("Imagens", res.Total)

	acc := newImageImportBatch()
	for _, e := range selection.Entries {
		if e.ID == "" || !e.Image {
			continue
		}
		display := normalizeTranslatedRel(e.ID)
		fail := func(reason string) {
			res.Failed = append(res.Failed, display+": "+reason)
			step.Issue(display, reason)
		}
		if display == "" {
			fail("caminho inválido")
			continue
		}
		src := filepath.Join(root, filepath.FromSlash(display))
		if !pathInside(root, src) {
			fail("caminho fora da pasta traduzida")
			continue
		}
		dds, err := os.ReadFile(src)
		if err != nil {
			fail(fmt.Sprintf("erro de leitura (IO): %v", err))
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(display)))
		base := filepath.Base(display)
		record := strings.TrimSuffix(base, filepath.Ext(base))
		target, err := resolveImageImportTarget(part, dir, record)
		if err != nil {
			fail(err.Error())
			continue
		}
		chunks, err := prepareImageMips(target, dds)
		if err != nil {
			fail(err.Error())
			continue
		}
		if err := acc.addPatch(target.payloadRel, target.payload, target.mips, chunks, display); err != nil {
			fail(err.Error())
			continue
		}
		step.Step(display)
	}

	// A entrega (mods e/ou filelist) é feita uma vez por ARQUIVO binário:
	// várias imagens do mesmo .imgb não se sobrescrevem.
	updated, failed, warnings := acc.deliver(s, part, opts)
	res.Updated = updated
	res.Failed = append(res.Failed, failed...)
	res.Warnings = append(res.Warnings, warnings...)

	step.End()
	batch.SetSummary(len(res.Updated), root)
	batch.End()
	InvalidateImageFileIndex(part)
	InvalidateImageContainerIndex(part)
	invalidateCollection(part)
	return res, nil
}

// pathInside diz se path está dentro de root (defesa contra "../").
func pathInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ImportImage aplica UM .dds na imagem ABERTA. O nome/caminho do arquivo não
// é usado para localizar nada: o alvo é a própria ref.
func (s *Service) ImportImage(part, ref, ddsPath string, opts dto.ImageImportOptions) error {
	res, err := s.importImageRefs(part, []string{ref}, ddsPath, opts)
	if err != nil {
		return err
	}
	if len(res.Failed) > 0 {
		return fmt.Errorf("%s", res.Failed[0])
	}
	for _, w := range res.Warnings {
		loggingService.Warn("%s", w)
	}
	return nil
}

// ImportImageGroup aplica o mesmo .dds na imagem e nos alvos extras. No FF13
// ainda não há índice de duplicatas, então a UI não abre o diálogo de
// alcance; a função fica pronta para quando o índice existir.
func (s *Service) ImportImageGroup(part, ref, ddsPath string, targets []string, opts dto.ImageImportOptions) (dto.ImageImportResult, error) {
	all := make([]string, 0, len(targets)+1)
	if strings.TrimSpace(ref) != "" {
		all = append(all, ref)
	}
	all = append(all, targets...)
	return s.importImageRefs(part, all, ddsPath, opts)
}

// importImageRefs é o miolo do import avulso/grupo: resolve cada ref, separa
// os registros de container dos arquivos diretos, acumula os patches por
// ARQUIVO binário e entrega uma vez em cada destino escolhido.
func (s *Service) importImageRefs(part string, refs []string, ddsPath string, opts dto.ImageImportOptions) (dto.ImageImportResult, error) {
	part = config.NormalizePart(part)
	opts = normalizeImageImportOptions(opts)
	res := dto.ImageImportResult{Updated: []string{}, Failed: []string{}, Warnings: []string{}, Total: len(refs)}
	if len(refs) == 0 {
		return res, nil
	}
	dds, err := os.ReadFile(ddsPath)
	if err != nil {
		return res, fmt.Errorf("lendo %s: %w", filepath.Base(ddsPath), err)
	}

	acc := newImageImportBatch()
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		r, derr := DecodeImageRef(ref)
		if derr != nil {
			res.Failed = append(res.Failed, ref+": "+derr.Error())
			continue
		}
		if r.Entry == "" && !IsContainerExt(r.Path) {
			rel, blob, perr := prepareDirectImage(part, r, dds)
			if perr != nil {
				res.Failed = append(res.Failed, ref+": "+perr.Error())
				continue
			}
			if aerr := acc.addDirect(rel, blob, ref); aerr != nil {
				res.Failed = append(res.Failed, ref+": "+aerr.Error())
			}
			continue
		}
		if r.Entry == "" {
			res.Failed = append(res.Failed, ref+": selecione uma imagem dentro do container")
			continue
		}
		target, terr := imageImportTargetFromRef(part, r)
		if terr != nil {
			res.Failed = append(res.Failed, ref+": "+terr.Error())
			continue
		}
		chunks, cerr := prepareImageMips(target, dds)
		if cerr != nil {
			res.Failed = append(res.Failed, ref+": "+cerr.Error())
			continue
		}
		if aerr := acc.addPatch(target.payloadRel, target.payload, target.mips, chunks, ref); aerr != nil {
			res.Failed = append(res.Failed, ref+": "+aerr.Error())
		}
	}

	updated, failed, warnings := acc.deliver(s, part, opts)
	res.Updated = append(res.Updated, updated...)
	res.Failed = append(res.Failed, failed...)
	res.Warnings = append(res.Warnings, warnings...)
	InvalidateImageFileIndex(part)
	InvalidateImageContainerIndex(part)
	invalidateCollection(part)
	return res, nil
}

// prepareDirectImage cobre as imagens que não estão dentro de um container:
// .dds/.png diretos e as faces residentes (.imgb sem tabela). Devolve o
// caminho interno e o binário que será entregue em mods/ e/ou no filelist.
func prepareDirectImage(part string, r ImageRef, dds []byte) (string, []byte, error) {
	rel := canonicalImageRel(part, r)
	if rel == "" {
		return "", nil, fmt.Errorf("caminho da imagem vazio")
	}
	if IsResidentFaceImage(r.Path) {
		info, err := imagery.ParseDDSHeader(dds)
		if err != nil {
			return "", nil, err
		}
		var diffs []string
		if info.Width != ResidentFaceWidth || info.Height != ResidentFaceHeight {
			diffs = append(diffs, fmt.Sprintf("dimensões: container %dx%d, arquivo %dx%d",
				ResidentFaceWidth, ResidentFaceHeight, info.Width, info.Height))
		}
		if info.Format != imagery.DDSFormatDXT1 {
			diffs = append(diffs, fmt.Sprintf("formato: container DXT1, arquivo %s", info.Format))
		}
		if info.MipMapCount != 1 {
			diffs = append(diffs, fmt.Sprintf("mipmaps: container 1, arquivo %d", info.MipMapCount))
		}
		if len(diffs) > 0 {
			return "", nil, imageMismatchError(diffs)
		}
		if len(dds) != ddsHeaderSize+ResidentFaceRawSize {
			return "", nil, imageMismatchError([]string{fmt.Sprintf(
				"tamanho do payload: container %d bytes, arquivo %d bytes",
				ResidentFaceRawSize, len(dds)-ddsHeaderSize)})
		}
		return rel, dds[ddsHeaderSize:], nil
	}

	ext := strings.ToLower(filepath.Ext(r.Path))
	info, err := imagery.ParseDDSHeader(dds)
	if err != nil {
		return "", nil, err
	}
	switch ext {
	case ".dds":
		cur, rerr := (Service{}).readImageBytes(part, r)
		if rerr != nil {
			return "", nil, rerr
		}
		curInfo, cerr := imagery.ParseDDSHeader(cur)
		if cerr != nil {
			return "", nil, fmt.Errorf("imagem atual: %w", cerr)
		}
		var diffs []string
		if info.Width != curInfo.Width || info.Height != curInfo.Height {
			diffs = append(diffs, fmt.Sprintf("dimensões: container %dx%d, arquivo %dx%d",
				curInfo.Width, curInfo.Height, info.Width, info.Height))
		}
		if info.Format != curInfo.Format {
			diffs = append(diffs, fmt.Sprintf("formato: container %s, arquivo %s",
				curInfo.Format, info.Format))
		}
		if info.MipMapCount != curInfo.MipMapCount {
			diffs = append(diffs, fmt.Sprintf("mipmaps: container %d, arquivo %d",
				curInfo.MipMapCount, info.MipMapCount))
		}
		if len(diffs) > 0 {
			return "", nil, imageMismatchError(diffs)
		}
		return rel, dds, nil
	case ".png":
		cur, rerr := (Service{}).readImageBytes(part, r)
		if rerr != nil {
			return "", nil, rerr
		}
		cfg, _, derr := image.DecodeConfig(bytes.NewReader(cur))
		if derr != nil {
			return "", nil, fmt.Errorf("imagem atual: %w", derr)
		}
		if uint32(cfg.Width) != info.Width || uint32(cfg.Height) != info.Height {
			return "", nil, imageMismatchError([]string{fmt.Sprintf(
				"dimensões: container %dx%d, arquivo %dx%d",
				cfg.Width, cfg.Height, info.Width, info.Height)})
		}
		png, perr := imagery.DDSToPNG(dds)
		if perr != nil {
			return "", nil, perr
		}
		return rel, png, nil
	default:
		return "", nil, fmt.Errorf("import de imagem: extensão não suportada (%s)", ext)
	}
}

// readImageBytes lê a fonte efetiva da imagem (mods-first quando ligado).
func (Service) readImageBytes(part string, r ImageRef) ([]byte, error) {
	f, err := ResolveImageFile(part, r)
	if err != nil {
		return nil, err
	}
	return f.Read()
}

// writeModsFile grava bytes em mods/partN no caminho interno informado.
func writeModsFile(part, rel string, raw []byte) error {
	if rel == "" || strings.Contains(rel, "..") || filepath.IsAbs(filepath.FromSlash(rel)) {
		return fmt.Errorf("caminho interno inválido: %q", rel)
	}
	mods, err := config.PartRoot("mods", part)
	if err != nil {
		return err
	}
	dst := filepath.Join(mods, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}
