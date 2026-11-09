import type { SideNode } from '@/lib/ffx/content-tree/model';
import { entryKindOfUnit } from '@/lib/ffx/text-kind';
import type { EntryRow } from '@/lib/ffx/tree-data';

/**
 * Árvore de UNIDADES de texto do FF13 dentro de UM kind (família do texto).
 *
 * A entrada não é um arquivo: é uma UNIDADE — o conjunto de idiomas que
 * compartilham o nome-base na mesma pasta (`txtres/resident/book/txtres`).
 *
 * O FF13 tem texto em UM formato só (o `.ztr`), então o que separa os tipos
 * é o DIRETÓRIO: o `txtres` é o pai de todos os subdiretórios de texto e a
 * pasta abaixo dele é a família (ac, resident, zone, event). O part3
 * acrescenta o `scenario` (db/ai/npc/pack) e não tem `ac`.
 *
 * IMPORTANTE: a raiz do kind (`kind:resident`, "Residentes (N)") é montada
 * pelo STORE (`buildKindNode`). Esta função devolve só os FILHOS dela — os
 * diretórios/folhas do kind. Sem isso a árvore ganhava duas raízes iguais
 * ("Residentes > Residentes > txtres > …").
 *
 * A montagem é híbrida e a omissão de diretórios é SÓ VISUAL:
 *
 *   - o prefixo estrutural do kind (`txtres/<kind>`, `db/ai/npc/pack` e o
 *     invólucro `zone/<z>/` do part1) sai do caminho de EXIBIÇÃO; o
 *     `entry.id` continua completo;
 *   - pasta sem unidade direta e com um único filho é colapsada (o nó
 *     visível é o diretório com conteúdo);
 *   - pasta com UMA unidade e sem subpasta → a pasta vira o NÓ clicável
 *     (o rótulo é o nome da pasta e o clique abre o texto);
 *   - pasta com VÁRIAS unidades (ou com subpastas) → a pasta é nó comum e
 *     cada unidade é uma folha (rótulo = nome-base, sem o sufixo de idioma);
 *   - cópias com o MESMO caminho visual viram UMA folha. O `data/` do part1
 *     mantém a cópia global (`txtres/...`) e a de zona
 *     (`zone/<z>/txtres/...`); as duas são idênticas no original, mas só
 *     uma pode ter tradução em `mods/`. A escolhida é a `modded` (e, no
 *     empate, a global/mais curta — a que o layout achatado vai manter).
 *
 * O id do nó de unidade é `leaf:<kind>:<id da unidade>` — o mesmo que o
 * shared usa em `entryNodeId`/`highlightedNodeId`; as pastas usam
 * `dir:<caminho visual>`.
 */

/** Pasta da árvore de unidades (diretório do id da unidade). */
interface DirNode {
  /** Caminho VISUAL da pasta, relativo à raiz ("" = raiz, sem barra final). */
  path: string;
  /** Nome da pasta (último segmento do caminho). */
  name: string;
  /** Unidades DIRETAS da pasta. */
  units: EntryRow[];
  /** Subpastas (por nome). */
  children: Map<string, DirNode>;
}

function newDir(path: string): DirNode {
  return {
    path,
    name: path.slice(path.lastIndexOf('/') + 1),
    units: [],
    children: new Map(),
  };
}

/**
 * Segmentos VISUAIS de uma unidade: o caminho sem o prefixo estrutural do
 * kind. O `entry.id` não muda — quem abre/exporta continua usando o caminho
 * completo.
 *
 *   txtres/event/ev_x/txtres            → ev_x/txtres
 *   txtres/resident/book/txtres         → book/txtres
 *   zone/z002/txtres/ac/ac_x/txtres     → ac_x/txtres
 *   zone/z002/txtres/zone/z002/txtres   → z002/txtres
 *   db/ai/npc/pack/auto_yus             → auto_yus
 *
 * Caminhos fora das pastas conhecidas (kind legado `ztr`) ficam inteiros.
 */
export function displaySegments(entry: EntryRow): string[] {
  const segments = entry.id.split('/').filter(Boolean);
  if (segments.length === 0) return segments;
  const kind = (entry.kind ?? entryKindOfUnit(entry.id)).toLowerCase();
  const lower = segments.map((s) => s.toLowerCase());

  // txtres/<kind> — cobre o global e o txtres de zona do part1.
  for (let i = 0; i + 1 < lower.length; i++) {
    if (lower[i] === 'txtres' && lower[i + 1] === kind) {
      return segments.slice(i + 2);
    }
  }

  // scenario: db/ai/npc/pack/<unidade>.
  if (kind === 'scenario') {
    const prefix = ['db', 'ai', 'npc', 'pack'];
    if (prefix.every((name, i) => lower[i] === name)) {
      return segments.slice(prefix.length);
    }
  }

  // zone sem o txtres/zone (fallback): o id da zona é o que interessa.
  if (kind === 'zone') {
    const i = lower.lastIndexOf('zone');
    if (i >= 0 && i + 1 < segments.length) {
      return segments.slice(i + 1);
    }
  }

  return segments;
}

