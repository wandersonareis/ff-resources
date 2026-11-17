package service

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ykd"
	"ffresources/internal/families/white/imagery"
	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/dto"
)

// Preview de TELA de um arquivo .ykd: o grafo de blocos + sprites + texturas
// decodificadas. O ykd não tem animação de texto nem o conteúdo dos textos;
// o preview desenha o layout estático (transform base) e placeholders de
// texto.

const (
	ykdCanvasWidth  = 1280
	ykdCanvasHeight = 720
)

type ykdTextureLocation struct {
	Rel  string
	Name string
	Ext  string
}

var (
	ykdTextureIndexMu    sync.Mutex
	ykdTextureIndexCache sync.Map // part -> map[string]ykdTextureLocation
	ykdTextureCache      sync.Map // ref -> dto.YkdTexture
)

// GetYkdScreen lê o registro .ykd da ref e devolve o preview da tela.
func (Service) GetYkdScreen(part, ref string) (dto.YkdScreen, error) {
	part = config.NormalizePart(part)
	r, err := DecodeImageRef(ref)
	if err != nil {
		return dto.YkdScreen{}, err
	}
	if strings.TrimSpace(r.Entry) == "" {
		return dto.YkdScreen{}, fmt.Errorf("ykd: a referência não aponta para um registro")
	}

	data, table, err := ImageSection(part, r)
	if err != nil {
		return dto.YkdScreen{}, err
	}
	file, err := ykd.Parse(data)
	if err != nil {
		return dto.YkdScreen{}, fmt.Errorf("ykd: %w", err)
	}

	raw, err := table.Read()
	if err != nil {
		return dto.YkdScreen{}, err
	}
	container, err := wpd.Read(raw)
	if err != nil {
		return dto.YkdScreen{}, err
	}

	localTextures := make(map[string]wpd.Record)
	for _, record := range container.Header.Records {
		if IsImageRecordExt(record.Extension) {
			key := strings.ToLower(record.Name)
			if _, exists := localTextures[key]; !exists {
				localTextures[key] = record
			}
		}
	}

	recordName, recordExt := SplitImageRecordKey(r.Entry)
	pristineWidths := ykdPristineTextWidths(part, table.Rel, recordName, recordExt)

	screen := dto.YkdScreen{
		CanvasWidth:  ykdCanvasWidth,
		CanvasHeight: ykdCanvasHeight,
		Textures:     make(map[string]dto.YkdTexture),
		Header: dto.YkdHeader{
			Unknown1: file.Header.Unknown1,
			U2:       int(file.Header.U2),
			U3:       int(file.Header.U3),
			U4:       int(file.Header.U4),
			U5:       int(file.Header.U5),
		},
	}

	// Seletor rápido: todas as telas do mesmo container WPD.
	for _, record := range container.Header.Records {
		if !strings.EqualFold(record.Extension, "ykd") {
			continue
		}
		siblingRef := ImageRef{
			Source:   r.Source,
			Root:     r.Root,
			Path:     r.Path,
			Entry:    fmt.Sprintf("rec:%s.%s", record.Name, record.Extension),
			Payload:  r.Payload,
			Platform: r.Platform,
		}
		screen.Siblings = append(screen.Siblings, dto.YkdScreenRef{
			Name: record.Name + "." + record.Extension,
			Ref:  EncodeImageRef(siblingRef),
		})
	}
	sort.Slice(screen.Siblings, func(i, j int) bool { return screen.Siblings[i].Name < screen.Siblings[j].Name })

	resources := make(map[int32]*dto.YkdResource, len(file.Resources))
	resourcePositions := make(map[int32]int, len(file.Resources))
	globalTextures := textureIndexFor(part)
	missing := make(map[string]bool)

	for i := range file.Resources {
		source := &file.Resources[i]
		resource := dto.YkdResource{
			Index:        source.Index,
			Type:         source.Type,
			TypeName:     ykdResourceTypeName(source.Type),
			Name:         source.Name,
			SourceX:      source.SourceX,
			SourceY:      source.SourceY,
			SourceWidth:  source.SourceWidth,
			SourceHeight: source.SourceHeight,
			ViewWidth:    source.ViewWidth,
			ViewHeight:   source.ViewHeight,
			Flags:        source.Flags,
			Colors:       source.Colors,
		}

		key := strings.TrimSpace(source.Name)
		if source.Type == 1 || key == "" {
			resource.Missing = source.Type != 1
		} else if texture, ok := resolveYkdTexture(part, r, key, localTextures, globalTextures); ok {
			screen.Textures[key] = texture
			resource.Texture = key
		} else {
			resource.Missing = true
			missing[key] = true
		}

		resourcePositions[source.Index] = len(screen.Resources)
		screen.Resources = append(screen.Resources, resource)
	}

	for index, position := range resourcePositions {
		resources[index] = &screen.Resources[position]
	}

	// Recursos Full/Empty não trazem recorte: completam com o tamanho real da
	// textura decodificada.
	for i := range screen.Resources {
		resource := &screen.Resources[i]
		if resource.Texture == "" {
			continue
		}
		texture := screen.Textures[resource.Texture]
		if resource.SourceWidth == 0 {
			resource.SourceWidth = int32(texture.Width)
		}
		if resource.SourceHeight == 0 {
			resource.SourceHeight = int32(texture.Height)
		}
		if resource.ViewWidth == 0 {
			resource.ViewWidth = int32(texture.Width)
		}
		if resource.ViewHeight == 0 {
			resource.ViewHeight = int32(texture.Height)
		}
	}

	byIndex := make(map[uint32]*ykd.Block, len(file.Blocks))
	for i := range file.Blocks {
		byIndex[file.Blocks[i].Index] = &file.Blocks[i]
	}
	if _, exists := byIndex[file.Background.Index]; !exists {
		byIndex[file.Background.Index] = &file.Background
	}

	for i := range file.Blocks {
		screen.Blocks = append(screen.Blocks, ykdBlockDTO(&file.Blocks[i]))
	}

	for i := range file.Blocks {
		block := &file.Blocks[i]
		world := ykdWorldOf(block, byIndex)

		switch block.Type {
		case 0, 4:
			for _, ref := range block.Tail.ResourceRefs {
				resource := resources[ref.ResourceIndex]
				if resource == nil || resource.Type == 1 {
					continue
				}
				item := dto.YkdItem{
					Kind:          "sprite",
					BlockIndex:    i,
					BlockType:     block.Type,
					NodeIndex:     block.Index,
					X:             world.X,
					Y:             world.Y,
					ScaleX:        world.ScaleX,
					ScaleY:        world.ScaleY,
					Rotation:      world.Rotation,
					PivotX:        block.Pivot[0],
					PivotY:        block.Pivot[1],
					FlipX:         world.FlipX,
					FlipY:         world.FlipY,
					ResourceIndex: ref.ResourceIndex,
					Texture:       resource.Texture,
					Missing:       resource.Missing,
					SourceX:       resource.SourceX,
					SourceY:       resource.SourceY,
					SourceWidth:   resource.SourceWidth,
					SourceHeight:  resource.SourceHeight,
					ViewWidth:     resource.ViewWidth,
					ViewHeight:    resource.ViewHeight,
					Flags:         resource.Flags,
				}
				screen.Items = append(screen.Items, item)
			}

		case 5, 6:
			if block.Tail.Text == nil {
				continue
			}
			text := block.Tail.Text
			originalWidth := text.Width
			if width, ok := pristineWidths[i]; ok {
				originalWidth = width
			}
			maxWidth := ykdMaxTextWidth(originalWidth, text.Width, world.X-block.Pivot[0])
			screen.Items = append(screen.Items, dto.YkdItem{
				Kind:       "text",
				BlockIndex: i,
				BlockType:  block.Type,
				NodeIndex:  block.Index,
				X:          world.X,
				Y:          world.Y,
				ScaleX:     world.ScaleX,
				ScaleY:     world.ScaleY,
				Rotation:   world.Rotation,
				PivotX:     block.Pivot[0],
				PivotY:     block.Pivot[1],
				Text: &dto.YkdText{
					Width:         text.Width,
					Height:        text.Height,
					FontSize:      text.FontSize,
					Flags:         text.Flags,
					Color1:        text.Color1,
					Color2:        text.Color2,
					Color3:        text.Color3,
					Mode:          text.Mode,
					OriginalWidth: originalWidth,
					MaxWidth:      maxWidth,
				},
			})
			screen.Stats.TextNodes++
		}
	}

	for name := range missing {
		screen.MissingTextures = appendMissing(screen.MissingTextures, name)
	}
	sort.Strings(screen.MissingTextures)

	screen.Stats.Blocks = len(screen.Blocks)
	screen.Stats.Resources = len(screen.Resources)
	screen.Stats.Items = len(screen.Items)
	screen.Stats.Textures = len(screen.Textures)
	screen.Stats.MissingTextures = len(screen.MissingTextures)
	for _, item := range screen.Items {
		if item.Kind == "sprite" {
			if item.Texture != "" {
				screen.Stats.TexturedItems++
			}
		}
	}

	return screen, nil
}

