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

// writeImagePayload grava o payload com os mips novos em mods/partN. O
// container não é tocado: o header GTEX e a tabela de mips não mudam (mesmas
// dimensões, formato e mipmaps) e uma cópia do container pristine apagaria
// traduções de texto do mesmo WPD.
func writeImagePayload(part string, target *imageImportTarget, chunks [][]byte) error {
	if target.payloadRel == "" || strings.Contains(target.payloadRel, "..") {
		return fmt.Errorf("caminho do payload inválido: %s", target.payloadRel)
	}
	patched := make([]byte, len(target.payload))
	copy(patched, target.payload)
	for i, m := range target.mips {
		start := int(m.Offset)
		end := start + int(m.Size)
		if start < 0 || end > len(patched) {
			return fmt.Errorf("mip %d fora do payload (offset %d, tamanho %d, payload %d)",
				i, start, m.Size, len(patched))
		}
		copy(patched[start:end], chunks[i])
	}
	mods, err := config.PartRoot("mods", part)
	if err != nil {
		return err
	}
	dst := filepath.Join(mods, filepath.FromSlash(target.payloadRel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, patched, 0o644)
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
	res := dto.ImageImportResult{Updated: []string{}, Failed: []string{}}
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
		if err := writeImagePayload(part, target, chunks); err != nil {
			fail(err.Error())
			continue
		}
		res.Updated = append(res.Updated, display)
		step.Step(display)
	}

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
func (s *Service) ImportImage(part, ref, ddsPath string) error {
	part = config.NormalizePart(part)
	r, err := DecodeImageRef(ref)
	if err != nil {
		return err
	}
	dds, err := os.ReadFile(ddsPath)
	if err != nil {
		return fmt.Errorf("lendo %s: %w", filepath.Base(ddsPath), err)
	}
	if r.Entry == "" && !IsContainerExt(r.Path) {
		return importDirectImage(part, r, dds)
	}
	if r.Entry == "" {
		return fmt.Errorf("selecione uma imagem dentro do container")
	}
	target, err := imageImportTargetFromRef(part, r)
	if err != nil {
		return err
	}
	chunks, err := prepareImageMips(target, dds)
	if err != nil {
		return err
	}
	if err := writeImagePayload(part, target, chunks); err != nil {
		return err
	}
	InvalidateImageFileIndex(part)
	InvalidateImageContainerIndex(part)
	invalidateCollection(part)
	return nil
}

// ImportImageGroup aplica o mesmo .dds na imagem e nos alvos extras. No FF13
// ainda não há índice de duplicatas, então a UI não abre o diálogo de
// alcance; a função fica pronta para quando o índice existir.
func (s *Service) ImportImageGroup(part, ref, ddsPath string, targets []string) (dto.ImageImportResult, error) {
	part = config.NormalizePart(part)
	all := make([]string, 0, len(targets)+1)
	all = append(all, ref)
	all = append(all, targets...)
	res := dto.ImageImportResult{Updated: []string{}, Failed: []string{}, Total: len(all)}
	for _, target := range all {
		if strings.TrimSpace(target) == "" {
			continue
		}
		if err := s.ImportImage(part, target, ddsPath); err != nil {
			res.Failed = append(res.Failed, target+": "+err.Error())
			continue
		}
		res.Updated = append(res.Updated, target)
	}
	return res, nil
}

// importDirectImage cobre as imagens que não estão dentro de um container:
// .dds/.png diretos e as faces residentes (.imgb sem tabela).
func importDirectImage(part string, r ImageRef, dds []byte) error {
	rel := canonicalImageRel(part, r)
	if rel == "" {
		return fmt.Errorf("caminho da imagem vazio")
	}
	if IsResidentFaceImage(r.Path) {
		info, err := imagery.ParseDDSHeader(dds)
		if err != nil {
			return err
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
			return imageMismatchError(diffs)
		}
		if len(dds) != ddsHeaderSize+ResidentFaceRawSize {
			return imageMismatchError([]string{fmt.Sprintf(
				"tamanho do payload: container %d bytes, arquivo %d bytes",
				ResidentFaceRawSize, len(dds)-ddsHeaderSize)})
		}
		return writeModsFile(part, rel, dds[ddsHeaderSize:])
	}

	ext := strings.ToLower(filepath.Ext(r.Path))
	info, err := imagery.ParseDDSHeader(dds)
	if err != nil {
		return err
	}
	switch ext {
	case ".dds":
		cur, rerr := (Service{}).readImageBytes(part, r)
		if rerr != nil {
			return rerr
		}
		curInfo, cerr := imagery.ParseDDSHeader(cur)
		if cerr != nil {
			return fmt.Errorf("imagem atual: %w", cerr)
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
			return imageMismatchError(diffs)
		}
		return writeModsFile(part, rel, dds)
	case ".png":
		cur, rerr := (Service{}).readImageBytes(part, r)
		if rerr != nil {
			return rerr
		}
		cfg, _, derr := image.DecodeConfig(bytes.NewReader(cur))
		if derr != nil {
			return fmt.Errorf("imagem atual: %w", derr)
		}
		if uint32(cfg.Width) != info.Width || uint32(cfg.Height) != info.Height {
			return imageMismatchError([]string{fmt.Sprintf(
				"dimensões: container %dx%d, arquivo %dx%d",
				cfg.Width, cfg.Height, info.Width, info.Height)})
		}
		png, perr := imagery.DDSToPNG(dds)
		if perr != nil {
			return perr
		}
		return writeModsFile(part, rel, png)
	default:
		return fmt.Errorf("import de imagem: extensão não suportada (%s)", ext)
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
