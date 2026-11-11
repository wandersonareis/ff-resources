// Origem: ztrgo/backend/core/encoding/key_tables.go, reorganizado por grupo
// de tag. Os bytes são a identidade; nomes e swatches são apenas display.

package tags

// rawTags: pares crus conhecidos.
var rawTags = []Tag{
	// O bloco latino 0x85 0x80 e 0x85 0xA0..0xBC é GLIFO (acentuados),
	// não raw — ver glyph.go. Aqui fica só o 0x97, ainda sem mapeamento
	// confirmado nos binários de data/.
	{Class: ClassRaw, B1: 0x85, B2: 0x97},
	{Class: ClassRaw, B1: 0x81, B2: 0x40},
	{Class: ClassRaw, B1: 0xFA, B2: 0x20},
	{Class: ClassRaw, B1: 0xFF, B2: 0x82},
	{Class: ClassRaw, B1: 0xFF, B2: 0x83},
	{Class: ClassRaw, B1: 0xFF, B2: 0x86},
	{Class: ClassRaw, B1: 0xFF, B2: 0x8F},
	{Class: ClassRaw, B1: 0xFF, B2: 0x90},
	{Class: ClassRaw, B1: 0xFF, B2: 0x91},
	{Class: ClassRaw, B1: 0xFF, B2: 0x93},
	{Class: ClassRaw, B1: 0xFF, B2: 0x94},
	{Class: ClassRaw, B1: 0xFF, B2: 0x96},
	{Class: ClassRaw, B1: 0xFF, B2: 0x99},
	{Class: ClassRaw, B1: 0xFF, B2: 0x9A},
	{Class: ClassRaw, B1: 0xFF, B2: 0x9B},
	{Class: ClassRaw, B1: 0xFF, B2: 0x9D},
	{Class: ClassRaw, B1: 0xFF, B2: 0x9E},
	{Class: ClassRaw, B1: 0xFF, B2: 0xA9},
	{Class: ClassRaw, B1: 0xFF, B2: 0xB8},
	{Class: ClassRaw, B1: 0xFF, B2: 0xC9},
	{Class: ClassRaw, B1: 0xFF, B2: 0xCC},
	{Class: ClassRaw, B1: 0xFF, B2: 0xCE},
	{Class: ClassRaw, B1: 0xFF, B2: 0xD0},
	{Class: ClassRaw, B1: 0xFF, B2: 0xD3},
	{Class: ClassRaw, B1: 0xFF, B2: 0xDA},
	{Class: ClassRaw, B1: 0xFF, B2: 0xDD},
	{Class: ClassRaw, B1: 0xFF, B2: 0xE0},
	{Class: ClassRaw, B1: 0xFF, B2: 0xE3},
	{Class: ClassRaw, B1: 0xFF, B2: 0xE4},
	{Class: ClassRaw, B1: 0xFF, B2: 0xE6},
	{Class: ClassRaw, B1: 0xFF, B2: 0xF1},
}

// rawTagsExtra: pares crus do segundo conjunto (ex-Unk2). O chinês
// não consulta este conjunto — igual ao C#.
var rawTagsExtra = []Tag{
	{Class: ClassRaw, B1: 0xF1, B2: 0x78},
	{Class: ClassRaw, B1: 0xF4, B2: 0x44},
	{Class: ClassRaw, B1: 0xF4, B2: 0x45},
	{Class: ClassRaw, B1: 0xF4, B2: 0x46},
	{Class: ClassRaw, B1: 0xF4, B2: 0x47},
	{Class: ClassRaw, B1: 0xF4, B2: 0x48},
	{Class: ClassRaw, B1: 0xF4, B2: 0x49},
	{Class: ClassRaw, B1: 0xF4, B2: 0x60},
	{Class: ClassRaw, B1: 0xF5, B2: 0x40},
	{Class: ClassRaw, B1: 0xF6, B2: 0x60},
}
