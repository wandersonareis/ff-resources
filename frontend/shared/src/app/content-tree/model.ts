import type * as dto from '../../dto';
import type { EntryKind } from '../display-names';
import type { EntryRow } from '../tree-data';

/**
 * Modelo PURO da árvore da sidebar (raiz de kind, grupo de eventos ou folha).
 *
 * Fora do componente de propósito: nada aqui toca store, DOM ou notificação —
 * só transforma nós. Quem monta os nós continua no store da view.
 */

/** Nó da árvore da sidebar (raiz de kind, grupo de eventos ou folha). */
export interface SideNode {
  id: string;
  label: string;
  kind?: EntryKind;
  /** Categoria de conteúdo (folder/file/system/mission/event...) — i18n no frontend. */
  type?: string;
  entry?: EntryRow;
  children?: SideNode[];
  /** Raiz de kind em carga: spinner no lugar do chevron, filhos a caminho. */
  loading?: boolean;

  // ---- Navegador do container .vbf (fonte somente leitura) ----
  /** true = nó que pertence ao container .vbf. */
  vbf?: boolean;
  /** Caminho absoluto do container — só nas raízes. */
  vbfRoot?: string;
  /** Caminho interno dentro do container ("" = raiz do .vbf). */
  vbfPath?: string;
  /** É macrodic.dcp: os filhos saem de loadVbfMacroChunks (chunk_XX). */
  macro?: boolean;
  /**
   * Pode ser expandido mesmo sem filhos carregados (diretório do .vbf ainda
   * não buscado). Vira false quando a listagem devolve vazio.
   */
  expandable?: boolean;
  /** Arquivo que o app não serve (áudio, vídeo, .exe…): clique avisa. */
  unsupported?: boolean;
  /** Metadado curto exibido como badge (idioma/plataforma, por exemplo). */
  badge?: string;
  /** Descrição longa da linha, usada como tooltip. */
  description?: string;

  // ---- Navegador de filelist (família White) ----
  /** true = nó que pertence a um filelist. */
  filelist?: boolean;
  /** Caminho absoluto do filelist que indexa este nó — só nas raízes. */
  filelistRoot?: string;
  /** Caminho interno indexado neste filelist ("" = raiz). */
  filelistPath?: string;
  /**
   * O próprio arquivo é um filelist: expansível como árvore, mas sobre OUTRO
   * arquivo (o white bin externo que ele indexa). Não é lido nesta fase.
   */
  nestedFilelist?: boolean;
  /**
   * Nó de ZONA: os filhos vêm da união da zona (índice de zona + entradas
   * "zone/<z>/…" dos demais índices), não da listagem do índice.
   */
  filelistZone?: string;
  /** Tooltip da linha (metadado do nó: plataforma, pasta de origem…). */
  hint?: string;

  // ---- FF13: containers de imagem/data ----
  /** Referência opaca da imagem/container (backend). */
  imageRef?: string;
  /** Nó expansível de container de imagem. */
  imageContainer?: boolean;
  /** Nó de imagem (ícone e painel). */
  image?: boolean;
  /**
   * Operação destrutiva (delete) indisponível neste nó. É uma capacidade da
   * FONTE (o .vbf é estritamente somente-leitura); não existe para o filelist.
   */
  readOnly?: boolean;
}

/** Uma folha de entrada do nó, com o kind que a identifica na seleção. */
export interface NodeEntryRef {
  kind: EntryKind;
  id: string;
}

/** IDS das folhas de entrada contidas no nó; grupos de imagens são aninhados. */
export function entryIdsInNode(node: SideNode): string[] {
  if (node.entry) return [node.entry.id];
  return (node.children ?? []).flatMap(entryIdsInNode);
}

/**
 * FOLHAS do nó com o kind de cada uma. Um grupo pode misturar kinds (um
 * diretório do filelist tem texto e imagens sob o mesmo nó), então a seleção
 * não pode ser indexada por um `kind` único do nó.
 */
export function nodeEntries(node: SideNode): NodeEntryRef[] {
  if (node.entry) return [{ kind: node.entry.kind, id: node.entry.id }];
  return (node.children ?? []).flatMap(nodeEntries);
}

/** Estado tri-state do checkbox de um nó data/ sobre todas as folhas abaixo. */
export function entryCheckState(
  node: SideNode,
  selected: ReadonlySet<string>,
): boolean | 'indeterminate' {
  const ids = entryIdsInNode(node);
  const count = ids.filter((id) => selected.has(id)).length;
  return count === 0 ? false : count === ids.length ? true : 'indeterminate';
}

