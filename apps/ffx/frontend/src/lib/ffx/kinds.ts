import type { EntryKind } from '@/lib/ffx/display-names';
import type { GameVersionId } from '@/lib/ffx/game-version';

/** Kinds de TEXTO servidos por versão no FFX/X-2 (régua do app). */
export function entryKindsFor(version: GameVersionId): EntryKind[] {
  if (version === 'eternalcalm') return ['events'];
  if (version === 'lastmiss') return ['events', 'objects'];
  if (version === 'ffx')
    return ['events', 'objects', 'macro', 'lockit', 'help', 'battletext', 'cloud', 'menumain'];
  return ['events', 'objects', 'macro', 'lockit', 'battletext', 'cloud', 'tutorial'];
}

/** Kinds de IMAGEM (.dds.phyre): só ffx e ffx2. */
export function imageKindsFor(version: GameVersionId): EntryKind[] {
  if (version === 'lastmiss' || version === 'eternalcalm') return [];
  return ['images'];
}

/** Todos os kinds da árvore da versão (texto + imagem), em ordem canônica. */
export function allKindsFor(version: GameVersionId): EntryKind[] {
  return [...entryKindsFor(version), ...imageKindsFor(version)];
}
