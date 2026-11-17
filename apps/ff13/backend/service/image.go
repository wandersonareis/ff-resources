package service

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"ffresources/apps/ff13/backend/dialog"
	"ffresources/internal/families/white/profile/config"
	whiteservice "ffresources/internal/families/white/service"
)

// SELETORES de imagem do app. O núcleo (internal/families/white/service) só
// lê/grava; quem abre o diálogo nativo é o app.

// imageExportRoot é a raiz padrão dos exports de imagem (mods/edits/images).
func imageExportRoot(part string) string {
	root, err := config.DataRoot("mods")
	if err != nil {
		return ""
	}
	return filepath.Join(root, "edits", "images", config.NormalizePart(part))
}

// SelectImageExportDir escolhe a pasta raiz do "Extrair para…".
func (FF13Service) SelectImageExportDir(part string) string {
	return dialog.PickDirectory("Extrair imagens para…", imageExportRoot(part))
}

// SelectImageFile abre o seletor de DDS para importar. A pasta inicial é a de
// trabalho da IMAGEM ABERTA (container/arquivo direto) — o arquivo escolhido
// pode vir de qualquer lugar; a origem serve só de conveniência.
func (FF13Service) SelectImageFile(part, id, version string) string {
	dir := imageExportRoot(part)
	if strings.TrimSpace(id) != "" {
		if path, err := (whiteservice.Service{}).ImageExportPath(part, id, "dds"); err == nil {
			if candidate := filepath.Dir(path); dirExists(candidate) {
				dir = candidate
			}
		}
	}
	return dialog.PickFile("Importar DDS", dir,
		application.FileFilter{DisplayName: "Texturas DDS (*.dds)", Pattern: "*.dds"})
}

// dirExists diz se o caminho é um diretório existente.
func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// SelectImageSavePath abre o diálogo "Salvar imagem".
func (FF13Service) SelectImageSavePath(part, format, suggestedName, id, version string) string {
	return dialog.PickSave("Salvar imagem", imageExportRoot(part), suggestedName,
		application.FileFilter{DisplayName: strings.ToUpper(format), Pattern: "*." + format})
}
