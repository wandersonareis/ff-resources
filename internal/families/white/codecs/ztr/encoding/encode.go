package encoding

import "fmt"

// ENCODE: texto UTF-8 (com as tags) -> bytes da linha, terminada em 00 00.
//
// Loop por RUNAS:
//   - '{' inicia uma tag: conteúdo até '}' (limite de 32); casada, resolve
//     por byte único/classe; não casada, rewind: emite '{' e reprocessa o
//     conteúdo como texto;
//   - runa ASCII vai a byte único (o reverse aplica índices baixos primeiro:
//     '-' é 0x2D, não o glifo 0x85 0x56);
//   - runa de glifo vira o par de 2 bytes (primeira ocorrência da tabela);
//   - demais runas vão para o charset, em segmentos contíguos.
//
// O parse da tag está isolado em tag_codec.go (encodeTag).

// EncodeLine converte texto UTF-8 em linha crua (terminador 00 00 incluso).
func (tc *TextCodec) EncodeLine(s string) ([]byte, error) {
	rs := []rune(s)
	n := len(rs)
	out := make([]byte, 0, len(s)+2)

	for i := 0; i < n; i++ {
		r := rs[i]

		if r == '{' {
			if end, ok := closeBrace(rs, i); ok {
				if b, ok := tc.encodeTag(string(rs[i+1 : end])); ok {
					out = append(out, b...)
					i = end // pula o '}'
					continue
				}
			}
			// rewind: tag desconhecida ou sem '}' próximo — emite '{' e
			// reprocessa o conteúdo como texto.
			out = append(out, '{')
			continue
		}

		// runa ASCII: byte único (não passa pelo charset).
		if r < 0x80 {
			out = append(out, byte(r))
			continue
		}

		// runa de glifo: par de 2 bytes (primeira ocorrência da tabela).
		if b1, b2, ok := tc.GlyphBytes(r); ok {
			out = append(out, b1, b2)
			continue
		}

		// runa PUA user-defined -> par cru (o inverso do decode).
		if b1, b2, ok := tc.userDefinedRune(r); ok {
			out = append(out, b1, b2)
			continue
		}

		// segmento contíguo para o charset
		j := i + 1
		for j < n {
			r2 := rs[j]
			if r2 == '{' || r2 < 0x80 {
				break
			}
			if _, _, ok := tc.GlyphBytes(r2); ok {
				break
			}
			if _, _, ok := tc.userDefinedRune(r2); ok {
				break
			}
			j++
		}
		bs, err := tc.Charset.Encode(string(rs[i:j]))
		if err != nil {
			return nil, fmt.Errorf("%w: runa %q sem mapeamento na codepage %d (o .NET usaria best-fit)",
				ErrInvalidInput, string(rs[i]), int(tc.Codepage))
		}
		out = append(out, bs...)
		i = j - 1
	}

	out = append(out, 0x00, 0x00)
	return out, nil
}

// EncodeLines codifica um bloco de linhas de texto.
func (tc *TextCodec) EncodeLines(lines []string) ([][]byte, error) {
	out := make([][]byte, len(lines))
	for i, line := range lines {
		b, err := tc.EncodeLine(line)
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", i, err)
		}
		out[i] = b
	}
	return out, nil
}
