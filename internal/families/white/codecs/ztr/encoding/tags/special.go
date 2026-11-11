// Origem: ztrgo/backend/core/encoding/key_tables.go, reorganizado por grupo
// de tag. Os bytes são a identidade; nomes e swatches
// são apenas display.

package tags

// specialTagsByCodepage: PAGE/NL/SPACE + VAR/CTR/KEY por codepage.
var specialTagsByCodepage = map[Codepage][]Tag{
	LatinJapanese: {
		{Class: ClassPage, B1: 0x40, B2: 0x70},
		{Class: ClassNL, B1: 0x40, B2: 0x72},
		{Class: ClassSpace, B1: 0x85, B2: 0x60},
		{Class: ClassVar, B1: 0xF4, B2: 0x40},
		{Class: ClassVar, B1: 0xF4, B2: 0x41},
		{Class: ClassVar, B1: 0xF4, B2: 0x42},
		{Class: ClassVar, B1: 0xF4, B2: 0x43},
		{Class: ClassKey, B1: 0xF6, B2: 0x40},
		{Class: ClassCtr, B1: 0xF7, B2: 0x40},
		{Class: ClassCtr, B1: 0xF7, B2: 0x41},
		{Class: ClassCtr, B1: 0xF7, B2: 0x42},
	},
	Chinese: {
		{Class: ClassPage, B1: 0x40, B2: 0x70},
		{Class: ClassNL, B1: 0x40, B2: 0x72},
		{Class: ClassSpace, B1: 0x85, B2: 0x60},
		{Class: ClassVar, B1: 0xFC, B2: 0x40},
		{Class: ClassVar, B1: 0xFC, B2: 0x41},
		{Class: ClassVar, B1: 0xFC, B2: 0x42},
		{Class: ClassVar, B1: 0xFC, B2: 0x43},
		{Class: ClassCtr, B1: 0xFD, B2: 0x40},
		{Class: ClassCtr, B1: 0xFD, B2: 0x41},
		{Class: ClassCtr, B1: 0xFD, B2: 0x42},
	},
	Korean: {
		{Class: ClassPage, B1: 0x40, B2: 0x70},
		{Class: ClassNL, B1: 0x40, B2: 0x72},
		{Class: ClassSpace, B1: 0x85, B2: 0x60},
		{Class: ClassVar, B1: 0xAA, B2: 0xA1},
		{Class: ClassVar, B1: 0xAA, B2: 0xA2},
		{Class: ClassVar, B1: 0xAA, B2: 0xA3},
		{Class: ClassVar, B1: 0xAA, B2: 0xA4},
		{Class: ClassCtr, B1: 0xAB, B2: 0xA1},
		{Class: ClassCtr, B1: 0xAB, B2: 0xA2},
		{Class: ClassCtr, B1: 0xAB, B2: 0xA3},
	},
}
