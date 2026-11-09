import { createContext, type ReactNode, useContext } from 'react';
import type { GameApi } from './GameApi';

/**
 * Provider do adapter de API. Cada app monta o seu (bindings Wails) e o
 * injeta aqui; os componentes compartilhados só conhecem a interface.
 */
const ApiContext = createContext<GameApi | null>(null);

export function ApiProvider({ api, children }: { api: GameApi; children: ReactNode }) {
  return <ApiContext.Provider value={api}>{children}</ApiContext.Provider>;
}

/** Adapter da API do app ativo. Lança se o provider não estiver montado. */
export function useApi(): GameApi {
  const api = useContext(ApiContext);
  if (!api) {
    throw new Error('useApi: falta o <ApiProvider> na árvore da UI');
  }
  return api;
}

/** Adapter opcional: devolve null quando não há provider (uso em testes). */
export function useApiOrNull(): GameApi | null {
  return useContext(ApiContext);
}
