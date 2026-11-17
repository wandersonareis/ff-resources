export const GAME_VERSIONS = [
  { id: 'ffx', label: 'FFX' },
  { id: 'eternalcalm', label: 'FFX Eternal Calm' },
  { id: 'ffx2', label: 'FFX-2' },
  { id: 'lastmiss', label: 'FFX-2 Last Mission' },
] as const;

export type GameVersionId = (typeof GAME_VERSIONS)[number]['id'];

/**
 * Versões com container .vbf PRÓPRIO na sidebar: eternalcalm e lastmiss
 * dividem a árvore do jogo-pai SEM container (vbfVersionOf do backend só
 * mapeia FFX_Data.vbf/FFX2_Data.vbf) — nessas abas a seção .vbf vazia é
 * por design e o aviso "aponte o executável" não pode aparecer.
 */
export function hasOwnVbfContainer(version: GameVersionId): boolean {
  return version === 'ffx' || version === 'ffx2';
}

export function isGameVersionId(value: unknown): value is GameVersionId {
  return (
    typeof value === 'string' &&
    (GAME_VERSIONS as readonly { id: string }[]).some((v) => v.id === value)
  );
}

/**
 * Árvore (bucket de arquivos) de uma versão: eternalcalm divide a árvore do
 * FFX e lastmiss a do FFX-2. É a chave dos avisos de carga do backend
 * (EventsLoadProgress/EventsLoaded), que valem para as duas abas.
 */
export function treeOfVersion(version: string): 'ffx' | 'ffx2' {
  return version === 'ffx2' || version === 'lastmiss' ? 'ffx2' : 'ffx';
}
