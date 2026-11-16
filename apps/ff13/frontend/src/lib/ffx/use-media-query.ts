'use client';

import { useCallback, useSyncExternalStore } from 'react';

/**
 * Media query reativa para o layout responsivo do shell.
 *
 * `useSyncExternalStore` evita o setState dentro de efeito (e o frame com
 * valor antigo): o primeiro render no cliente já lê `matchMedia`, e mudanças
 * de largura re-renderizam só quando a query realmente cruza o limiar.
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onChange: () => void) => {
      const mql = window.matchMedia(query);
      mql.addEventListener('change', onChange);
      return () => mql.removeEventListener('change', onChange);
    },
    [query],
  );
  const getSnapshot = useCallback(() => window.matchMedia(query).matches, [query]);
  const getServerSnapshot = useCallback(() => false, []);
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

/** Janela estreita: sidebar vira gaveta e os painéis empilham. */
export const NARROW_QUERY = '(max-width: 767px)';
