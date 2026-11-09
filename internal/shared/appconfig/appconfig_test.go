package appconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/shared/appconfig"
)

// TestLoadCreatesPlaceholder garante que o arquivo de config é criado com
// placeholders quando não existe (e nunca é sobrescrito depois).
func TestLoadCreatesPlaceholder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ff13.config.json")
	t.Setenv(appconfig.EnvPath, path)

	cfg, gotPath, err := appconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if gotPath != path {
		t.Fatalf("path=%q, esperado %q", gotPath, path)
	}
	if cfg.Root("part1") == "" {
		t.Fatal("placeholder sem raiz para part1")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("arquivo de config nao criado: %v", err)
	}

	// Segunda carga: lê o que está no arquivo, sem recriar.
	cfg.Roots["part1"] = `D:\Games\FFXIII`
	if err := appconfig.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	again, _, err := appconfig.Load()
	if err != nil {
		t.Fatalf("Load 2: %v", err)
	}
	if again.Root("part1") != `D:\Games\FFXIII` {
		t.Fatalf("root=%q", again.Root("part1"))
	}
}

// TestLoadInvalid devolve erro (sem sobrescrever) quando o JSON é inválido.
func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ff13.config.json")
	if err := os.WriteFile(path, []byte("{ nao é json"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(appconfig.EnvPath, path)
	if _, _, err := appconfig.Load(); err == nil {
		t.Fatal("esperava erro de config invalida")
	}
}
