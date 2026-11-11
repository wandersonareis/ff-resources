package encoding

import (
	"fmt"
	"sync"
)

// Registro de TextCodecs por (game, codepage): publicação por substituição de
// referência, acesso concorrente seguro e fail-fast em Prepare/Ensure
// (configuração ausente aborta antes de produzir milhares de placeholders).
var (
	textCodecsMu sync.RWMutex
	textCodecs   = map[GameCode]map[Codepage]*TextCodec{}
)

func pairKey(b1, b2 byte) uint16 { return uint16(b1)<<8 | uint16(b2) }

// PrepareTextCodec monta os índices do par (game, codepage) a partir das
// tabelas do pacote tags e publica o conjunto. Idempotente.
func PrepareTextCodec(game GameCode, cp Codepage) error {
	tc, err := buildTextCodec(game, cp)
	if err != nil {
		return err
	}
	textCodecsMu.Lock()
	defer textCodecsMu.Unlock()
	if textCodecs[game] == nil {
		textCodecs[game] = map[Codepage]*TextCodec{}
	}
	textCodecs[game][cp] = tc
	return nil
}

// EnsureTextCodec é o fail-fast: só leitura, não monta nada.
func EnsureTextCodec(game GameCode, cp Codepage) error {
	textCodecsMu.RLock()
	defer textCodecsMu.RUnlock()
	if _, ok := textCodecs[game]; !ok {
		return fmt.Errorf("%w: %s", ErrGameCodeNotFound, game)
	}
	if _, ok := textCodecs[game][cp]; !ok {
		return fmt.Errorf("%w: %d (game=%s)", ErrCodepageNotFound, cp, game)
	}
	return nil
}

// TextCodecFor devolve o conjunto publicado (ou erro de configuração).
func TextCodecFor(game GameCode, cp Codepage) (*TextCodec, error) {
	textCodecsMu.RLock()
	defer textCodecsMu.RUnlock()
	if _, ok := textCodecs[game]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrGameCodeNotFound, game)
	}
	tc, ok := textCodecs[game][cp]
	if !ok {
		return nil, fmt.Errorf("%w: %d (game=%s)", ErrCodepageNotFound, cp, game)
	}
	return tc, nil
}

// ClearTextCodecs limpa o registro (testes).
func ClearTextCodecs() {
	textCodecsMu.Lock()
	defer textCodecsMu.Unlock()
	textCodecs = map[GameCode]map[Codepage]*TextCodec{}
}