// ApplyYkdText edita uma caixa de texto (largura, altura, fonte, flags,
// posição e pivô) e grava o container em mods/partN. A edição é in-place:
// os campos têm tamanho fixo, então offsets e tamanhos do WPD não mudam.
//
// Este é o ajuste "de tamanho onde o texto vai": aumentar Width permite mais
// caracteres; Height/FontSize/Flags controlam a caixa e o modo de exibição.
func (Service) ApplyYkdText(part, ref string, patch dto.YkdTextPatch) (dto.YkdScreen, error) {
	part = config.NormalizePart(part)
	r, err := DecodeImageRef(ref)
	if err != nil {
		return dto.YkdScreen{}, err
	}
	if strings.TrimSpace(r.Entry) == "" {
		return dto.YkdScreen{}, fmt.Errorf("ykd: a referência não aponta para um registro")
	}

	_, table, err := ImageSection(part, r)
	if err != nil {
		return dto.YkdScreen{}, err
	}
	raw, err := table.Read()
	if err != nil {
		return dto.YkdScreen{}, err
	}
	container, err := wpd.Read(raw)
	if err != nil {
		return dto.YkdScreen{}, err
	}

	name, ext := SplitImageRecordKey(r.Entry)
	recordIndex := -1
	for i := range container.Header.Records {
		record := container.Header.Records[i]
		if strings.EqualFold(record.Name, name) && strings.EqualFold(record.Extension, ext) {
			recordIndex = i
			break
		}
	}
	if recordIndex < 0 {
		return dto.YkdScreen{}, fmt.Errorf("ykd: registro %s não encontrado", r.Entry)
	}

	record := container.Header.Records[recordIndex]
	start := int(record.Offset)
	end := start + int(record.Size)
	if start < 0 || end > len(raw) {
		return dto.YkdScreen{}, fmt.Errorf("ykd: registro fora do container")
	}
	data := raw[start:end]

	file, err := ykd.Parse(data)
	if err != nil {
		return dto.YkdScreen{}, fmt.Errorf("ykd: %w", err)
	}
	if patch.BlockIndex < 0 || patch.BlockIndex >= len(file.Blocks) {
		return dto.YkdScreen{}, fmt.Errorf("ykd: bloco %d inválido", patch.BlockIndex)
	}
	block := &file.Blocks[patch.BlockIndex]
	if block.Tail.Text == nil {
		return dto.YkdScreen{}, fmt.Errorf("ykd: o bloco %d não é um nó de texto", patch.BlockIndex)
	}
	text := block.Tail.Text

	if text.ByteOffset+16+4 > len(data) {
		return dto.YkdScreen{}, fmt.Errorf("ykd: campos fora do registro")
	}

	byIndex := make(map[uint32]*ykd.Block, len(file.Blocks))
	for i := range file.Blocks {
		byIndex[file.Blocks[i].Index] = &file.Blocks[i]
	}
	world := ykdWorldOf(block, byIndex)
	recordName, recordExt := SplitImageRecordKey(r.Entry)
	pristineWidths := ykdPristineTextWidths(part, table.Rel, recordName, recordExt)
	originalWidth := text.Width
	if width, ok := pristineWidths[patch.BlockIndex]; ok {
		originalWidth = width
	}
	maxWidth := ykdMaxTextWidth(originalWidth, text.Width, world.X-block.Pivot[0])
	if patch.Width < 1 || patch.Width > maxWidth {
		return dto.YkdScreen{}, fmt.Errorf("ykd: largura %d fora do limite seguro (1..%d)", patch.Width, maxWidth)
	}

	putI32(data, text.ByteOffset+16, patch.Width)

	// Revalida o registro antes de gravar.
	if _, err := ykd.Parse(data); err != nil {
		return dto.YkdScreen{}, fmt.Errorf("ykd: edição inválida: %w", err)
	}

	modsRoot, err := config.PartRoot("mods", part)
	if err != nil {
		return dto.YkdScreen{}, err
	}
	target := filepath.Join(modsRoot, filepath.FromSlash(table.Rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return dto.YkdScreen{}, err
	}
	if err := os.WriteFile(target, raw, 0o644); err != nil {
		return dto.YkdScreen{}, err
	}

	// A prévia recarrega do container recém-gravado (mods-first).
	return (Service{}).GetYkdScreen(part, ref)
}

func putI32(data []byte, offset int, value int32) {
	binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(value))
}

