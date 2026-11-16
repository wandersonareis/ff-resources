package common

import (
	"ffresources/internal/shared/artifact"
	"ffresources/internal/shared/dto"
)

// ArtifactOptions devolve o perfil de serialização do FFX para uma chamada:
// fonte `us`, alvo `br`, identidade posicional, ordem por índice, dedup
// ligado e a trava de escrita do jogo. A lista de idiomas é o filtro pedido
// pelo chamador (nil/vazio = todos).
func ArtifactOptions(langs []string) formatters.Options {
	return formatters.Options{
		SourceLang:          DefaultLocalization,
		TargetLang:          DefaultTargetLocalization,
		TargetLanguageName:  TranslationLanguage,
		Languages:           langs,
		Identity:            dto.IdentityIndex,
		Ordered:             true,
		CheckWrite:          CheckWritablePath,
		ReservedKeyPrefixes: []string{"ffx", "ffx2", "lastmiss", "eternalcalm"},
	}
}
