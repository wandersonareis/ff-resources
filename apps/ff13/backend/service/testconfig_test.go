package service

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/apps/ff13/backend/formats"
	"ffresources/internal/families/white/codecs/wpd"
	"ffresources/internal/families/white/codecs/ztr"
	"ffresources/internal/shared/appconfig"
)

// testZtrBlob monta um .ztr com os IDs/linhas pedidos (charset do part1).
func testZtrBlob(t *testing.T, ids []string, texts []string) []byte {
	t.Helper()
	ks, err := formats.TextCodec("part1")
	if err != nil {
		t.Fatalf("TextCodec: %v", err)
	}
	lines, err := ks.EncodeLines(texts)
	if err != nil {
		t.Fatalf("EncodeLines: %v", err)
	}
	blob, err := ztr.WriteTie(&ztr.File{IDs: ids, Lines: lines}, ztr.Compressed, ztr.TieMinLast)
	if err != nil {
		t.Fatalf("WriteTie: %v", err)
	}
	return blob
}

// writeWpd grava um container WPD com os registros pedidos.
func writeWpd(t *testing.T, path string, parts []wpd.Part) {
	t.Helper()
	blob, err := wpd.WriteFromParts(parts)
	if err != nil {
		t.Fatalf("WriteFromParts %s: %v", path, err)
	}
	writeFile(t, path, blob)
}

// writeFile grava o arquivo do teste criando os diretórios.
func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// partDataRoot devolve a pasta data/partN do app. Os testes rodam no
// diretório do pacote (backend/service), então o bin/ do app fica três
// níveis acima.
func partDataRoot(t *testing.T, part string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "bin", "data", part)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("data/%s ausente: %v", part, err)
	}
	return dir
}

// partModsRoot devolve a pasta mods/partN do app (relativa ao package do
// teste).
func partModsRoot(t *testing.T, part string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "bin", "data", "mods", part)
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
