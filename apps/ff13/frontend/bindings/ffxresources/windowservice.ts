// Adaptador FF13 do serviço de JANELAS.
//
// O FF13 tem a janela principal e a janela de log ("w", servida em
// /w/#logs). Não há backend de janelas bound: o runtime do Wails faz o
// trabalho — Window.Get(nome) devolve o handle da janela pelo nome.
import { Window } from '@wailsio/runtime';

/** Nomes de janela do app (o FF13 não tem a janela "config" do FFX). */
const WINDOWS: Record<string, string> = {
  logs: 'w',
};

/** Abre (ou foca) uma janela filha pelo nome lógico ("logs"). */
export function OpenWindow(name: string): Promise<void> {
  const id = WINDOWS[name];
  if (!id) return Promise.resolve();
  return Window.Get(id).Show();
}

/** Fecha a janela filha pelo nome lógico ("logs"). */
export function CloseWindow(name: string): Promise<void> {
  const id = WINDOWS[name];
  if (!id) return Promise.resolve();
  return Window.Get(id).Hide();
}