// ykdPristineTextWidths lê as larguras ORIGINAIS (data/, sem mods) das caixas
// de texto do mesmo registro. É a base estável do limite de crescimento: um
// mod salvo não aumenta o teto, então não dá para crescer indefinidamente.
func ykdPristineTextWidths(part, tableRel, recordName, recordExt string) map[int]int32 {
	if tableRel == "" || recordName == "" {
		return nil
	}
	pristine, ok := resolveRelFilePristine(part, tableRel)
	if !ok {
		return nil
	}
	raw, err := pristine.Read()
	if err != nil {
		return nil
	}
	container, err := wpd.Read(raw)
	if err != nil {
		return nil
	}
	for _, record := range container.Header.Records {
		if !strings.EqualFold(record.Name, recordName) || !strings.EqualFold(record.Extension, recordExt) {
			continue
		}
		start := int(record.Offset)
		end := start + int(record.Size)
		if start < 0 || end > len(raw) {
			return nil
		}
		file, perr := ykd.Parse(raw[start:end])
		if perr != nil {
			return nil
		}
		widths := make(map[int]int32)
		for i := range file.Blocks {
			if text := file.Blocks[i].Tail.Text; text != nil {
				widths[i] = text.Width
			}
		}
		return widths
	}
	return nil
}

// ykdMaxTextWidth é o teto de largura de uma caixa de texto:
//   - no máximo 2× a largura ORIGINAL (ou original+64 em caixas pequenas);
//   - nunca além da borda direita da tela (left + width <= 1280);
//   - nunca abaixo da largura já salva (não força “encolher” um mod).
func ykdMaxTextWidth(original, current int32, left float32) int32 {
	if original <= 0 {
		original = current
	}
	if original <= 0 {
		return 1
	}
	limit := original * 2
	if limit < original+64 {
		limit = original + 64
	}
	screen := int32(ykdCanvasWidth)
	if left > 0 {
		screen -= int32(left)
	}
	if screen < 1 {
		screen = 1
	}
	if limit > screen {
		limit = screen
	}
	if limit < current {
		limit = current
	}
	return limit
}

