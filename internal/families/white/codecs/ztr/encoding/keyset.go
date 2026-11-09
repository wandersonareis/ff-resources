package encoding

// KeySet é a composição de dicionários usada por um par (game, codepage),
// equivalente aos switches dos helpers C# (KeysDecoder/KeysEncoder LJ/Ch/Kr).
//
// Os pares agora são Tag (classe + bytes + nome de display): a identidade é
// o PAR, e a consulta é por CLASSE numa ordem única (ver indexes.go). O
// chara fica no meio da ordem, como no C#.
type KeySet struct {
	Game     GameCode
	Codepage Codepage
	Charset  Charset // codepage física (texto comum)

	Single  []KVByte // 0x00..0x05
	Color   []Tag
	Icon    []Tag
	Btn     []Tag
	Special []Tag // PAGE/NL/TAB + VAR/CTR/KEY
	Raw     []Tag // exchara + unk
	Raw2    []Tag // unk2; Ch não usa (nil), como no C#
	Chara   []CharaEntry

	charaLead byte

	// Índices por categoria (montados por buildIndexes; imutáveis depois).
	singleFwd map[byte]string             // byte -> tag de byte único
	singleRev map[string]byte             // tag de byte único -> byte
	pairFwd   map[TagClass]map[uint16]Tag // classe -> par -> tag
	nameFwd   map[TagClass]map[string]Tag // classe -> nome -> tag (COLOR)
	charaFwd  map[uint16]rune             // par -> runa
	charaRev  map[rune]uint16             // runa -> par (1ª ocorrência)
}

// Ordem de consulta dos pares. O C# intercalava o chara entre Btn e Special;
// a ordem do encoder era exchara antes de special. Como NENHUM par aparece
// em duas categorias (validado pelo gerador), uma ordem única serve para os
// dois sentidos.
var (
	pairPreChara  = []TagClass{ClassColor, ClassIcon, ClassBtn}
	pairPostChara = []TagClass{
		ClassPage, ClassNL, ClassTab,
		ClassVar, ClassCtr, ClassKey,
		ClassRaw,
	}
)
