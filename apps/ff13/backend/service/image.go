package service

import (
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"ffresources/apps/ff13/backend/dialog"
	"ffresources/internal/families/white/profile/config"
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

// SelectImageFile abre o seletor de DDS para importar.
func (FF13Service) SelectImageFile(part, id, version string) string {
	return dialog.PickFile("Importar DDS", imageExportRoot(part),
		application.FileFilter{DisplayName: "Texturas DDS (*.dds)", Pattern: "*.dds"})
}

// SelectImageSavePath abre o diálogo "Salvar imagem".
func (FF13Service) SelectImageSavePath(part, format, suggestedName, id, version string) string {
	return dialog.PickSave("Salvar imagem", imageExportRoot(part), suggestedName,
		application.FileFilter{DisplayName: strings.ToUpper(format), Pattern: "*." + format})
}
