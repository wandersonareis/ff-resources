package encoding

// buildIndexes monta os mapas forward/reverse POR CATEGORIA. O reverse guarda
// a PRIMEIRA ocorrência na ordem de inserção (espelha o GetDictByteKey do C#).
// O chara tem índices próprios: par -> runa e runa -> par.
func buildIndexes(set *KeySet) {
	set.singleFwd = map[byte]string{}
	set.singleRev = map[string]byte{}
	set.pairFwd = map[string]map[uint16]string{}
	set.pairRev = map[string]map[string]uint16{}
	set.charaFwd = map[uint16]rune{}
	set.charaRev = map[rune]uint16{}

	for _, e := range set.Single {
		set.singleFwd[e.Key] = e.Value
		if _, dup := set.singleRev[e.Value]; !dup {
			set.singleRev[e.Value] = e.Key
		}
	}

	addPair := func(cat string, s []KVPair) {
		if s == nil {
			return
		}
		fwd := make(map[uint16]string, len(s))
		rev := make(map[string]uint16, len(s))
		for _, e := range s {
			k := pairKey(e.B1, e.B2)
			fwd[k] = e.Value
			if _, dup := rev[e.Value]; !dup {
				rev[e.Value] = k
			}
		}
		set.pairFwd[cat] = fwd
		set.pairRev[cat] = rev
	}

	addPair(catColor, set.Color)
	addPair(catIcon, set.Icon)
	addPair(catBtn, set.Btn)
	addPair(catExChara, set.ExChara)
	addPair(catSpecial, set.Special)
	addPair(catUnk, set.Unk)
	addPair(catUnk2, set.Unk2)

	for _, e := range set.Chara {
		k := pairKey(set.charaLead, e.Trail)
		set.charaFwd[k] = e.Rune
		// O reverse só recebe runas não-ASCII: texto ASCII comum (ex.: o
		// hífen 0x2D) codifica como byte único, nunca como par chara —
		// mesmo princípio do reverse do Pulse Core.Encoding, que prefere
		// os índices baixos.
		if e.Rune >= 0x80 {
			if _, dup := set.charaRev[e.Rune]; !dup {
				set.charaRev[e.Rune] = k
			}
		}
	}
}

// resolveDecodePair percorre a ordem do decoder e devolve a tag estrutural
// OU a runa chara do par (b1,b2), parando no primeiro hit — igual às
// condições encadeadas do C#.
func (ks *KeySet) resolveDecodePair(b1, b2 byte) (tag string, r rune, isRune, ok bool) {
	k := pairKey(b1, b2)
	for _, cat := range ks.decodeOrder {
		if cat == catChara {
			if v, hit := ks.charaFwd[k]; hit {
				return "", v, true, true
			}
			continue
		}
		if m, has := ks.pairFwd[cat]; has {
			if v, hit := m[k]; hit {
				return v, 0, false, true
			}
		}
	}
	return "", 0, false, false
}

// resolveEncodeTag percorre a ordem do encoder para uma tag "{...}":
// devolve (b1, b2, true) no primeiro hit.
func (ks *KeySet) resolveEncodeTag(tag string) (byte, byte, bool) {
	for _, cat := range ks.encodeOrder {
		if cat == catChara {
			continue // chara não é tag: é resolvido por runa no encoder
		}
		if m, has := ks.pairRev[cat]; has {
			if v, hit := m[tag]; hit {
				return byte(v >> 8), byte(v), true
			}
		}
	}
	return 0, 0, false
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
