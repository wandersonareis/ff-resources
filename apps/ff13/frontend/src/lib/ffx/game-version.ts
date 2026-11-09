export const GAME_VERSIONS = [
  { id: 'part1', label: 'FFXIII' },
  { id: 'part2', label: 'FFXIII-2' },
  { id: 'part3', label: 'LRFFXIII' },
] as const;

export type GameVersionId = (typeof GAME_VERSIONS)[number]['id'];

/**
 * O FF13 não tem container .vbf; o filelist é lido pelo importador e as
 * árvores editáveis vivem em data/partN + mods/partN.
 */
export function hasOwnVbfContainer(_version: GameVersionId): boolean {
  return false;
}

export function isGameVersionId(value: unknown): value is GameVersionId {
  return (
    typeof value === 'string' &&
    (GAME_VERSIONS as readonly { id: string }[]).some((v) => v.id === value)
  );
}