/**
 * Monta os FILHOS da raiz de um kind a partir das entradas do backend.
 *
 * O kind já foi escolhido pelo store (`loadKindEntries(kind)`); esta função
 * não cria raiz nenhuma. Se receber entradas de kinds diferentes, cada uma
 * usa o próprio prefixo — mas o contrato é uma chamada por kind.
 */
export function buildUnitTree(entries: EntryRow[]): SideNode[] {
  return renderDir(collectDirs(entries));
}

/**
 * true quando `candidate` é a cópia que deve representar um caminho visual
 * repetido: a que já tem tradução em `mods/`; no empate, a global/mais curta
 * (a que o layout achatado vai manter).
 */
function preferEntry(candidate: EntryRow, current: EntryRow): boolean {
  const candidateModded = candidate.modded === true;
  const currentModded = current.modded === true;
  if (candidateModded !== currentModded) return candidateModded;
  if (candidate.id.length !== current.id.length) {
    return candidate.id.length < current.id.length;
  }
  return candidate.id < current.id;
}

/** Agrupa as unidades pelo caminho VISUAL (o id continua completo). */
function collectDirs(entries: EntryRow[]): DirNode {
  const root = newDir('');

  // Dedupe pelo caminho visual: cópias viram uma folha só. A escolhida é a
  // que tem tradução em mods/ (ou a global, quando nenhuma tem).
  const byDisplay = new Map<string, EntryRow>();
  for (const entry of entries) {
    const segments = displaySegments(entry);
    if (segments.length === 0) continue;
    const key = segments.join('/');
    const current = byDisplay.get(key);
    if (current && !preferEntry(entry, current)) continue;
    byDisplay.set(key, entry);
  }

  for (const entry of byDisplay.values()) {
    const segments = displaySegments(entry);
    let dir = root;
    for (let i = 0; i < segments.length - 1; i++) {
      const name = segments[i];
      let child = dir.children.get(name);
      if (!child) {
        child = newDir(dir.path ? `${dir.path}/${name}` : name);
        dir.children.set(name, child);
      }
      dir = child;
    }
    dir.units.push(entry);
  }
  return root;
}

/**
 * Colapsa a cadeia de diretórios sem unidade direta e com filho único. O nó
 * visível é o ÚLTIMO diretório da cadeia (o que tem conteúdo) e o caminho
 * visual completo continua no `path` do nó.
 */
function collapseChain(dir: DirNode): DirNode {
  let current = dir;
  while (current.units.length === 0 && current.children.size === 1) {
    current = current.children.values().next().value as DirNode;
  }
  return current;
}

/** Nó de uma pasta: as unidades diretas viram folhas e as subpastas, nós. */
function renderDir(dir: DirNode): SideNode[] {
  const nodes: SideNode[] = [];
  for (const unit of dir.units) {
    nodes.push(unitLeaf(unit));
  }
  for (const child of dir.children.values()) {
    const node = collapseChain(child);
    if (node.units.length === 1 && node.children.size === 0) {
      // Pasta com UMA unidade: a pasta É o nó clicável (rótulo = nome da
      // pasta, sem o nome do arquivo).
      const unit = node.units[0];
      nodes.push({
        id: unitNodeId(unit),
        label: node.name,
        kind: unit.kind,
        type: unit.type ?? 'text',
        entry: { ...unit, label: node.name },
      });
      continue;
    }
    nodes.push({
      id: `dir:${node.path}`,
      label: node.name,
      type: 'folder',
      children: renderDir(node),
    });
  }
  return nodes.sort((a, b) => a.label.localeCompare(b.label));
}

/** Folha de uma unidade: o rótulo é o nome-base (sem o sufixo de idioma). */
function unitLeaf(unit: EntryRow): SideNode {
  return {
    id: unitNodeId(unit),
    label: unit.label,
    kind: unit.kind,
    type: unit.type ?? 'text',
    entry: unit,
  };
}

/** Id do nó de uma unidade (o mesmo do `entryNodeId` do shared). */
function unitNodeId(unit: EntryRow): string {
  return `leaf:${unit.kind}:${unit.id}`;
}
