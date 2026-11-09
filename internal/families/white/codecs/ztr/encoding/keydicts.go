// Package encoding reproduz a camada de texto do ZTRtool C# (trilogia FFXIII):
// codepage física, "chara keys" (pares de 2 bytes que o jogo renderiza como
// caracteres Latin-1/acentuados) e tags estruturais (color, icon, btn, ...).
//
// Diferenças deliberadas em relação ao C# (design Pulse/Core.Encoding):
//   - chara keys são CARACTERES da codepage, não tags: no txt saem sem
//     chaves (0x85 0x57 -> "—"); chaves apenas nas tags estruturais;
//   - o encoder trabalha sobre runas, não sobre bytes da codepage, o que
//     elimina a colisão do C# entre o byte 0x7B ('{') e trails DBCS.
//
// As tabelas de dicionários são geradas por scripts/gen_key_tables.go no
// formato comum + variantes: o que é igual entre jogos/idiomas vive uma
// única vez (single, chara, exchara, unk, unk2, special-base), e só as
// partes exclusivas (color/icon/btn por jogo e codepage) ficam separadas.
package encoding

import "errors"

// Sentinelas distinguindo configuração ausente (aborta) de chave ausente
// no dado (o chamador decide: tag literal ou placeholder).
var (
	ErrGameCodeNotFound  = errors.New("game code não encontrado nas tabelas de keys")
	ErrCodepageNotFound  = errors.New("codepage não encontrada para esse game code")
	ErrKeyNotFound       = errors.New("chave não encontrada no dicionário")
	ErrDuplicateKeyValue = errors.New("valor duplicado no dicionário (encode ambíguo)")
	ErrInvalidInput      = errors.New("linha inválida")
)

// Tipos de par chave->valor usados pelas tabelas geradas.
type (
	// KVByte é um mapa byte único -> tag, ex.: 0x00 -> "{End}" (SingleKeys).
	KVByte struct {
		Key   byte
		Value string
	}

	// KVPair é um mapa byte,byte -> tag, ex.: (0xF9,0x32) -> "{Color Ex00}".
	KVPair struct {
		B1    byte
		B2    byte
		Value string
	}

	// CharaEntry é a "codepage estendida" do jogo: trail do par chara ->
	// runa exibida. O lead (0x85 no Shift-JIS, 0x80 no Big5/EUC-KR) varia
	// por codepage e é aplicado na composição (charaLeadByCP).
	CharaEntry struct {
		Trail byte
		Rune  rune
	}
)
