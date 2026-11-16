// Adaptador FF13 do serviço de ATUALIZAÇÃO: delega para os bindings gerados
// do UpdateService (mesmo contrato do FFX).
import { Application } from '@wailsio/runtime';
import {
  AppVersion,
  CheckNow,
  CurrentVersion,
  Restart,
} from '@ff13-bindings/ffresources/apps/ff13/backend/service/updateservice';

export { AppVersion, CheckNow, CurrentVersion, Restart };

/** Encerra o app (usado pelo fluxo de sair). */
export function QuitApp(): Promise<void> {
  return Application.Quit();
}