// resolveYkdTexture procura a textura no MESMO container e, se não achar, no
// índice global de containers de GUI da part. Devolve o PNG já decodificado.
func resolveYkdTexture(part string, base ImageRef, name string, local map[string]wpd.Record,
	global map[string]ykdTextureLocation) (dto.YkdTexture, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))

	if record, ok := local[lower]; ok {
		ref := ImageRef{
			Source:   base.Source,
			Root:     base.Root,
			Path:     base.Path,
			Entry:    fmt.Sprintf("rec:%s.%s", record.Name, record.Extension),
			Payload:  base.Payload,
			Platform: base.Platform,
		}
		if texture, err := decodeYkdTexture(part, ref); err == nil {
			return texture, true
		}
	}

	if location, ok := global[lower]; ok {
		ref := ImageRef{
			Source:   "data",
			Path:     location.Rel,
			Entry:    fmt.Sprintf("rec:%s.%s", location.Name, location.Ext),
			Payload:  ImagePairPath(location.Rel),
			Platform: ImagePlatformOf(location.Rel),
		}
		if texture, err := decodeYkdTexture(part, ref); err == nil {
			return texture, true
		}
	}

	return dto.YkdTexture{}, false
}

// decodeYkdTexture decodifica um registro WPD de textura para PNG (data URL),
// com cache por ref. Diferente de GetImageEntry, não calcula duplicatas nem
// o estado de mods: é só preview.
func decodeYkdTexture(part string, ref ImageRef) (dto.YkdTexture, error) {
	key := EncodeImageRef(ref)
	if cached, ok := ykdTextureCache.Load(key); ok {
		return cached.(dto.YkdTexture), nil
	}

	dds, info, err := DecodeImageDDS(part, ref, false)
	if err != nil {
		return dto.YkdTexture{}, err
	}
	png, err := imagery.DDSToPNG(dds)
	if err != nil {
		return dto.YkdTexture{}, err
	}

	texture := dto.YkdTexture{
		PNGData: ImageDataURL("image/png", png),
		Width:   int(info.Width),
		Height:  int(info.Height),
		Source:  ref.Entry,
	}
	ykdTextureCache.Store(key, texture)
	return texture, nil
}

