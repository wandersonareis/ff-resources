// Package ykd interpreta o formato .ykd da trilogia Final Fantasy XIII
// (definição de tela de UI): blocos hierárquicos com transform, animações
// nomeadas e uma tabela de sprites (recursos) que referenciam texturas do
// mesmo container WPD.
//
// O layout foi levantado lendo os arquivos do jogo. Tudo aqui é little
// endian — inclusive os 20 floats do transform (ao contrário do WPD, que é
// big endian).
package ykd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// Magic é o identificador "YKD_" (0x5F444B59 em little endian).
const Magic uint32 = 0x5F444B59

// Header são os 16 bytes iniciais.
type Header struct {
	Magic    uint32
	Unknown1 int32
	U2, U3   byte
	U4, U5   byte
	Dummy    int32
}

// Vec4 é um dos cinco vetores do transform do bloco.
type Vec4 [4]float32

// Block é um nó da tela.
type Block struct {
	Type            uint32
	Index           uint32
	AssociatedIndex uint32 // 0xFFFFFFFF = raiz
	Unknown         uint32

	Scale    Vec4 // multiplicador do tamanho do recurso
	Rotation Vec4 // radianos (X, Y, Z)
	Position Vec4 // pixels
	Pivot    Vec4 // ponto de rotação/escala dentro do elemento
	Extra    Vec4

	Entries []Entry
	Tail    Tail

	// Offset é o offset absoluto do bloco no segmento (para edição in-place).
	Offset int
	// End é o offset absoluto logo após o bloco (usado só pela leitura).
	End int
}

// ResourceRef é uma referência de sprite na cauda de um bloco.
type ResourceRef struct {
	ResourceIndex int32
	Variant       int32
	Flags         int32
	Extra         int32
}

// TextTail são os parâmetros de um nó de texto (types 5 e 6).
//
// A cauda tem 12 words: (0,0,0,0), (largura, altura, fonte, flags) e três
// cores + modo. A POSIÇÃO do texto é a `Position` do bloco — a cauda define a
// CAIXA (o "onde o texto cabe"), que é o que limita o número de letras.
type TextTail struct {
	Unknown    [4]int32
	Width      int32
	Height     int32
	FontSize   int32
	Flags      int32
	Color1     int32
	Color2     int32
	Color3     int32
	Mode       int32
	ByteOffset int // offset absoluto das 12 words (para edição in-place)
}

// Tail é a cauda opcional do bloco. O tipo do bloco decide o conteúdo.
type Tail struct {
	Kind         string // "none", "type0", "type4", "type5", "type6"
	ResourceRefs []ResourceRef
	Text         *TextTail
	Words        []int32
}

// Entry é uma animação nomeada ("in", "loop", "out").
type Entry struct {
	Name   string
	Groups []FrameGroup
	End    int
}

// FrameGroup é uma trilha de keyframes.
type FrameGroup struct {
	Unknown1 int32
	Unknown2 int32
	Unknown3 int32
	Frames   []Frame
	End      int
}

// Frame é um keyframe.
type Frame struct {
	Time       int32
	Value      float32
	Type       int32
	Flags      int32
	InTangent  float32
	OutTangent float32
	X          float32
	Y          float32
}

// Resource é um sprite (recorte de textura + tamanho de exibição + cores).
type Resource struct {
	Type    int32
	Index   int32
	Dummy2  int32
	Dummy3  int32
	Name    string
	Unknown [4]int32 // reservado (Fragment/Extra/Full variam)

	SourceX      int32
	SourceY      int32
	SourceWidth  int32
	SourceHeight int32
	ViewWidth    int32
	ViewHeight   int32
	Flags        int32
	Unknown5     int32
	Colors       [4]uint32

	// Full (type 3)
	UnknownFloats [4]float32
	Unknown1      int32
	Unknown2      int32

	// Extra (type 5)
	Tail [4]int32

	End int
}

// File é o conteúdo lógico de um .ykd.
type File struct {
	Header     Header
	Background Block
	Blocks     []Block
	Resources  []Resource
}

