package encoding

import (
	"errors"

	"ffresources/internal/families/white/codecs/ztr/encoding/tags"
)

// Sentinelas distinguindo configuração ausente (aborta) de chave ausente
// no dado (o chamador decide: tag literal ou placeholder).
var (
	ErrGameCodeNotFound  = tags.ErrUnknownGame
	ErrCodepageNotFound  = tags.ErrUnknownCodepage
	ErrKeyNotFound       = errors.New("chave não encontrada no dicionário")
	ErrDuplicateKeyValue = errors.New("valor duplicado no dicionário (encode ambíguo)")
	ErrInvalidInput      = errors.New("linha inválida")
	// ErrPaletteUnavailable é o fail-fast de idioma sem paleta: o catálogo
	// não pode oferecer cores com bytes de outro idioma.
	ErrPaletteUnavailable = errors.New("paleta de cores indisponível")
)