// textureIndexFor monta (uma vez por part) um índice nome -> container de
// textura para os .xgr de GUI. É o fallback para recursos que referenciam
// texturas de outro pacote (ex.: review usa btl_number01 de system).
func textureIndexFor(part string) map[string]ykdTextureLocation {
	part = config.NormalizePart(part)
	if cached, ok := ykdTextureIndexCache.Load(part); ok {
		return cached.(map[string]ykdTextureLocation)
	}

	ykdTextureIndexMu.Lock()
	defer ykdTextureIndexMu.Unlock()
	if cached, ok := ykdTextureIndexCache.Load(part); ok {
		return cached.(map[string]ykdTextureLocation)
	}

	index := make(map[string]ykdTextureLocation)
	for rel, file := range ImageFileIndexFor(parts.Get(part)) {
		lower := strings.ToLower(rel)
		if !(strings.HasPrefix(lower, "gui/") || strings.Contains(lower, "/gui/")) ||
			!strings.HasSuffix(lower, ".win32.xgr") {
			continue
		}
		var raw []byte
		var err error
		if resolved, ok := ResolveRelFile(part, rel); ok {
			raw, err = resolved.Read()
		} else {
			raw, err = file.Read()
		}
		if err != nil {
			continue
		}
		header, err := wpd.ReadTable(raw)
		if err != nil {
			continue
		}
		for _, record := range header.Records {
			if !IsImageRecordExt(record.Extension) {
				continue
			}
			key := strings.ToLower(record.Name)
			if _, exists := index[key]; exists {
				continue
			}
			index[key] = ykdTextureLocation{Rel: file.Rel, Name: record.Name, Ext: record.Extension}
		}
	}

	ykdTextureIndexCache.Store(part, index)
	return index
}

