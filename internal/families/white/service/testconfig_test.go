package service

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/shared/appconfig"
)

// partDataRoot devolve a pasta data/partN do app (relativa ao package do
// teste, que é a raiz do app ff13).
func partDataRoot(t *testing.T, part string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", part)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("data/%s ausente: %v", part, err)
	}
	return dir
}

// partModsRoot devolve a pasta mods/partN do app (relativa ao package do
// teste).
func partModsRoot(t *testing.T, part string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data", "mods", part)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("mods/%s ausente: %v", part, err)
	}
	return dir
}

// testGameRoot devolve a raiz de instalação da part lida do ARQUIVO DE CONFIG
// local (ff13.config.json — o mesmo que o app usa). Skippa o teste quando o
// caminho não está configurado ou o jogo não está instalado.
func testGameRoot(t *testing.T, part string) string {
	t.Helper()
	cfg, path, err := appconfig.Load()
	if err != nil {
		t.Skipf("config invalida (%s): %v", path, err)
	}
	root := cfg.Root(part)
	if root == "" {
		t.Skipf("config sem raiz para %s (%s)", part, path)
	}
	if _, err := os.Stat(root); err != nil {
		t.Skipf("jogo %s nao instalado em %s: %v", part, root, err)
	}
	return root
}
