package encoding

// ÍNDICES: o que a máquina de estados consulta. Montados uma vez por
// TextCodec (buildIndexes) e imutáveis depois.
//
//   - controlFwd/Rev: controles de byte único (0x00..0x05);
//   - pairFwd: classe -> par de bytes -> Tag (o decode);
//   - nameFwd: classe -> nome -> Tag (validação/autocomplete; o encode de
//     par usa o HEX, não o nome);
//   - glyphFwd/Rev: par <-> rune.
//
// Os reversos guardam a PRIMEIRA ocorrência na ordem de inserção.

func buildIndexes(tc *TextCodec) {
	tc.controlFwd = map[byte]string{}
	tc.controlRev = map[string]byte{}
	tc.pairFwd = map[TagClass]map[uint16]Tag{}
	tc.nameFwd = map[TagClass]map[string]Tag{}
	tc.glyphFwd = map[uint16]rune{}
	tc.glyphRev = map[rune]uint16{}

	for _, e := range tc.Controls {
		tc.controlFwd[e.Key] = e.Value
		if _, dup := tc.controlRev[e.Value]; !dup {
			tc.controlRev[e.Value] = e.Key
		}
	}

	add := func(list []Tag) {
		for _, t := range list {
			byPair := tc.pairFwd[t.Class]
			if byPair == nil {
				byPair = map[uint16]Tag{}
				tc.pairFwd[t.Class] = byPair
			}
			byPair[t.Pair()] = t

			byName := tc.nameFwd[t.Class]
			if byName == nil {
				byName = map[string]Tag{}
				tc.nameFwd[t.Class] = byName
			}
			if _, dup := byName[t.Name]; !dup {
				byName[t.Name] = t
			}
		}
	}

	add(tc.Colors)
	add(tc.Icons)
	add(tc.Buttons)
	add(tc.Specials)
	add(tc.Raw)
	if tc.RawExtra != nil {
		add(tc.RawExtra)
	}

	for _, e := range tc.Glyphs {
		k := pairKey(tc.glyphLead, e.Trail)
		tc.glyphFwd[k] = e.Rune
		// O reverse só recebe runas não-ASCII: texto ASCII comum (ex.: o
		// hífen 0x2D) codifica como byte único, nunca como par de glifo.
		if e.Rune >= 0x80 {
			if _, dup := tc.glyphRev[e.Rune]; !dup {
				tc.glyphRev[e.Rune] = k
			}
		}
	}
}

// resolveDecodePair devolve a Tag do par (b1,b2) consultando as classes na
// ordem do decoder, com os glifos no meio — parando no primeiro hit.
func (tc *TextCodec) resolveDecodePair(b1, b2 byte) (tag Tag, r rune, isRune, ok bool) {
	k := pairKey(b1, b2)
	for _, class := range classesBeforeGlyphs {
		if m, has := tc.pairFwd[class]; has {
			if t, hit := m[k]; hit {
				return t, 0, false, true
			}
		}
	}
	if v, hit := tc.glyphFwd[k]; hit {
		return Tag{}, v, true, true
	}
	for _, class := range classesAfterGlyphs {
		if m, has := tc.pairFwd[class]; has {
			if t, hit := m[k]; hit {
				return t, 0, false, true
			}
		}
	}
	return Tag{}, 0, false, false
}

// tagByName devolve a Tag de uma classe pelo NOME (validação/autocomplete).
func (tc *TextCodec) tagByName(class TagClass, name string) (Tag, bool) {
	if m, has := tc.nameFwd[class]; has {
		t, ok := m[name]
		return t, ok
	}
	return Tag{}, false
}

// tagByPair devolve a Tag de um par em qualquer classe (validação/log).
func (tc *TextCodec) tagByPair(b1, b2 byte) (Tag, bool) {
	k := pairKey(b1, b2)
	for _, class := range append(append([]TagClass{}, classesBeforeGlyphs...), classesAfterGlyphs...) {
		if m, has := tc.pairFwd[class]; has {
			if t, hit := m[k]; hit {
				return t, true
			}
		}
	}
	return Tag{}, false
}

// GlyphBytes devolve o par de uma runa (1ª ocorrência na ordem de inserção).
func (tc *TextCodec) GlyphBytes(r rune) (byte, byte, bool) {
	v, ok := tc.glyphRev[r]
	if !ok {
		return 0, 0, false
	}
	return byte(v >> 8), byte(v), true
}

// DecodeControl resolve byte único -> tag.
func (tc *TextCodec) DecodeControl(b byte) (string, bool) {
	v, ok := tc.controlFwd[b]
	return v, ok
}

// EncodeControl resolve tag -> byte único.
func (tc *TextCodec) EncodeControl(tag string) (byte, bool) {
	v, ok := tc.controlRev[tag]
	return v, ok
}
