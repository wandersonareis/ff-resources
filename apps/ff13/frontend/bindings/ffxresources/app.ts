// Adaptador FF13 do serviço de SESSÃO (equivalente ao App do FFX).
//
// O FF13 ainda não tem um serviço de sessão bound no Go: as ações de ciclo
// de vida ficam aqui, em volta do runtime do Wails. A flag de edições não
// salvas é mantida localmente (a aba ativa a sincroniza) — a janela de
// configuração lê a MESMA instância do módulo, então a guarda do toggle de
// mods continua funcionando.
import { Application } from '@wailsio/runtime';
import * as ff13 from '@ff13-bindings/ffresources/apps/ff13/ff13service';

/** Flag de rascunho pendente (espelhada entre janelas do mesmo módulo). */
let unsavedEdits = false;

/** Encerra o app (usado no fluxo de salvar + sair). */
export function QuitApp(): Promise<void> {
  return Application.Quit();
}

/** Há edições não salvas? (guarda do toggle de mods na janela de config). */
export function HasUnsavedEdits(): Promise<boolean> {
  return Promise.resolve(unsavedEdits);
}

/** Marca/desmarca as edições pendentes (chamado pela aba a cada mudança). */
export function SetUnsavedEdits(flag: boolean): Promise<void> {
  unsavedEdits = Boolean(flag);
  return Promise.resolve();
}

/**
 * Aquece as parts indicadas em background (o backend monta a coleção de cada
 * part em goroutine, sem travar a UI; idempotente).
 */
export function PreloadVersions(versions: string[]): Promise<void> {
  return ff13.PreloadVersions(versions) as Promise<void>;
}