// InvalidateYkdCaches descarta o índice de texturas e as texturas decodificadas
// da part (chamado quando data/mods mudam).
func InvalidateYkdCaches(part string) {
	part = config.NormalizePart(part)
	ykdTextureIndexCache.Delete(part)
	ykdTextureCache.Range(func(key, _ any) bool {
		ykdTextureCache.Delete(key)
		return true
	})
}

func ykdBlockDTO(block *ykd.Block) dto.YkdBlock {
	result := dto.YkdBlock{
		Type:        block.Type,
		Index:       block.Index,
		ParentIndex: block.AssociatedIndex,
		Unknown:     block.Unknown,
		Scale:       block.Scale,
		Rotation:    block.Rotation,
		Position:    block.Position,
		Pivot:       block.Pivot,
		Extra:       block.Extra,
	}
	for _, ref := range block.Tail.ResourceRefs {
		result.ResourceIndexes = append(result.ResourceIndexes, ref.ResourceIndex)
	}
	if block.Tail.Text != nil {
		text := block.Tail.Text
		result.Text = &dto.YkdText{
			Width:    text.Width,
			Height:   text.Height,
			FontSize: text.FontSize,
			Flags:    text.Flags,
			Color1:   text.Color1,
			Color2:   text.Color2,
			Color3:   text.Color3,
			Mode:     text.Mode,
		}
	}
	for _, entry := range block.Entries {
		result.Entries = append(result.Entries, entry.Name)
	}
	return result
}

type ykdWorld struct {
	X, Y     float32
	ScaleX   float32
	ScaleY   float32
	Rotation float32
	FlipX    bool
	FlipY    bool
}

// ykdWorldOf acumula a posição/escala/rotação dos pais. É uma aproximação:
// posições de filhos são somadas no espaço local (a maioria dos nós usa
// escala 1) e a rotação 2D usa o eixo Z. Os eixos X/Y em π viram flip.
func ykdWorldOf(block *ykd.Block, byIndex map[uint32]*ykd.Block) ykdWorld {
	world := ykdWorld{
		X:        block.Position[0],
		Y:        block.Position[1],
		ScaleX:   block.Scale[0],
		ScaleY:   block.Scale[1],
		Rotation: block.Rotation[2],
	}
	if isHalfTurn(block.Rotation[0]) {
		world.FlipX = !world.FlipX
	}
	if isHalfTurn(block.Rotation[1]) {
		world.FlipY = !world.FlipY
	}

	seen := map[uint32]bool{block.Index: true}
	parent := block.AssociatedIndex
	for depth := 0; depth < 32 && parent != 0xFFFFFFFF; depth++ {
		node, ok := byIndex[parent]
		if !ok || seen[parent] {
			break
		}
		seen[parent] = true
		world.X += node.Position[0]
		world.Y += node.Position[1]
		world.ScaleX *= node.Scale[0]
		world.ScaleY *= node.Scale[1]
		world.Rotation += node.Rotation[2]
		if isHalfTurn(node.Rotation[0]) {
			world.FlipX = !world.FlipX
		}
		if isHalfTurn(node.Rotation[1]) {
			world.FlipY = !world.FlipY
		}
		parent = node.AssociatedIndex
	}
	return world
}

func isHalfTurn(value float32) bool {
	return math.Abs(float64(value)-math.Pi) < 0.01
}

func ykdResourceTypeName(value int32) string {
	switch value {
	case 0:
		return "Fragment"
	case 1:
		return "Empty"
	case 3:
		return "Full"
	case 5:
		return "Extra"
	default:
		return "Unknown"
	}
}

func appendMissing(list []string, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return list
	}
	for _, existing := range list {
		if existing == name {
			return list
		}
	}
	return append(list, name)
}
