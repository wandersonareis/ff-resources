import type { SideNode } from '@/lib/ffx/content-tree/model';
import type { PartUi } from '@/lib/parts';
import { zoneRoot } from './filelist';
import {
  type FilelistNode,
  type FilelistRoot,
  filelistLabel,
  lowerName,
  mergeLabel,
  sourceLabel,
} from './filelist-common';
import {
  filelistEntryOf,
  filelistImageEntryOf,
  filelistLeafLabel,
  filelistOpaqueLabel,
  isFilelistImageContainer,
  isFilelistImageLeaf,
  isFilelistTextLeaf,
} from './filelist-entry';

/**
 * RENDERIZAÇÃO da árvore de filelist: DTO do backend → nós da sidebar.
 *
 * As funções são PURAS (sem store, sem bindings): quem chama monta os nós e
 * a store cuida da carga preguiçosa. Toda a formatação de texto acontece
 * aqui e no `filelist-common` (o backend não manda rótulo pronto).
 */

/** Nó de um índice (filelist) raiz: expansível, com a plataforma no tooltip. */
function indexNode(root: FilelistRoot): SideNode {
  return {
    id: `filelist:${root.path}|`,
    label: filelistLabel(root),
    type: 'folder',
    filelist: true,
    filelistRoot: root.path,
    filelistPath: '',
    expandable: true,
    hint: rootHint(root),
  };
}

/** Nó de uma ZONA (part1): o conteúdo vem da união da zona, não do índice. */
function zoneNode(zone: string, hint?: string): SideNode {
  return {
    id: `filelist:${zoneRoot(zone)}`,
    label: zone,
    type: 'folder',
    filelist: true,
    filelistRoot: zoneRoot(zone),
    filelistZone: zone,
    expandable: true,
    hint,
  };
}

/**
 * Tooltip da raiz: a PLATAFORMA do filelist (que vale para tudo abaixo
 * dele) e a pasta onde a cópia vencedora mora ("PC · white_data").
 */
function rootHint(root: FilelistRoot): string | undefined {
  const parts = [sourceLabel(root.source), root.sourceDir].filter(Boolean);
  return parts.length > 0 ? parts.join(' · ') : undefined;
}

/**
 * Raízes do navegador de filelist.
 *
 * - os índices GLOBAIS viram um nó cada (rótulo pelo `filelist-common`);
 * - os índices de ZONA (part1) viram o nó "Zone", com um filho por zona
 *   (z000, z002…); no part2/part3 eles continuam no grupo "zone", como antes;
 * - a ordem dos globais é da part (`filelistRootOrder`).
 */
export function buildFilelistRoots(roots: FilelistRoot[], part: PartUi): SideNode[] {
  const orderKey = part.filelistRootOrder ?? ((name: string) => name.toLowerCase());
  const globals: { node: SideNode; key: string }[] = [];
  const zoneIndexes: { node: SideNode; key: string }[] = [];
  const zones = new Map<string, SideNode>();
  let zoneHint: string | undefined;

  for (const root of roots) {
    if (root.group === 'zone') {
      if (part.zoneAsTopLevel) {
        const zone = root.zone ?? '';
        if (!zone || zones.has(zone)) continue;
        zoneHint = zoneHint ?? rootHint(root);
        zones.set(zone, zoneNode(zone, zoneHint));
        continue;
      }
      zoneIndexes.push({ node: indexNode(root), key: orderKey(root.name) });
      continue;
    }
    globals.push({ node: indexNode(root), key: orderKey(root.name) });
  }

  const byKey = (a: { key: string }, b: { key: string }) => a.key.localeCompare(b.key);
  globals.sort(byKey);
  zoneIndexes.sort(byKey);

  const nodes: SideNode[] = globals.map((g) => g.node);
  const zoneChildren = part.zoneAsTopLevel
    ? [...zones.values()].sort((a, b) => a.label.localeCompare(b.label))
    : zoneIndexes.map((z) => z.node);
  if (zoneChildren.length > 0) {
    nodes.push({
      id: 'filelist:zone|',
      label: part.filelistZoneLabel,
      type: 'folder',
      filelist: true,
      expandable: true,
      hint: zoneHint,
      children: zoneChildren,
    });
  }
  return nodes;
}

/**
 * Filhos de um nó do filelist (um nível do índice).
 *
 * - diretório → pasta expansível (o rótulo é a cadeia fundida, quando o
 *   backend fundiu diretórios com filho único);
 * - arquivo de TEXTO (.ztr solto ou .bin container WPD) → folha clicável que
 *   abre a unidade de texto (o rótulo é o nome do arquivo, com a extensão);
 * - arquivo fora do escopo → folha "fora do escopo" (nada é decodificado);
 * - índice embutido (kind "index") → folha expansível, com o rótulo de
 *   filelist (nome sem extensão + idioma).
 *
 * A folha de texto (que entry ela abre, com que idioma e como se chama) é do
 * `filelist-entry` — aqui só mora a MONTAGEM da árvore.
 *
 * root é a raizvirtual do índice (o caminho do índice, o do índice embutido
 * "pai#virtual" ou "zone:z000" da zona).
 */
export function buildFilelistChildren(
  root: string,
  nodes: FilelistNode[],
  part: PartUi,
  version: string,
): SideNode[] {
  const out: SideNode[] = [];
  for (const node of nodes) {
    // No part1 o diretório "zone" dentro de um índice some: o conteúdo da
    // zona vive só no nó "Zone".
    if (part.zoneAsTopLevel && node.isDir && lowerName(node.name) === 'zone') continue;
    const id = node.imageContainer && node.imageRef
      ? `imagecontainer:${node.imageRef}`
      : `filelist:${root}|${node.path}`;
    if (node.isDir) {
      out.push({
        id,
        label: mergeLabel(node),
        type: 'folder',
        filelist: true,
        filelistRoot: root,
        filelistPath: node.path,
        expandable: true,
      });
      continue;
    }
    const nested = node.kind === 'index';
    const imageContainer = isFilelistImageContainer(node);
    const imageLeaf = isFilelistImageLeaf(node);
    const entry =
      nested || imageContainer
        ? undefined
        : isFilelistTextLeaf(node)
          ? filelistEntryOf(node, version)
          : imageLeaf
            ? filelistImageEntryOf(node)
            : undefined;
    out.push({
      id,
      label: nested
        ? filelistLabel(node)
        : isFilelistTextLeaf(node) || imageLeaf
          ? filelistLeafLabel(node)
          : filelistOpaqueLabel(node),
      type: imageContainer ? 'folder' : 'file',
      kind: imageContainer || imageLeaf ? 'images' : undefined,
      filelist: true,
      filelistRoot: root,
      filelistPath: node.path,
      entry,
      unsupported: !nested && !entry && !imageContainer,
      nestedFilelist: nested,
      expandable: nested || imageContainer,
      imageContainer,
      imageRef: node.imageRef,
      readOnly: imageContainer || imageLeaf || undefined,
    });
  }
  return out;
}
