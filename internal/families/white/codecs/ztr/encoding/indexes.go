package encoding

// ÍNDICES: o que a máquina de estados consulta. Montados uma vez por
// KeySet (buildIndexes) e imutáveis depois.
//
//   - singleFwd/Rev: controles de byte único (0x00..0x05);
//   - pairFwd: classe -> par de bytes -> Tag (o decode);
//   - nameFwd: classe -> nome -> Tag (o encode do COLOR, a única classe
//     resolvida por nome — as outras carregam o hex no texto);
//   - charaFwd/Rev: par <-> runa.
//
// Os reversos guardam a PRIMEIRA ocorrência na ordem de inserção (espelham o
// GetDictByteKey do C#), e a ordem dos slices gerados é a ordem de inserção
// do fonte C#.

func buildIndexes(set *KeySet) {
	set.singleFwd = map[byte]string{}
	set.singleRev = map[string]byte{}
	set.pairFwd = map[TagClass]map[uint16]Tag{}
	set.nameFwd = map[TagClass]map[string]Tag{}
	set.charaFwd = map[uint16]rune{}
	set.charaRev = map[rune]uint16{}

	for _, e := range set.Single {
		set.singleFwd[e.Key] = e.Value
		if _, dup := set.singleRev[e.Value]; !dup {
			set.singleRev[e.Value] = e.Key
		}
	}

	add := func(list []Tag) {
		for _, t := range list {
			byPair := set.pairFwd[t.Class]
			if byPair == nil {
				byPair = map[uint16]Tag{}
				set.pairFwd[t.Class] = byPair
			}
			byPair[t.Pair()] = t

			byName := set.nameFwd[t.Class]
			if byName == nil {
				byName = map[string]Tag{}
				set.nameFwd[t.Class] = byName
			}
			if _, dup := byName[t.Name]; !dup {
				byName[t.Name] = t
			}
		}
	}

	add(set.Color)
	add(set.Icon)
	add(set.Btn)
	add(set.Special)
	add(set.Raw)
	if set.Raw2 != nil {
		add(set.Raw2)
	}

	for _, e := range set.Chara {
		k := pairKey(set.charaLead, e.Trail)
		set.charaFwd[k] = e.Rune
		// O reverse só recebe runas não-ASCII: texto ASCII comum (ex.: o
		// hífen 0x2D) codifica como byte único, nunca como par chara.
		if e.Rune >= 0x80 {
			if _, dup := set.charaRev[e.Rune]; !dup {
				set.charaRev[e.Rune] = k
			}
		}
	}
}

// resolveDecodePair devolve a Tag do par (b1,b2) consultando as classes na
// ordem do decoder, com o chara no meio — parando no primeiro hit, igual às
// condições encadeadas do C#.
func (ks *KeySet) resolveDecodePair(b1, b2 byte) (tag Tag, r rune, isRune, ok bool) {
	k := pairKey(b1, b2)
	for _, class := range pairPreChara {
		if m, has := ks.pairFwd[class]; has {
			if t, hit := m[k]; hit {
				return t, 0, false, true
			}
		}
	}
	if v, hit := ks.charaFwd[k]; hit {
		return Tag{}, v, true, true
	}
	for _, class := range pairPostChara {
		if m, has := ks.pairFwd[class]; has {
			if t, hit := m[k]; hit {
				return t, 0, false, true
			}
		}
	}
	return Tag{}, 0, false, false
}

// tagByName devolve a Tag de uma classe pelo NOME (usado pelo COLOR).
func (ks *KeySet) tagByName(class TagClass, name string) (Tag, bool) {
	if m, has := ks.nameFwd[class]; has {
		t, ok := m[name]
		return t, ok
	}
	return Tag{}, false
}

// tagByPair devolve a Tag de um par em qualquer classe (validação/log).
func (ks *KeySet) tagByPair(b1, b2 byte) (Tag, bool) {
	k := pairKey(b1, b2)
	for _, class := range append(append([]TagClass{}, pairPreChara...), pairPostChara...) {
		if m, has := ks.pairFwd[class]; has {
			if t, hit := m[k]; hit {
				return t, true
			}
		}
	}
	return Tag{}, false
}

// CharaPair devolve o par chara de uma runa (1ª ocorrência na ordem de
// inserção, como GetDictByteKey).
func (ks *KeySet) CharaPair(r rune) (byte, byte, bool) {
	v, ok := ks.charaRev[r]
	if !ok {
		return 0, 0, false
	}
	return byte(v >> 8), byte(v), true
}

// DecodeByte resolve byte único -> tag (SingleKeys).
func (ks *KeySet) DecodeByte(b byte) (string, bool) {
	v, ok := ks.singleFwd[b]
	return v, ok
}

// EncodeByte resolve tag -> byte único (SingleKeys).
func (ks *KeySet) EncodeByte(tag string) (byte, bool) {
	v, ok := ks.singleRev[tag]
	return v, ok
}
