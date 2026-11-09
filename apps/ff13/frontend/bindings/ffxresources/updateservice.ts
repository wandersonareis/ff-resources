// Adaptador FF13 do serviço de ATUALIZAÇÃO.
//
// O FF13 ainda não tem updater bound no Go: as funções resolvem sem efeito
// (a faixa de atualização do FFX fica inerte até existir o serviço).
import { Application } from '@wailsio/runtime';

/** Versão atual do app ("" = desconhecida). */
export function CurrentVersion(): Promise<string> {
  return Promise.resolve('');
}

/** Procura atualização (sem serviço: no-op). */
export function CheckNow(): Promise<void> {
  return Promise.resolve();
}

/** Reinicia o app para aplicar a atualização (sem serviço: no-op). */
export function Restart(): Promise<void> {
  return Promise.resolve();
}

/** Encerra o app (usado pelo fluxo de sair). */
export function QuitApp(): Promise<void> {
  return Application.Quit();
}