// Parse interpreta um segmento .ykd completo.
func Parse(data []byte) (*File, error) {
	p := &parser{data: data}
	if len(data) < 16 {
		return nil, fmt.Errorf("ykd: arquivo pequeno demais (%d bytes)", len(data))
	}

	magic, _ := p.u32(0)
	if magic != Magic {
		return nil, fmt.Errorf("ykd: magic %#x, esperado %#x", magic, Magic)
	}

	file := &File{}
	file.Header.Magic = magic
	file.Header.Unknown1, _ = p.i32(4)
	file.Header.U2 = data[8]
	file.Header.U3 = data[9]
	file.Header.U4 = data[10]
	file.Header.U5 = data[11]
	file.Header.Dummy, _ = p.i32(12)

	background, err := p.block(16, true)
	if err != nil {
		return nil, err
	}
	file.Background = *background

	blockTable, err := p.table(background.End)
	if err != nil {
		return nil, err
	}
	file.Blocks = make([]Block, 0, blockTable.Count)
	end := blockTable.End
	if background.End > end {
		end = background.End
	}
	for i := 0; i < blockTable.Count; i++ {
		block, berr := p.block(int(blockTable.Offsets[i]), false)
		if berr != nil {
			return nil, fmt.Errorf("ykd: bloco %d: %w", i, berr)
		}
		file.Blocks = append(file.Blocks, *block)
		if block.End > end {
			end = block.End
		}
	}

	resourceTable, err := p.table(end)
	if err != nil {
		return nil, err
	}
	file.Resources = make([]Resource, 0, resourceTable.Count)
	for i := 0; i < resourceTable.Count; i++ {
		resource, rerr := p.resource(int(resourceTable.Offsets[i]))
		if rerr != nil {
			return nil, fmt.Errorf("ykd: recurso %d: %w", i, rerr)
		}
		file.Resources = append(file.Resources, *resource)
	}

	return file, nil
}

type parser struct {
	data []byte
}

type offsetTable struct {
	Count   int
	Unknown [3]int32
	Offsets []int32
	End     int
}

func (p *parser) bytes(pos, size int) ([]byte, error) {
	if pos < 0 || size < 0 || pos+size > len(p.data) {
		return nil, fmt.Errorf("offset %d+%d fora do arquivo (%d bytes)", pos, size, len(p.data))
	}
	return p.data[pos : pos+size], nil
}

