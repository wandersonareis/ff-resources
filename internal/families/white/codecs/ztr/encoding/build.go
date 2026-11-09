package encoding

import "fmt"

// buildKeySet compõe os dicionários do par (game, codepage) a partir das
// tabelas geradas (formato Tag): variantes por jogo (color/icon/btn) +
// comuns (single, chara, special por codepage, raw, raw2).
func buildKeySet(game GameCode, cp Codepage) (*KeySet, error) {
	charset, err := CharsetFor(cp)
	if err != nil {
		return nil, err
	}
	set := &KeySet{
		Game:      game,
		Codepage:  cp,
		Charset:   charset,
		Single:    cmnSingle,
		Chara:     cmnChara,
		Raw:       cmnRaw,
		Raw2:      cmnRaw2,
		Special:   cmnSpecialByCP[cp],
		charaLead: charaLeadByCP[cp],
	}

	switch game {
	case FF131:
		set.Color, set.Icon, set.Btn = xiiiColor[cp], xiiiIcon[cp], xiiiBtn[cp]
	case FF132:
		set.Color, set.Icon, set.Btn = xiii2Color[cp], xiii2Icon[cp], xiii2Btn[cp]
	case FF133:
		set.Color, set.Icon, set.Btn = xiii3Color[cp], xiii3Icon[cp], xiii3Btn[cp]
	default:
		return nil, fmt.Errorf("%w: %s", ErrGameCodeNotFound, game)
	}

	switch cp {
	case Chinese:
		// KeysDecoderCh/KeysEncoderCh não consultam Unk2Keys.
		set.Raw2 = nil
	case Korean:
		// mesmas regras do LJ
	default:
		if cp != LatinJapanese {
			return nil, fmt.Errorf("%w: %d", ErrCodepageNotFound, int(cp))
		}
	}

	buildIndexes(set)
	return set, nil
}
