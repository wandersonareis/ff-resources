import type { EntryKind } from '@/lib/ffx/display-names';

/**
 * Modelo PURO do menu de contexto da árvore: o que o DOM entrega quando o
 * usuário clica com o botão direito e o que o menu entende.
 */

export interface TreeCtxNode {
  id: string;
  kind?: EntryKind;
  label: string;
  vbfRoot?: string;
  vbfPath?: string;
  selectedVbfPaths?: string[];
  /** Nó com operação destrutiva indisponível (.vbf é somente-leitura). */
  readOnly?: boolean;
}

/** O nó sob o cursor: folha tem id (arquivo); grupo/raiz, não. */
export interface EntryMenuTarget {
  kind?: EntryKind;
  id?: string | null;
  label: string;
  /** Presente para nó do navegador .vbf. */
  vbfRoot?: string;
  vbfPath?: string;
  /** Operação destrutiva indisponível (.vbf). */
  readOnly?: boolean;
}

/** Nó da árvore → alvo do menu (folha carrega o id, grupo não). */
export function menuTargetOf(node: TreeCtxNode | null): EntryMenuTarget | null {
  if (!node) return null;
  if (node.vbfRoot) {
    return {
      label: node.label,
      vbfRoot: node.vbfRoot,
      vbfPath: node.vbfPath ?? '',
      readOnly: true,
    };
  }
  if (node.id.startsWith('imagecontainer:')) {
    return {
      kind: 'images',
      id: node.id.slice('imagecontainer:'.length),
      label: node.label,
      readOnly: node.readOnly,
    };
  }
  // Folhas do filelist não derivam o id do nó (o id é `filelist:<root>|<path>`);
  // o alvo de imagem sai do id `imagecontainer:<ref>`.
  const prefix = node.kind ? `leaf:${node.kind}:` : null;
  const isLeaf = prefix ? node.id.startsWith(prefix) : false;
  const id = isLeaf ? node.id.slice(prefix!.length) : null;
  return {
    kind: node.kind,
    id,
    // O rótulo exibido é o do NÓ (nome de arquivo/unidade). Nunca derive de
    // `id`: em imagens o id é a referência opaca em base64.
    label: node.label,
    readOnly: node.readOnly,
  };
}