func (p *parser) u32(pos int) (uint32, error) {
	b, err := p.bytes(pos, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (p *parser) i32(pos int) (int32, error) {
	v, err := p.u32(pos)
	return int32(v), err
}

func (p *parser) f32(pos int) (float32, error) {
	v, err := p.u32(pos)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

func (p *parser) name(pos, size int) (string, error) {
	b, err := p.bytes(pos, size)
	if err != nil {
		return "", err
	}
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.ToValidUTF8(string(b), "?"), nil
}

func (p *parser) table(pos int) (*offsetTable, error) {
	count, err := p.i32(pos)
	if err != nil {
		return nil, err
	}
	if count < 0 || int(count) > (len(p.data)-pos-16)/4 {
		return nil, fmt.Errorf("tabela em 0x%X com contagem inválida (%d)", pos, count)
	}

	t := &offsetTable{Count: int(count)}
	for i := 0; i < 3; i++ {
		t.Unknown[i], _ = p.i32(pos + 4 + i*4)
	}

	t.Offsets = make([]int32, t.Count)
	for i := 0; i < t.Count; i++ {
		offset, oerr := p.i32(pos + 16 + i*4)
		if oerr != nil {
			return nil, oerr
		}
		if offset <= 0 || int(offset) >= len(p.data) {
			return nil, fmt.Errorf("offset %d da tabela em 0x%X fora do arquivo", offset, pos)
		}
		t.Offsets[i] = offset
	}

	pad := (4 - t.Count%4) % 4
	t.End = pos + 16 + (t.Count+pad)*4
	if t.End > len(p.data) {
		return nil, fmt.Errorf("tabela em 0x%X termina fora do arquivo", pos)
	}
	return t, nil
}

func (p *parser) block(pos int, first bool) (*Block, error) {
	if _, err := p.bytes(pos, 96); err != nil {
		return nil, err
	}

	block := &Block{Offset: pos}
	var err error
	block.Type, _ = p.u32(pos)
	block.Index, _ = p.u32(pos + 4)
	block.AssociatedIndex, _ = p.u32(pos + 8)
	block.Unknown, _ = p.u32(pos + 12)

	readVec := func(offset int) (Vec4, error) {
		var v Vec4
		for i := 0; i < 4; i++ {
			v[i], err = p.f32(offset + i*4)
			if err != nil {
				return v, err
			}
		}
		return v, nil
	}

	if block.Scale, err = readVec(pos + 16); err != nil {
		return nil, err
	}
	if block.Rotation, err = readVec(pos + 32); err != nil {
		return nil, err
	}
	if block.Position, err = readVec(pos + 48); err != nil {
		return nil, err
	}
	if block.Pivot, err = readVec(pos + 64); err != nil {
		return nil, err
	}
	if block.Extra, err = readVec(pos + 80); err != nil {
		return nil, err
	}

	entries, err := p.table(pos + 96)
	if err != nil {
		return nil, err
	}
	block.Entries = make([]Entry, 0, entries.Count)
	end := entries.End
	for i := 0; i < entries.Count; i++ {
		entry, eerr := p.entry(int(entries.Offsets[i]))
		if eerr != nil {
			return nil, eerr
		}
		block.Entries = append(block.Entries, *entry)
		if entry.End > end {
			end = entry.End
		}
	}

	block.Tail.Kind = "none"
	if !first {
		tail, tailEnd, terr := p.tail(end, block.Type)
		if terr != nil {
			return nil, terr
		}
		block.Tail = *tail
		if tailEnd > end {
			end = tailEnd
		}
		block.End = tailEnd
	} else {
		block.End = end
	}
	return block, nil
}

func (p *parser) entry(pos int) (*Entry, error) {
	name, err := p.name(pos, 16)
	if err != nil {
		return nil, err
	}
	table, err := p.table(pos + 16)
	if err != nil {
		return nil, err
	}

	entry := &Entry{Name: name, Groups: make([]FrameGroup, 0, table.Count)}
	end := table.End
	for i := 0; i < table.Count; i++ {
		group, gerr := p.frames(int(table.Offsets[i]))
		if gerr != nil {
			return nil, gerr
		}
		entry.Groups = append(entry.Groups, *group)
		if group.End > end {
			end = group.End
		}
	}
	entry.End = end
	return entry, nil
}

func (p *parser) frames(pos int) (*FrameGroup, error) {
	group := &FrameGroup{}
	var err error
	group.Unknown1, _ = p.i32(pos)
	group.Unknown2, _ = p.i32(pos + 4)
	count, _ := p.i32(pos + 8)
	group.Unknown3, _ = p.i32(pos + 12)

	if count < 0 || int(count) > (len(p.data)-pos-16)/4 {
		return nil, fmt.Errorf("trilha em 0x%X com contagem inválida (%d)", pos, count)
	}

	offsets := make([]int32, count)
	for i := int32(0); i < count; i++ {
		offsets[i], err = p.i32(pos + 16 + int(i)*4)
		if err != nil {
			return nil, err
		}
	}
	pad := (4 - int(count)%4) % 4
	tableEnd := pos + 16 + (int(count)+pad)*4
	if tableEnd > len(p.data) {
		return nil, fmt.Errorf("trilha em 0x%X termina fora do arquivo", pos)
	}

	group.Frames = make([]Frame, 0, count)
	end := tableEnd
	for i := 0; i < int(count); i++ {
		frame, ferr := p.frame(int(offsets[i]))
		if ferr != nil {
			return nil, ferr
		}
		group.Frames = append(group.Frames, *frame)
		if int(offsets[i])+32 > end {
			end = int(offsets[i]) + 32
		}
	}
	group.End = end
	return group, nil
}

func (p *parser) frame(pos int) (*Frame, error) {
	if _, err := p.bytes(pos, 32); err != nil {
		return nil, err
	}
	f := &Frame{}
	f.Time, _ = p.i32(pos)
	f.Value, _ = p.f32(pos + 4)
	f.Type, _ = p.i32(pos + 8)
	f.Flags, _ = p.i32(pos + 12)
	f.InTangent, _ = p.f32(pos + 16)
	f.OutTangent, _ = p.f32(pos + 20)
	f.X, _ = p.f32(pos + 24)
	f.Y, _ = p.f32(pos + 28)
	return f, nil
}

func (p *parser) tail(pos int, blockType uint32) (*Tail, int, error) {
	tail := &Tail{Kind: "none"}
	switch blockType {
	case 0:
		tail.Kind = "type0"
		ref, err := p.readRef(pos)
		if err != nil {
			return nil, 0, err
		}
		tail.ResourceRefs = []ResourceRef{*ref}
		return tail, pos + 16, nil
	case 4:
		tail.Kind = "type4"
		count, err := p.i32(pos)
		if err != nil {
			return nil, 0, err
		}
		if count < 0 || int(count) > (len(p.data)-pos-16)/16 {
			return nil, 0, fmt.Errorf("cauda tipo 4 em 0x%X com contagem inválida (%d)", pos, count)
		}
		tail.ResourceRefs = make([]ResourceRef, 0, count)
		for i := 0; i < int(count); i++ {
			ref, rerr := p.readRef(pos + 16 + i*16)
			if rerr != nil {
				return nil, 0, rerr
			}
			tail.ResourceRefs = append(tail.ResourceRefs, *ref)
		}
		return tail, pos + 16 + int(count)*16, nil
	case 5, 6:
		tail.Kind = fmt.Sprintf("type%d", blockType)
		tail.Text = &TextTail{ByteOffset: pos}
		if _, err := p.bytes(pos, 48); err != nil {
			return nil, 0, err
		}
		for i := 0; i < 4; i++ {
			tail.Text.Unknown[i], _ = p.i32(pos + i*4)
		}
		tail.Text.Width, _ = p.i32(pos + 16)
		tail.Text.Height, _ = p.i32(pos + 20)
		tail.Text.FontSize, _ = p.i32(pos + 24)
		tail.Text.Flags, _ = p.i32(pos + 28)
		tail.Text.Color1, _ = p.i32(pos + 32)
		tail.Text.Color2, _ = p.i32(pos + 36)
		tail.Text.Color3, _ = p.i32(pos + 40)
		tail.Text.Mode, _ = p.i32(pos + 44)
		return tail, pos + 48, nil
	default:
		return tail, pos, nil
	}
}

func (p *parser) readRef(pos int) (*ResourceRef, error) {
	if _, err := p.bytes(pos, 16); err != nil {
		return nil, err
	}
	ref := &ResourceRef{}
	ref.ResourceIndex, _ = p.i32(pos)
	ref.Variant, _ = p.i32(pos + 4)
	ref.Flags, _ = p.i32(pos + 8)
	ref.Extra, _ = p.i32(pos + 12)
	return ref, nil
}

func (p *parser) resource(pos int) (*Resource, error) {
	var err error
	resource := &Resource{}
	resource.Type, _ = p.i32(pos)
	resource.Index, _ = p.i32(pos + 4)
	resource.Dummy2, _ = p.i32(pos + 8)
	resource.Dummy3, _ = p.i32(pos + 12)
	resource.Name, err = p.name(pos+16, 16)
	if err != nil {
		return nil, err
	}

	end := pos + 32
	switch resource.Type {
	case 0: // Fragment
		resource.SourceX, _ = p.i32(pos + 32)
		resource.SourceY, _ = p.i32(pos + 36)
		resource.SourceWidth, _ = p.i32(pos + 40)
		resource.SourceHeight, _ = p.i32(pos + 44)
		resource.ViewWidth, _ = p.i32(pos + 48)
		resource.ViewHeight, _ = p.i32(pos + 52)
		resource.Flags, _ = p.i32(pos + 56)
		resource.Unknown5, _ = p.i32(pos + 60)
		for i := 0; i < 4; i++ {
			color, _ := p.u32(pos + 64 + i*4)
			resource.Colors[i] = color
		}
		end = pos + 80
	case 1: // Empty
		end = pos + 32
	case 3: // Full
		for i := 0; i < 4; i++ {
			resource.UnknownFloats[i], _ = p.f32(pos + 32 + i*4)
		}
		resource.ViewWidth, _ = p.i32(pos + 48)
		resource.ViewHeight, _ = p.i32(pos + 52)
		for i := 0; i < 4; i++ {
			color, _ := p.u32(pos + 56 + i*4)
			resource.Colors[i] = color
		}
		resource.Unknown1, _ = p.i32(pos + 72)
		resource.Unknown2, _ = p.i32(pos + 76)
		end = pos + 80
	case 5: // Extra
		resource.SourceX, _ = p.i32(pos + 32)
		resource.SourceY, _ = p.i32(pos + 36)
		resource.SourceWidth, _ = p.i32(pos + 40)
		resource.SourceHeight, _ = p.i32(pos + 44)
		resource.ViewWidth, _ = p.i32(pos + 48)
		resource.ViewHeight, _ = p.i32(pos + 52)
		resource.Flags, _ = p.i32(pos + 56)
		resource.Unknown5, _ = p.i32(pos + 60)
		for i := 0; i < 4; i++ {
			color, _ := p.u32(pos + 64 + i*4)
			resource.Colors[i] = color
		}
		for i := 0; i < 4; i++ {
			resource.Tail[i], _ = p.i32(pos + 80 + i*4)
		}
		end = pos + 96
	default:
		return nil, fmt.Errorf("recurso em 0x%X com tipo desconhecido (%d)", pos, resource.Type)
	}

	if end > len(p.data) {
		return nil, fmt.Errorf("recurso em 0x%X termina fora do arquivo", pos)
	}
	resource.End = end
	return resource, nil
}
