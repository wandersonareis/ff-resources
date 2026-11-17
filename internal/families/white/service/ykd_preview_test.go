package service

import (
	"os"
	"path/filepath"
	"testing"

	"ffresources/internal/families/white/profile/config"
	"ffresources/internal/shared/dto"
)

// TestYkdScreenRealTitle0 valida o parser + resolução de texturas contra o
// container title0 que o repositório usa nos testes de imagem. Sem dados,
// vira skip.
func TestYkdScreenRealTitle0(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "apps", "ff13", "bin", "data"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "part1", "gui", "resident", "title0.win32.xgr")); err != nil {
		t.Skip("dados do FF13 ausentes")
	}
	t.Setenv("FF_RESOURCES_DATA", root)
	t.Setenv("FF_RESOURCES_MODS", t.TempDir())
	// A config do app (quando existe) permite o fallback entre containers
	// (title0 referencia texturas de title1). Sem ela, o teste ainda vale:
	// as texturas locais bastam para a prévia básica.
	if configPath, cerr := filepath.Abs(filepath.Join("..", "..", "..", "..", "apps", "ff13", "ff13.config.json")); cerr == nil {
		if _, serr := os.Stat(configPath); serr == nil {
			t.Setenv("FF_RESOURCES_CONFIG", configPath)
		}
	}
	t.Setenv("FF_RESOURCES_GAME_ROOT", "")
	InvalidateYkdCaches("part1")

	ref := EncodeImageRef(ImageRef{
		Source:   "data",
		Path:     "gui/resident/title0.win32.xgr",
		Entry:    "rec:gr_title_parts.ykd",
		Payload:  "gui/resident/title0.win32.imgb",
		Platform: "win32",
	})
	screen, err := (Service{}).GetYkdScreen("part1", ref)
	if err != nil {
		t.Fatalf("GetYkdScreen: %v", err)
	}
	if screen.Stats.Blocks == 0 || screen.Stats.Resources == 0 || screen.Stats.Items == 0 {
		t.Fatalf("tela vazia: %+v", screen.Stats)
	}
	if screen.Stats.Textures == 0 {
		t.Fatalf("nenhuma textura resolvida: missing=%v", screen.MissingTextures)
	}
	for _, item := range screen.Items {
		if item.Kind != "sprite" {
			continue
		}
		if item.Texture != "" {
			texture, ok := screen.Textures[item.Texture]
			if !ok || texture.PNGData == "" {
				t.Fatalf("item com textura %q sem PNG", item.Texture)
			}
		}
		if item.ViewWidth == 0 || item.ViewHeight == 0 {
			t.Fatalf("item sem tamanho de exibição: %+v", item)
		}
	}
	t.Logf("title0: blocks=%d resources=%d items=%d textures=%d missing=%d text=%d",
		screen.Stats.Blocks, screen.Stats.Resources, screen.Stats.Items,
		screen.Stats.Textures, screen.Stats.MissingTextures, screen.Stats.TextNodes)

	// Edição de uma caixa de texto: grava o container em mods/ e a prévia
	// recarregada reflete a nova largura.
	textIndex := -1
	for i, block := range screen.Blocks {
		if block.Text != nil {
			textIndex = i
			break
		}
	}
	if textIndex < 0 {
		t.Fatal("nenhum nó de texto para editar")
	}
	original := *screen.Blocks[textIndex].Text
	patch := dto.YkdTextPatch{BlockIndex: textIndex, Width: original.Width + 137}
	updated, err := (Service{}).ApplyYkdText("part1", ref, patch)
	if err != nil {
		t.Fatalf("ApplyYkdText: %v", err)
	}
	if got := updated.Blocks[textIndex].Text.Width; got != patch.Width {
		t.Fatalf("largura não atualizada: got=%d want=%d", got, patch.Width)
	}
	modsRoot, _ := config.PartRoot("mods", "part1")
	modsFile := filepath.Join(modsRoot, "gui", "resident", "title0.win32.xgr")
	if _, serr := os.Stat(modsFile); serr != nil {
		t.Fatalf("container não gravado em mods: %v", serr)
	}

	// Acima do teto (2× a largura original/limite da tela) é recusado.
	if _, err := (Service{}).ApplyYkdText("part1", ref, dto.YkdTextPatch{
		BlockIndex: textIndex,
		Width:      original.Width*3 + 1,
	}); err == nil {
		t.Fatalf("largura acima do teto deveria falhar (original=%d)", original.Width)
	}
}
