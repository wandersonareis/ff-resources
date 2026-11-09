package encoding

import (
	"fmt"
	"sync"
)

// Registro de KeySets por (game, codepage), no espírito do ffxencoding:
// publicação por substituição de referência, acesso concorrente seguro e
// fail-fast em Prepare/Ensure (configuração ausente aborta antes de
// produzir milhares de placeholders).
var (
	keySetsMu sync.RWMutex
	keySets   = map[GameCode]map[Codepage]*KeySet{}
)

func pairKey(b1, b2 byte) uint16 { return uint16(b1)<<8 | uint16(b2) }

// PrepareKeySet monta os índices do par (game, codepage) a partir das tabelas
// geradas e publica o conjunto. Idempotente: rechamar sobrescreve.
//
// Forward é muitos-para-um invertido (byte→tag). Reverse (tag→byte) guarda a
// PRIMEIRA ocorrência na ordem de inserção — espelha o GetDictByteKey do C#,
// que percorre o Dictionary na ordem de inserção e para no primeiro valor
// igual; a ordem dos slices gerados é a ordem de inserção do fonte C#.
func PrepareKeySet(game GameCode, cp Codepage) error {
	set, err := buildKeySet(game, cp)
	if err != nil {
		return err
	}
	keySetsMu.Lock()
	defer keySetsMu.Unlock()
	if keySets[game] == nil {
		keySets[game] = map[Codepage]*KeySet{}
	}
	keySets[game][cp] = set
	return nil
}

// EnsureKeySet é o fail-fast: só leitura, não monta nada.
func EnsureKeySet(game GameCode, cp Codepage) error {
	keySetsMu.RLock()
	defer keySetsMu.RUnlock()
	if _, ok := keySets[game]; !ok {
		return fmt.Errorf("%w: %s", ErrGameCodeNotFound, game)
	}
	if _, ok := keySets[game][cp]; !ok {
		return fmt.Errorf("%w: %d (game=%s)", ErrCodepageNotFound, cp, game)
	}
	return nil
}

// KeySetFor devolve o conjunto publicado (ou erro de configuração).
func KeySetFor(game GameCode, cp Codepage) (*KeySet, error) {
	keySetsMu.RLock()
	defer keySetsMu.RUnlock()
	if _, ok := keySets[game]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrGameCodeNotFound, game)
	}
	set, ok := keySets[game][cp]
	if !ok {
		return nil, fmt.Errorf("%w: %d (game=%s)", ErrCodepageNotFound, cp, game)
	}
	return set, nil
}

// ClearKeySets limpa o registro (testes).
func ClearKeySets() {
	keySetsMu.Lock()
	defer keySetsMu.Unlock()
	keySets = map[GameCode]map[Codepage]*KeySet{}
}