/**
 * Tri-state por KIND: cada folha é consultada no conjunto do seu próprio
 * kind (o mapa é `version|kind` → ids projetado pelo app). É o estado usado
 * pela árvore, porque um grupo pode misturar texto e imagem.
 *
 * Grupo com subárvore AINDA NÃO CARREGADA nunca fica "cheio": o que está
 * marcado é só o que já foi listado. Isso mantém o tri-state honesto (o
 * usuário vê que falta conteúdo) sem exigir indexar o índice inteiro.
 */
export function entryCheckStateByKind(
  node: SideNode,
  selectedByKind: ReadonlyMap<EntryKind, ReadonlySet<string>>,
): boolean | 'indeterminate' {
  const entries = nodeEntries(node);
  if (entries.length === 0) return false;
  const count = entries.filter((entry) => selectedByKind.get(entry.kind)?.has(entry.id)).length;
  if (count === 0) return false;
  if (count < entries.length) return 'indeterminate';
  return hasUnloadedGroupChildren(node) ? 'indeterminate' : true;
}

/**
 * true quando o nó ainda tem conteúdo de grupo por listar. Nó com `entry`
 * próprio é FOLHA de seleção (ex.: .bin que também é container): o checkbox
 * marca a unidade, não os registros internos.
 */
function hasUnloadedGroupChildren(node: SideNode): boolean {
  if (node.entry) return false;
  if (node.expandable && (node.children?.length ?? 0) === 0) return true;
  return (node.children ?? []).some(hasUnloadedGroupChildren);
}

/**
 * Próximo estado do checkbox ao clicar no nó.
 *
 * O Radix pede `true` SEMPRE que o estado visual é `indeterminate` (ver
 * `requestedChecked` no pacote react-checkbox). Num grupo com subárvore ainda
 * não listada — que `entryCheckStateByKind` mantém `indeterminate` de
 * propósito — isso prende o checkbox: cada clique re-marca e o usuário não
 * consegue desmarcar. A regra aqui é a INTENÇÃO do clique: quando tudo o que
 * está listado abaixo do nó já está marcado, o próximo clique remove; senão
 * ele completa o que está listado.
 */
export function nextCheckState(
  node: SideNode,
  selectedByKind: ReadonlyMap<EntryKind, ReadonlySet<string>>,
): boolean {
  const entries = nodeEntries(node);
  if (entries.length === 0) return true;
  const allSelected = entries.every((entry) => selectedByKind.get(entry.kind)?.has(entry.id));
  return !allSelected;
}

/**
 * Containers de imagem (WPD/TRB) abaixo do nó cujos filhos ainda NÃO foram
 * listados. O checkbox de grupo usa a lista para carregar tudo num lote só
 * antes de marcar — sem isso, marcar uma pasta deixaria de fora todo registro
 * dentro de um container fechado (o container só existe como nó expansível,
 * sem `entry`).
 *
 * Nó que TEM `entry` e também é container (WPD de texto do filelist) fica de
 * fora: a seleção dele é a própria unidade, não os registros internos.
 */
export function unloadedImageContainers(node: SideNode): SideNode[] {
  const out: SideNode[] = [];
  const walk = (current: SideNode): void => {
    if (
      !current.entry &&
      current.imageContainer &&
      current.imageRef &&
      (current.children?.length ?? 0) === 0
    ) {
      out.push(current);
      // Filhos de container só existem depois de carregados: nada a descer.
      return;
    }
    current.children?.forEach(walk);
  };
  walk(node);
  return out;
}

/**
 * Id estável de uma folha da árvore. Folhas de .vbf são identificadas pelo
 * CAMINHO no container (o mesmo id de data/ pode existir nos dois lados e
 * em dois containers diferentes).
 */
export function entryNodeId(entry: EntryRow): string {
  return entry.vbf
    ? `vbfleaf:${entry.vbf.root}|${entry.vbf.path}`
    : `leaf:${entry.kind}:${entry.id}`;
}

export const EMPTY_IDS: ReadonlySet<string> = new Set<string>();

/** ID visual do grupo de kinds textuais de data/. */
export const TEXT_ROOT_NODE_ID = 'kind:text';

/**
 * true para a raiz de UM kind de data/ (`kind:events`, `kind:battletext`…).
 * `Texto` (kind:text) também casa no prefixo, mas NÃO tem `kind` — o
 * agrupamento é só visual e nunca é enviado ao backend como EntryKind.
 * Use com `node.kind` para decidir o que é selecionável/exportável.
 */
export function isKindRootNode(node: SideNode): boolean {
  return node.id.startsWith('kind:');
}

/**
 * Agrupa as raizes reais para exibicao. `Texto` e apenas um no visual sem
 * `kind`, portanto nao e enviado ao backend como EntryKind. Imagens e todos
 * os kinds textuais preservam seus nos/kinds originais.
 */
