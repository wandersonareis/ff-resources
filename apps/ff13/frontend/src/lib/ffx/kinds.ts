import { type EntryKind, KIND_LABELS } from '@/lib/ffx/display-names';
import type { GameVersionId } from '@/lib/ffx/game-version';

/**
 * Kinds servidos no FF13.
 *
 * O FF13 tem texto em UM formato só (o `.ztr`) — dividir o escopo por tipo de
 * ARQUIVO não separaria nada. A régua é o DIRETÓRIO: o `txtres` é o pai de
 * todos os subdiretórios de texto, e a pasta logo abaixo dele é a família do
 * conteúdo (ac, resident, zone, event). O part3 (LR) acrescenta o `scenario`
 * (db/ai/npc/pack) e não tem `ac`.
 *
 * Estes são os ângulos do export/import (dropdown "Exportar"/"Importar" e
 * raízes da árvore): um kind que a part não tem NÃO entra na lista.
 */
const KINDS_BY_PART: Record<GameVersionId, EntryKind[]> = {
  part1: ['ac', 'resident', 'zone', 'event'],
  part2: ['ac', 'resident', 'zone', 'event'],
  part3: ['resident', 'zone', 'event', 'scenario'],
};

/** Ordem canônica das famílias de texto do FF13 (usada na lista completa). */
export const ALL_FF13_KINDS: EntryKind[] = [
  'ac',
  'resident',
  'zone',
  'event',
  'scenario',
];

export function entryKindsFor(version: GameVersionId): EntryKind[] {
  return [...(KINDS_BY_PART[version] ?? KINDS_BY_PART.part1)];
}

/**
 * Comparador da ordem de EXIBIÇÃO dos kinds: alfabética pelo rótulo PT-BR
 * ("Eventos", "Habilidades", "Residentes", "Zonas"...). A régua é o rótulo,
 * não a chave: o primeiro item do dropdown/modal é o que o usuário lê.
 */
export function compareKindLabels(a: EntryKind, b: EntryKind): number {
  return KIND_LABELS[a].localeCompare(KIND_LABELS[b], 'pt-BR');
}

/**
 * Kinds da part na ordem alfabética de exibição — usada nos menus
 * Exportar/Importar e no lote do export (ordem dos steps do modal).
 * `entryKindsFor` continua sendo a ordem canônica (raízes da árvore).
 */
export function entryKindsAlpha(version: GameVersionId): EntryKind[] {
  return entryKindsFor(version).sort(compareKindLabels);
}

export function imageKindsFor(_version: GameVersionId): EntryKind[] {
  // As imagens do FF13 moram nas pastas reais de data/partN e são misturadas
  // com o texto. A raiz "Imagens" é o único agrupamento visual usado para
  // elas; o filelist tem as suas próprias folhas.
  return ['images'];
}

export function allKindsFor(version: GameVersionId): EntryKind[] {
  return [...entryKindsFor(version), ...imageKindsFor(version)];
}
