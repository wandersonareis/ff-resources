package main

import (
	"ffresources/internal/platform/appwindows"
)

// WindowService expõe as janelas FILHAS ao frontend do FF13.
//
// O catálogo (config, logs) vive no appwindows — título, geometria e URL
// ("/w/#<nome>") num lugar só; o frontend só pede pelo nome. A janela
// principal é a "main" do appwindows (main.go): as filhas nascem ligadas a
// ela e morrem com ela.
//
// O About NÃO existe no FF13: o appwindows tem a ficha, mas o frontend do
// FF13 não monta a janela — nenhum botão a pede.
type WindowService struct{}

// OpenWindow abre (ou foca, se já aberta) a janela filha nomeada.
func (w *WindowService) OpenWindow(name string) error {
	return appwindows.Open(name)
}

// CloseWindow fecha a janela filha nomeada (inexistente não é erro: quem
// chama está sincronizando estado de UI, não gerenciando ciclo de vida).
func (w *WindowService) CloseWindow(name string) error {
	return appwindows.Close(name)
}
