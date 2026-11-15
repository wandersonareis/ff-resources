// Package config concentra a CONFIGURAÇÃO de execução do app FF13: onde os
// dados e as traduções moram, a raiz de instalação de cada jogo e as
// preferências persistidas (enableMods e part ativa).
//
// Nada aqui conhece o filelist, os formatos ou os serviços Wails: é a base que
// os outros pacotes consultam. Os caminhos seguem a convenção do app
// (<bin>/data/partN e <bin>/data/mods/partN), com override por variável de
// ambiente para cópias de release.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"ffresources/internal/families/white/profile/parts"
	"ffresources/internal/shared/appconfig"
)

// NormalizePart devolve o id canônico da part (part1/part2/part3). Ids
// desconhecidos caem no part1, mantendo o comportamento antigo.
func NormalizePart(part string) string {
	return parts.NormalizedPart(part)
}

// NormalizeInternalPath normaliza um caminho interno (relativo a data/partN):
// barras "/", sem a barra inicial e sem "..". Devolve "" para caminho vazio ou
// que tenta escapar da raiz.
func NormalizeInternalPath(id string) string {
	id = filepath.ToSlash(strings.TrimSpace(id))
	id = strings.TrimPrefix(id, "/")
	if id == "" || strings.Contains(id, "..") {
		return ""
	}
	return id
}

// DataRoot devolve a raiz de dados do FF13, RELATIVA AO EXECUTÁVEL — a mesma
// convenção do projeto FFX antigo (bin/data):
//
//	<bin>/data/partN        originais (filelist/white bin)
//	<bin>/data/mods/partN   traduções (7z)
//
// Overrides: FF_RESOURCES_DATA (raiz dos originais) e FF_RESOURCES_MODS
// (raiz das traduções) para uma cópia de release em outra pasta.
func DataRoot(kind string) (string, error) {
	if kind == "data" {
		if env := strings.TrimSpace(os.Getenv("FF_RESOURCES_DATA")); env != "" {
			return env, nil
		}
	} else if kind == "mods" {
		if env := strings.TrimSpace(os.Getenv("FF_RESOURCES_MODS")); env != "" {
			return env, nil
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("localizando o executável: %w", err)
	}
	data := filepath.Join(filepath.Dir(exe), "data")
	if kind == "mods" {
		return filepath.Join(data, "mods"), nil
	}
	return data, nil
}

// PartRoot garante que <data>/partN e <data>/mods/partN existem e devolve o
// diretório.
func PartRoot(kind, part string) (string, error) {
	base, err := DataRoot(kind)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, NormalizePart(part))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// GameRoot devolve a pasta de instalação do jogo da PART. Prioridade:
//
// 1) env FF_RESOURCES_GAME_ROOT (override pontual, ex.: testes manuais);
// 2) arquivo de config local (ff13.config.json) — o mesmo que os testes usam.
func GameRoot(part string) string {
	if env := strings.TrimSpace(os.Getenv("FF_RESOURCES_GAME_ROOT")); env != "" {
		return env
	}
	cfg, _, err := appconfig.Load()
	if err != nil {
		return ""
	}
	return cfg.Root(NormalizePart(part))
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

// Snapshot monta o retrato da config local (sem falhar: devolve vazio).
func Snapshot() ConfigSnapshot {
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

// activePart é a part que a aba principal está mostrando. É estado de
// EXECUÇÃO (não config): a aba manda a part na tela e o snapshot a devolve.
var activePart atomic.Value

// ActivePart devolve a part ativa (default "part1").
func ActivePart() string {
	if v, ok := activePart.Load().(string); ok && v != "" {
		return v
	}
	return "part1"
}

// SetActivePart registra a part ativa (sem emitir evento: quem chama decide
// quando publicar o snapshot).
func SetActivePart(part string) {
	activePart.Store(NormalizePart(part))
}

// EnableModsEnabled devolve a preferência real (default: ligada, mods-first).
func EnableModsEnabled() bool {
	cfg, _, err := appconfig.Load()
	if err != nil || cfg == nil {
		return true
	}
	return cfg.EnableModsEnabled()
}

// SetEnableMods persiste a preferência de mods (sem emitir evento).
func SetEnableMods(enabled bool) error {
	cfg, path, err := appconfig.Load()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = appconfig.Placeholder()
	}
	cfg.SetEnableMods(enabled)
	return appconfig.Save(path, cfg)
}
