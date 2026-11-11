package encoding

import "ffresources/internal/families/white/codecs/ztr/encoding/tags"

// Codepage identifica a variante de encoding dos .ztr (932/950/51949).
type Codepage = tags.Codepage

const (
	LatinJapanese = tags.LatinJapanese // -lj (Shift-JIS)
	Chinese       = tags.Chinese       // -ch (Big5)
	Korean        = tags.Korean        // -kr (EUC-KR)
)

// GameCode identifica o jogo dono do arquivo.
type GameCode = tags.GameCode

const (
	FF131 = tags.FF131
	FF132 = tags.FF132
	FF133 = tags.FF133
)
