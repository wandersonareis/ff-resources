package main

import (
	"os"
	"strings"

	"ffresources/apps/ff13/parts"
	"ffresources/internal/shared/appconfig"
)

// gameRoot devolve a pasta de instalação do jogo da PART. Prioridade:
//
// 1) env FF_RESOURCES_GAME_ROOT (override pontual, ex.: testes manuais);
// 2) arquivo de config local (ff13.config.json) — o mesmo que os testes usam.
func gameRoot(part string) string {
	if env := strings.TrimSpace(os.Getenv("FF_RESOURCES_GAME_ROOT")); env != "" {
		return env
	}
	cfg, _, err := appconfig.Load()
	if err != nil {
		return ""
	}
	return cfg.Root(parts.NormalizedPart(part))
}

// ConfigSnapshot é o que a interface precisa saber sobre a configuração local:
// onde está o arquivo, as raízes dos jogos e os binários apontados.
type ConfigSnapshot struct {
	// Path do arquivo de config em uso.
	Path string `json:"path"`
	// Roots: pasta de instalação por part.
	Roots map[string]string `json:"roots"`
	// Binaries: nome -> caminho de binários configurados.
	Binaries map[string]string `json:"binaries"`
}

// configSnapshot monta o retrato da config local (sem falhar: devolve vazio).
func configSnapshot() ConfigSnapshot {
	cfg, path, _ := appconfig.Load()
	snap := ConfigSnapshot{
		Path:     path,
		Roots:    map[string]string{},
		Binaries: map[string]string{},
	}
	if cfg == nil {
		return snap
	}
	for k, v := range cfg.Roots {
		snap.Roots[k] = v
	}
	for k, v := range cfg.Binaries {
		snap.Binaries[k] = v
	}
	return snap
}
