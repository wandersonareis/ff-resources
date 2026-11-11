package encoding

import (
	"fmt"

	xencoding "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/traditionalchinese"
)

// Charset converte a parte "texto comum" de uma linha (bytes que não são
// tags nem pares chara) entre a codepage física e UTF-8.
//
// Na implementação de referência isso é Encoding.Convert(CodepageToUse, UTF8, ...) no decode
// e Encoding.Convert(UTF8, CodepageToUse, ...) no encode. Aqui a dependência
// fica atrás de uma interface para permitir trocar a implementação (ex.:
// tabelas extraídas do .NET) sem tocar nas máquinas de estado.
type Charset interface {
	Decode(raw []byte) (string, error)
	Encode(s string) ([]byte, error)
}

// xTextCharset usa golang.org/x/text. As variantes seguem o WHATWG Encoding
// Standard: Shift_JIS == Windows-31J (cp932, incluindo as remapeações PUA
// que o .NET também faz), Big5 == cp950, EUC-KR == cp51949.
type xTextCharset struct {
	enc xencoding.Encoding
}

func (c xTextCharset) Decode(raw []byte) (string, error) {
	out, err := c.enc.NewDecoder().Bytes(raw)
	return string(out), err
}

func (c xTextCharset) Encode(s string) ([]byte, error) {
	return c.enc.NewEncoder().Bytes([]byte(s))
}

// CharsetFor devolve o charset físico da codepage.
func CharsetFor(cp Codepage) (Charset, error) {
	switch cp {
	case LatinJapanese:
		return xTextCharset{enc: japanese.ShiftJIS}, nil
	case Chinese:
		return xTextCharset{enc: traditionalchinese.Big5}, nil
	case Korean:
		return xTextCharset{enc: korean.EUCKR}, nil
	}
	return nil, fmt.Errorf("%w: %d", ErrCodepageNotFound, int(cp))
}
