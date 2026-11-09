package encoding

// Codepage identifica a variante de encoding dos .ztr (equivalente aos
// EncodingSwitches do ZTRtool: -lj/-ch/-kr → 932/950/51949).
type Codepage int

const (
	LatinJapanese Codepage = 932   // -lj (Shift-JIS)
	Chinese       Codepage = 950   // -ch (Big5)
	Korean        Codepage = 51949 // -kr (EUC-KR)
)

// GameCode identifica o jogo dono do arquivo (-ff131/-ff132/-ff133).
type GameCode string

const (
	FF131 GameCode = "ff131"
	FF132 GameCode = "ff132"
	FF133 GameCode = "ff133"
)
