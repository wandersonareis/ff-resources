// Package dialog concentra os SELETORES NATIVOS do app FFX (pasta, arquivo e
// "Salvar como"). Os serviços Wails só pedem o caminho; a decisão de abrir o
// diálogo nativo fica neste pacote — inclusive o nil-safe fora do ciclo de
// vida do app (testes/CLI).
package dialog

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// PickDirectory abre o seletor nativo de PASTA com o título e a pasta inicial
// pedidos. Devolve "" fora do ciclo de vida do app e no cancelamento.
//
// O Wails v3 não tem OpenDirectoryDialog: escolher pasta é o OpenFileDialog
// com CanChooseDirectories(true). CanChooseFiles precisa ser false EXPLÍCITO
// porque o SetOptions do v3 liga de volta os dois desligados.
func PickDirectory(title, defaultDirectory string) string {
	app := application.Get()
	if app == nil {
		return ""
	}
	selection, err := app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                title,
		Directory:            defaultDirectory,
		CanChooseFiles:       false,
		CanChooseDirectories: true,
	}).PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}

// PickFile abre o seletor nativo de ARQUIVO com o título, a pasta inicial e
// os filtros pedidos (pares DisplayName/Pattern, como no v2). Devolve "" fora
// do ciclo de vida do app e no cancelamento.
func PickFile(title, defaultDirectory string, filters ...application.FileFilter) string {
	app := application.Get()
	if app == nil {
		return ""
	}
	selection, err := app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:     title,
		Directory: defaultDirectory,
		Filters:   filters,
	}).PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}

// PickSave abre o diálogo "Salvar como". filename é o nome sugerido; devolve
// "" fora do ciclo de vida do app e no cancelamento.
func PickSave(title, defaultDirectory, filename string, filters ...application.FileFilter) string {
	app := application.Get()
	if app == nil {
		return ""
	}
	selection, err := app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:     title,
		Directory: defaultDirectory,
		Filename:  filename,
		Filters:   filters,
	}).PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}
