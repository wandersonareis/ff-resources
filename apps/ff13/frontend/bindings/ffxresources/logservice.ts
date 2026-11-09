// Adaptador FF13 do serviço de LOG da UI.
//
// O FF13 tem a mesma ponte de log (internal/platform/uilog): o histórico
// vive no backend e a janela mostra ao vivo pelos eventos. Aqui o histórico
// começa vazio e o log é escrito pelo WriteLog do adapter de API.
import type { Entry } from './backend/uilog/models';

/** Histórico do log da sessão (vazio até o backend expor a consulta). */
export function History(): Promise<Entry[]> {
  return Promise.resolve([]);
}

/** Limpa o histórico do log. */
export function ClearLog(): Promise<void> {
  return Promise.resolve();
}