export function buildContentRoots(kindRoots: readonly SideNode[]): SideNode[] {
  const textRoots: SideNode[] = [];
  const imageRoots: SideNode[] = [];
  for (const root of kindRoots) {
    if (root.kind === 'images') imageRoots.push(root);
    else textRoots.push(root);
  }

  return [
    ...(textRoots.length > 0
      ? [{ id: TEXT_ROOT_NODE_ID, label: 'Texto', children: textRoots }]
      : []),
    ...imageRoots,
  ];
}

/**
 * Índice id → nó de TODOS os nós visíveis (containers .vbf e data/). O
 * callback dos hotkeys recebe só o evento, então o nó é resolvido pelo DOM
 * ([data-node-id]) e devolvido por este mapa — as teclas nunca carregam
 * referências de nó na closure.
 */
export function buildNodeIndex(
  roots: readonly SideNode[],
  vbfRoots: readonly SideNode[],
  filelistRoots?: readonly SideNode[],
): Map<string, SideNode> {
  const map = new Map<string, SideNode>();
  const walk = (node: SideNode) => {
    map.set(node.id, node);
    node.children?.forEach(walk);
  };
  // Ordem dos containers primeiro: só afeta a ordem de iteração do Map, que a
  // árvore não usa (consulta é sempre por id).
  vbfRoots.forEach(walk);
  roots.forEach(walk);
  filelistRoots?.forEach(walk);
  return map;
}

/**
 * Reaproveita por ID as subárvores JÁ CARREGADAS ao remontar uma lista de
 * raízes. A listagem do filelist/.vbf é preguiçosa: o reload reconstrói as
 * raízes sem filhos, e sem esta fusão tudo o que estava aberto "fecha" (os
 * ids continuam no `expanded`, mas os filhos somem).
 *
 * `next` manda no que existe: nó que sumiu da listagem não volta; nó novo
 * entra fechado; nó que continua e tem filhos carregados em `prev` mantém a
 * subárvore (recursivamente). Filhos estáticos de `next` (ex.: o grupo de
 * zonas) são mantidos e completados a partir de `prev`.
 */
export function mergeLoadedChildren(
  next: readonly SideNode[],
  prev: readonly SideNode[],
): SideNode[] {
  if (prev.length === 0) return [...next];
  const loaded = new Map<string, readonly SideNode[]>();
  const walk = (nodes: readonly SideNode[]): void => {
    for (const node of nodes) {
      if (node.children && node.children.length > 0) {
        loaded.set(node.id, node.children);
        walk(node.children);
      }
    }
  };
  walk(prev);

  const attach = (nodes: readonly SideNode[]): SideNode[] =>
    nodes.map((node) => {
      const base = node.children && node.children.length > 0 ? node.children : (loaded.get(node.id) ?? []);
      if (base.length === 0) return node;
      return { ...node, children: attach(base) };
    });

  return attach(next);
}

/**
 * Caminho da RAIZ até a folha (inclusive), de cima para baixo — é o que o
 * reveal da busca expande no `expanded` sem tocar no resto do que o usuário
 * abriu. [] = a folha não existe nas raízes dadas (árvore ainda em carga).
 */
export function ancestorPathOf(roots: readonly SideNode[], leafId: string): string[] {
  const walk = (nodes: readonly SideNode[]): string[] | null => {
    for (const node of nodes) {
      if (node.id === leafId) return [node.id];
      const hit = node.children ? walk(node.children) : null;
      if (hit) return [node.id, ...hit];
    }
    return null;
  };
  return walk(roots) ?? [];
}

/**
 * Nó realçado para a seleção. Em imagens a árvore só guarda o
 * REPRESENTANTE do grupo de cópias — quando a seleção é uma cópia oculta
 * (a lista "Repetidas" navega até ela), o realce salta para o representante
 * em vez de sumir da árvore.
 */
export function highlightedNodeId(
  selectedEntry: EntryRow | null,
  image: dto.ImageEntry | null,
): string | null {
  if (!selectedEntry) return null;
  // Folha de .vbf é identificada pelo caminho no container: o mesmo id de
  // data/ pode existir dos dois lados (e em dois containers).
  if (selectedEntry.vbf) return entryNodeId(selectedEntry);
  const base = `leaf:${selectedEntry.kind}:${selectedEntry.id}`;
  if (selectedEntry.kind !== 'images' || !image) return base;
  if (image.metadata.id !== selectedEntry.id) return base;
  const group = [selectedEntry.id, ...(image.duplicates ?? []).map((d) => d.id)].sort();
  return `leaf:images:${group[0]}`;
}