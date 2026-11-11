package encoding

import (
	"fmt"
	"unicode/utf8"
)

// DECODE: bytes da linha -> texto UTF-8 com as tags.
//
// Máquina de estados, um byte por passo:
//
//  1. trail de multibyte do charset (isTrail)      -> acumula no segmento
//  2. controle único 0x00..0x05 (controlFwd)       -> tag de byte único
//     (00 00 = terminador da linha)
//  3. par por categoria (resolveDecodePair, com os
//     glifos no meio) ou PUA do Shift-JIS           -> tag / rune
//  4. 0xFF isolado                                  -> {B:FF}
//  5. demais bytes                                  -> acumula p/ charset
//
// O texto puro é decodificado em SEGMENTOS (flush) para que as tags nunca
// atravessem o charset.

// DecodeLine converte uma linha crua (bytes da codepage, terminador 00 00
// incluído, sem CRLF) em texto UTF-8.
func (tc *TextCodec) DecodeLine(raw []byte) (string, error) {
	out := make([]byte, 0, len(raw)*2+16)
	pass := make([]byte, 0, 64) // segmento corrente do charset

	// flush decodifica o segmento acumulado antes de emitir uma tag, para
	// que as tags não atravessem o charset.
	flush := func() error {
		if len(pass) == 0 {
			return nil
		}
		s, err := tc.Charset.Decode(pass)
		if err != nil {
			return fmt.Errorf("%w: charset: %v", ErrInvalidInput, err)
		}
		out = append(out, s...)
		pass = pass[:0]
		return nil
	}

	prev := byte(0xFF)
	n := len(raw)
	for i := 0; i < n; i++ {
		cur := raw[i]
		written := false

		// (1) 2º byte de multibyte: vai cru para o buffer do charset e
		// currentByte zera (evita ler trails como controles).
		if tc.IsMultibyteSecondByte(prev, cur) {
			pass = append(pass, cur)
			written = true
			cur = 0
		}

		// (2) controles de byte único.
		if !written {
			if tag, ok := tc.controlFwd[cur]; ok {
				if cur == 0 && i+1 < n && raw[i+1] == 0 {
					i++      // terminador 00 00: consome ambos em silêncio,
					prev = 0 // nada é escrito
					continue
				}
				if err := flush(); err != nil {
					return "", err
				}
				out = append(out, tag...)
				written = true
				cur = 0
			}
		}

		// (3) pares (cur,next) consultados por categoria na ordem do
		// decoder; pares de glifo produzem a RUNA direto (sem chaves).
		if !written && i+1 < n {
			if tag, r, isRune, ok := tc.resolveDecodePair(cur, raw[i+1]); ok {
				if err := flush(); err != nil {
					return "", err
				}
				if isRune {
					out = utf8.AppendRune(out, r)
				} else {
					out = tag.AppendText(out)
				}
				i++ // consome o 2º byte do par
				// sem tocar em prev — igual ao fluxo atual
				continue
			}
			// Área user-defined (PUA) do Shift-JIS — depois dos dicts, que
			// têm prioridade (alguns pares F0/F2 são tags de icon no jogo).
			if r, ok := tc.userDefinedPair(cur, raw[i+1]); ok {
				if err := flush(); err != nil {
					return "", err
				}
				out = utf8.AppendRune(out, r)
				i++
				continue
			}
		}

		// (4) 0xFF isolado.
		if !written && cur == 0xFF {
			if err := flush(); err != nil {
				return "", err
			}
			out = append(out, "{B:FF}"...)
			written = true // cur segue 0xFF: prev = 0xFF, como hoje
		}

		// (5) byte comum: acumula para o charset.
		if !written {
			pass = append(pass, cur)
		}
		prev = cur
	}
	if err := flush(); err != nil {
		return "", err
	}
	return string(out), nil
}

// DecodeLines decodifica um bloco de linhas cruas (blob).
func (tc *TextCodec) DecodeLines(raw [][]byte) ([]string, error) {
	out := make([]string, len(raw))
	for i, line := range raw {
		s, err := tc.DecodeLine(line)
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", i, err)
		}
		out[i] = s
	}
	return out, nil
}
