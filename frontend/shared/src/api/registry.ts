import type { GameApi } from './GameApi';

/**
 * Registro do adapter ativo.
 *
 * O <ApiProvider> serve os componentes React; este registro serve os módulos
 * fora do React (stores, loaders de árvore/tabela, rascunhos), que precisam
 * chamar o backend sem prop drilling. O app registra o adapter uma vez no
 * boot (main.tsx).
 */
let current: GameApi | null = null;

export function setCurrentApi(api: GameApi): void {
  current = api;
}

export function getCurrentApi(): GameApi {
  if (!current) {
    throw new Error('getCurrentApi: nenhum adapter registrado (chame setCurrentApi no boot)');
  }
  return current;
}

export function hasCurrentApi(): boolean {
  return current !== null;
}
