import type { EntryKind } from '@/lib/ffx/display-names';

/**
 * KIND de uma unidade de texto do FF13, pelo CAMINHO.
 *
 * O FF13 tem texto em um formato só (o `.ztr`), então não há como dividir o
 * escopo por tipo de arquivo: a régua é o DIRETÓRIO. O `txtres` é o pai de
 * todos os subdiretórios de texto e a pasta logo abaixo dele é a família do
 * conteúdo. O `scenario` é o `db/ai/npc/pack` do part3 — o único kind cujo
 * texto mora num container `.bin` (WPD) em vez de num `.ztr` solto.
 *
 *   txtres/ac/…          → ac
 *   txtres/resident/…    → resident
 *   txtres/zone/…        → zone
 *   txtres/event/…       → event
 *   db/ai/npc/pack/…     → scenario
 *   zone/z004/txtres/ac/…→ ac   (o txtres de zona do part1)
 *
 * É o MESMO mapeamento do backend (`parts.TextKind`): o frontend precisa
 * dele porque a folha do filelist abre a unidade ANTES de qualquer listagem
 * do `/data` — e o id do nó (`leaf:<kind>:<id>`) tem de casar com o que a
 * árvore `/data` vai montar.
 */

/** Pastas abaixo de `txtres` que são famílias de texto. */
const TXTRES_KINDS: Record<string, string> = {
  ac: 'ac',
  resident: 'resident',
  zone: 'zone',
  event: 'event',
};

/** Prefixos de caminho do kind `scenario` (só o part3/LR os tem). */
const SCENARIO_PREFIXES = ['db/ai/npc/pack'];

/** Kind derivado do caminho; `undefined` quando não casa com nenhuma pasta. */
export function textKindOfPath(unitId: string): EntryKind | undefined {
  const p = unitId.replace(/\\/g, '/').toLowerCase();
  for (const prefix of SCENARIO_PREFIXES) {
    if (p === prefix || p.startsWith(prefix + '/')) return 'scenario';
  }
  const segs = p.split('/');
  for (let i = 0; i < segs.length - 1; i++) {
    if (segs[i] !== 'txtres') continue;
    const family = TXTRES_KINDS[segs[i + 1]];
    if (family) return family as EntryKind;
  }
  return undefined;
}

/** Kind efetivo de uma unidade (cai no fallback `ztr` quando não casa). */
export function entryKindOfUnit(unitId: string): EntryKind {
  return textKindOfPath(unitId) ?? 'ztr';
}
