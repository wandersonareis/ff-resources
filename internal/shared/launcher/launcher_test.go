package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

// TestManifestRaizValido valida o manifest.json da raiz do repositorio: e ele
// que guia a tela do launcher, e os campos `exe` tem de bater com os nomes de
// distribuicao gerados pelos builds (apps/ffx/bin/spira.exe, apps/ff13/bin/pulse.exe),
// que o stage:launcher copia para apps/launcher/bin/<id>/<exe>.
func TestManifestRaizValido(t *testing.T) {
	m, err := LoadManifest(filepath.Join("..", "..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Apps) != 2 {
		t.Fatalf("esperava 2 apps no manifest, veio %d", len(m.Apps))
	}

	wantExe := map[string]string{
		"ffx":  "ffx/spira.exe",
		"ff13": "ff13/pulse.exe",
	}
	for _, app := range m.Apps {
		if want, ok := wantExe[app.ID]; ok {
			if app.Exe != want {
				t.Errorf("app %q: exe = %q, esperava %q (nome de distribuicao do build)", app.ID, app.Exe, want)
			}
			if app.Cover == "" {
				t.Errorf("app %q: sem cover", app.ID)
			}
			continue
		}
		t.Errorf("app desconhecido no manifest: %q", app.ID)
	}
}

// TestLoadManifestInvalido garante que um JSON quebrado devolve erro, nao panic.
func TestLoadManifestInvalido(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte("{ nao eh json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("esperava erro para manifest invalido")
	}
}
