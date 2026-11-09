package encoding

// KeySet é a composição de dicionários usada por um par (game, codepage),
// equivalente aos switches dos helpers C# (KeysDecoder/KeysEncoder LJ/Ch/Kr).
//
// As ordens de consulta preservam a sequência de condições do C# — repare
// que decode e encode NÃO coincidem (exchara/special invertidos) e que o
// chara fica no meio da ordem de decode, antes de special.
type KeySet struct {
	Game     GameCode
	Codepage Codepage
	Charset  Charset // codepage física (texto comum)

	Single  []KVByte // SingleKeys (comum): 0x00..0x05
	Color   []KVPair // por jogo + codepage
	Icon    []KVPair
	Btn     []KVPair
	ExChara []KVPair // comum
	Special []KVPair // base comum + sufixo por codepage
	Unk     []KVPair // comum
	Unk2    []KVPair // comum; Ch não usa (nil), como no C#

	// chara: trails comuns + lead por codepage.
	Chara     []CharaEntry
	charaLead byte

	decodeOrder []string
	encodeOrder []string

	// Índices por categoria (montados por buildIndexes; imutáveis depois).
	singleFwd map[byte]string
	singleRev map[string]byte
	pairFwd   map[string]map[uint16]string // categoria -> par -> tag
	pairRev   map[string]map[string]uint16 // categoria -> tag -> par
	charaFwd  map[uint16]rune              // par -> runa
	charaRev  map[rune]uint16              // runa -> par (1ª ocorrência)
}

// Categorias de pares, na ordem em que o C# as consulta por codepage.
const (
	catColor   = "color"
	catIcon    = "icon"
	catBtn     = "btn"
	catChara   = "chara"
	catExChara = "exchara"
	catSpecial = "special"
	catUnk     = "unk"
	catUnk2    = "unk2"
)
