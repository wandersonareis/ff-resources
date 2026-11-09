package encoding

import "fmt"

// DecodeLine converte uma linha crua (bytes da codepage, terminador 00 00
// incluído, sem CRLF) em texto UTF-8. Máquina de estados fiel aos
// KeysDecoderLJ/Ch/Kr do C#:
//
//  1. multibyte do charset (IsMultibyteSecondByte): o 2º byte vai cru para o
//     buffer do charset e currentByte zera (evita ler trails como controles);
//  2. SingleKeys: 0x00 seguido de 0x00 é o terminador (consome ambos em
//     silêncio); 0x00 isolado vira "{End}"; 0x01..0x05 viram suas tags;
//  3. pares (cur,next) consultados por categoria na ordem do decoder;
//     pares chara produzem a RUNA direto (sem chaves), tags produz a tag;
//  4. 0xFF isolado vira "{FF}";
//  5. qualquer outro byte vai cru para o buffer do charset.
//
// Diferenças conscientes em relação ao C#: o texto comum é convertido por
// segmentos contíguos (mesmo resultado da conversão global, já que
// segmentos só quebram em fronteiras de caracteres) e os pares chara
// já saem como runas (o C# emitia "{0x85_40}" e reescrevia depois).
func (ks *KeySet) DecodeLine(raw []byte) (string, error) {
	var out []byte  // texto UTF-8 final
	var pass []byte // bytes crus acumulados p/ conversão pelo charset
	flush := func() error {
		if len(pass) == 0 {
			return nil
		}
		s, err := ks.Charset.Decode(pass)
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

		if ks.IsMultibyteSecondByte(prev, cur) {
			pass = append(pass, cur)
			written = true
			cur = 0
		}

		if !written {
			if tag, ok := ks.singleFwd[cur]; ok {
				if cur == 0 && i+1 < n && raw[i+1] == 0 {
					i++      // terminador 00 00: consome ambos,
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

		if !written && i+1 < n {
			if tag, r, isRune, ok := ks.resolveDecodePair(cur, raw[i+1]); ok {
				if err := flush(); err != nil {
					return "", err
				}
				if isRune {
					out = append(out, string(r)...)
				} else {
					out = append(out, tag...)
				}
				i++
				written = true
				cur = 0
				continue
			}
			// Área user-defined (PUA) do Shift-JIS — depois dos dicts, que
			// têm prioridade (alguns pares F0/F2 são tags de icon no jogo).
			if r, ok := ks.userDefinedPair(cur, raw[i+1]); ok {
				if err := flush(); err != nil {
					return "", err
				}
				out = append(out, string(r)...)
				i++
				written = true
				cur = 0
				continue
			}
		}

		if !written && cur == 0xFF {
			if err := flush(); err != nil {
				return "", err
			}
			out = append(out, "{FF}"...)
			written = true
		}

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
func (ks *KeySet) DecodeLines(raw [][]byte) ([]string, error) {
	out := make([]string, len(raw))
	for i, line := range raw {
		s, err := ks.DecodeLine(line)
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", i, err)
		}
		out[i] = s
	}
	return out, nil
}

// EncodeLine converte texto UTF-8 em linha crua (terminador 00 00 incluso).
// Loop por RUNAS (o C# varria bytes da codepage, o que permitia que um
// trail 0x7B fosse confundido com '{'):
//
//   - '{' inicia uma tag: conteúdo até '}' (limite de 32, como o
//     DeriveSymbolString); casada, resolve por single/color/icon/btn/
//     exchara/special/unk/unk2 na ordem do encoder, depois "{FF}";
//     não casada, rewind: emite '{' e reprocessa o conteúdo como texto
//     (o C# descartaria até 33 bytes — aqui nada se perde);
//   - runa ASCII vai a byte único (o reverse do Pulse Core.Encoding aplica
//     índices baixos primeiro: '-' é 0x2D, não o chara 0x85 0x56);
//   - runa chara vira o par de 2 bytes (primeira ocorrência da tabela);
//   - demais runas vão para o charset, em segmentos contíguos.
func (ks *KeySet) EncodeLine(s string) ([]byte, error) {
	rs := []rune(s)
	n := len(rs)
	var out []byte

	for i := 0; i < n; i++ {
		r := rs[i]

		if r == '{' {
			j := i + 1
			for j < n && rs[j] != '}' && j-i <= 32 {
				j++
			}
			if j < n && rs[j] == '}' && j-i-1 <= 32 {
				tag := string(rs[i : j+1])
				if b, ok := ks.EncodeByte(tag); ok {
					out = append(out, b)
					i = j
					continue
				}
				if b1, b2, ok := ks.resolveEncodeTag(tag); ok {
					out = append(out, b1, b2)
					i = j
					continue
				}
				if tag == "{FF}" {
					out = append(out, 0xFF)
					i = j
					continue
				}
				// tag desconhecida: rewind
				out = append(out, '{')
				continue
			}
			// sem '}' próximo: emite '{' e segue (o C# lançaria EOF)
			out = append(out, '{')
			continue
		}

		if r < 0x80 {
			out = append(out, byte(r))
			continue
		}

		if b1, b2, ok := ks.CharaPair(r); ok {
			out = append(out, b1, b2)
			continue
		}

		// runa PUA user-defined -> par cru (o inverso do decode)
		if b1, b2, ok := ks.userDefinedRune(r); ok {
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
			if _, _, ok := ks.CharaPair(r2); ok {
				break
			}
			if _, _, ok := ks.userDefinedRune(r2); ok {
				break
			}
			j++
		}
		bs, err := ks.Charset.Encode(string(rs[i:j]))
		if err != nil {
			return nil, fmt.Errorf("%w: runa %q sem mapeamento na codepage %d (o .NET usaria best-fit)",
				ErrInvalidInput, string(rs[i]), int(ks.Codepage))
		}
		out = append(out, bs...)
		i = j - 1
	}

	out = append(out, 0x00, 0x00)
	return out, nil
}

// EncodeLines codifica um bloco de linhas de texto.
func (ks *KeySet) EncodeLines(lines []string) ([][]byte, error) {
	out := make([][]byte, len(lines))
	for i, line := range lines {
		b, err := ks.EncodeLine(line)
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", i, err)
		}
		out[i] = b
	}
	return out, nil
}
