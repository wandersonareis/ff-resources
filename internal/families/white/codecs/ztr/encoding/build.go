package encoding

import "fmt"

// Ordens de consulta por codepage, preservadas fielmente do C#. O chara
// fica no meio da ordem de decode (color, icon, btn, chara, special,
// exchara, unk[, unk2]); o encode consulta exchara antes de special.
var (
	decodeOrderLJ = []string{catColor, catIcon, catBtn, catChara, catSpecial, catExChara, catUnk, catUnk2}
	encodeOrderLJ = []string{catColor, catIcon, catBtn, catChara, catExChara, catSpecial, catUnk, catUnk2}
	decodeOrderCh = []string{catColor, catIcon, catBtn, catChara, catSpecial, catExChara, catUnk}
	encodeOrderCh = []string{catColor, catIcon, catBtn, catChara, catExChara, catSpecial, catUnk}
)

// buildKeySet compõe os dicionários do par (game, codepage) a partir das
// tabelas geradas: comuns (single/chara/exchara/unk/unk2/special-base) +
// variantes por jogo (color/icon/btn) + sufixos por codepage (special).
func buildKeySet(game GameCode, cp Codepage) (*KeySet, error) {
	charset, err := CharsetFor(cp)
	if err != nil {
		return nil, err
	}
	set := &KeySet{
		Game:        game,
		Codepage:    cp,
		Charset:     charset,
		Single:      cmnSingle,
		Chara:       cmnChara,
		ExChara:     cmnExChara,
		Unk:         cmnUnk,
		Unk2:        cmnUnk2,
		charaLead:   charaLeadByCP[cp],
		decodeOrder: decodeOrderLJ,
		encodeOrder: encodeOrderLJ,
	}

	var color, icon, btn map[Codepage][]KVPair
	switch game {
	case FF131:
		color, icon, btn = xiiiColor, xiiiIcon, xiiiBtn
	case FF132:
		color, icon, btn = xiii2Color, xiii2Icon, xiii2Btn
	case FF133:
		color, icon, btn = xiii3Color, xiii3Icon, xiii3Btn
	default:
		return nil, fmt.Errorf("%w: %s", ErrGameCodeNotFound, game)
	}
	set.Color, set.Icon, set.Btn = color[cp], icon[cp], btn[cp]

	// special = base comum + sufixo da codepage (Entity/Counter).
	set.Special = append(append([]KVPair{}, cmnSpecialBase...), specialSuffixByCP[cp]...)

	switch cp {
	case Chinese:
		// KeysDecoderCh/KeysEncoderCh não consultam Unk2Keys.
		set.Unk2 = nil
		set.decodeOrder, set.encodeOrder = decodeOrderCh, encodeOrderCh
	case Korean:
		// mesmas ordens do LJ
	default:
		if cp != LatinJapanese {
			return nil, fmt.Errorf("%w: %d", ErrCodepageNotFound, int(cp))
		}
	}

	buildIndexes(set)
	return set, nil
}
