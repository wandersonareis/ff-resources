import type { GameVersionId } from '@/lib/ffx/game-version';
import { part1 } from './part1';
import { part2 } from './part2';
import { part3 } from './part3';
import type { PartUi } from './part-ui';

export type { PartUi } from './part-ui';

const PARTS: Record<GameVersionId, PartUi> = {
  part1,
  part2,
  part3,
};

/** Particularidades de UI do jogo; ids desconhecidos caem no part1. */
export function partUi(version: GameVersionId): PartUi {
  return PARTS[version] ?? part1;
}
