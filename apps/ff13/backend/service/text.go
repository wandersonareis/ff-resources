package service

import (
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"ffresources/apps/ff13/backend/dialog"
	"ffresources/internal/families/white/profile/config"
)

// SelectImportFile abre o seletor de UM artefato de texto (JSON/Strings/CSV/
// XLSX/PO). O diretório inicial é o edits/ do export; se ele não existir,
// cai no translated/text/<part> e, por fim, na raiz de mods.
func (FF13Service) SelectImportFile(part string) string {
	part = config.NormalizePart(part)
	root, err := config.DataRoot("mods")
	if err != nil {
		return dialog.PickFile("Importar textos…", "", textImportFilters()...)
	}
	dir := filepath.Join(root, "edits")
	if !dirExists(dir) {
		dir = filepath.Join(root, "translated", "text", part)
	}
	if !dirExists(dir) {
		dir = root
	}
	return dialog.PickFile("Importar textos…", dir, textImportFilters()...)
}

func textImportFilters() []application.FileFilter {
	return []application.FileFilter{
		{DisplayName: "JSON (*.json)", Pattern: "*.json"},
		{DisplayName: "Strings (*.strings)", Pattern: "*.strings"},
		{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
		{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		{DisplayName: "PO (*.po)", Pattern: "*.po"},
		{DisplayName: "Todos os arquivos (*.*)", Pattern: "*.*"},
	}
}
