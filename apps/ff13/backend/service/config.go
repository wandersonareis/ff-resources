package service

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"ffresources/apps/ff13/backend/config"
)

// PREFERÊNCIA DE MODS (a flag enableMods) e PART ATIVA.
//
// enableMods: ligada = a tabela mostra a TRADUÇÃO (mods/); desligada = mostra
// o pristine de data/. A preferência é PERSISTIDA no ff13.config.json e vale
// para as janelas todas — o eco AppConfigChanged distribui o valor (cada
// janela tem a sua instância do store).
//
// O EXPORTE não muda com a flag: ele sempre usa source (pristine) e target
// (tradução).
//
// A part ativa é estado de EXECUÇÃO (não config): a aba principal manda a part
// que está na tela e o ConfigSnapshot a devolve. É assim que a janela de
// config (outro React root, outra instância do store) mostra a instalação da
// part certa sem acoplamento entre janelas.

// SetActivePart registra a part ativa (a aba mandou) e re-emite o snapshot:
// as janelas precisam do valor (o da config mostra a instalação da part).
func (FF13Service) SetActivePart(part string) {
	config.SetActivePart(part)
	emitConfigChanged()
}

// SetEnableMods persiste a preferência e re-emite o snapshot.
//
// A coleção de dados NÃO é invalidada aqui: o conteúdo cacheado guarda o
// pristine E a tradução (a flag só decide qual dos dois a tabela mostra), e
// essa escolha acontece na leitura. Invalidar custaria uma releitura de
// disco sem mudar nada.
func (FF13Service) SetEnableMods(enabled bool) error {
	if err := config.SetEnableMods(enabled); err != nil {
		return err
	}
	emitConfigChanged()
	return nil
}

// ResolveEnableModsChange publica a DECISÃO da guarda de edições não salvas.
//
// O backend não aplica nada aqui: ele publica o pedido (enabled + decision)
// e a janela PRINCIPAL — dona dos rascunhos — executa salvar/descartar e só
// então chama SetEnableMods. "save" que falha deixa o rascunho sujo e o
// toggle não aplica.
func (FF13Service) ResolveEnableModsChange(enabled bool, decision string) {
	emitEvent("EnableModsChange", map[string]any{
		"enabled":  enabled,
		"decision": decision,
	})
}

// emitConfigChanged re-emite o snapshot do config: é o "eco" que toda janela
// aplica (o valor persistido, já normalizado).
func emitConfigChanged() {
	emitEvent("AppConfigChanged", FF13Service{}.ConfigSnapshot())
}

// emitEvent publica um evento do app (nil-safe: fora do app — teste, CLI —
// não emite nada).
func emitEvent(name string, payload any) {
	a := application.Get()
	if a == nil {
		return
	}
	a.Event.Emit(name, payload)
}

// notifyWarn publica um AVISO no frontend (evento "Notify") — o mesmo
// contrato do notificador do FFX (message + severity): o ProgressNotifier
// do frontend transforma em toast.
func notifyWarn(message string) {
	emitEvent("Notify", map[string]any{
		"message":  message,
		"severity": "warn",
	})
}
