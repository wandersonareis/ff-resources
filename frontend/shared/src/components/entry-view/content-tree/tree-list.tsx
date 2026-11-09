'use client';

import type { MouseEvent, ReactNode, RefObject } from 'react';
import { ScrollArea } from '../../ui/scroll-area';
import type { SideNode } from '../../../app/content-tree/model';
import { TreeItem } from './tree-item';

/** Grupo de nós da árvore (ex.: containers .vbf, arquivos de data/). */
export interface TreeGroup {
  id: string;
  label?: string;
  nodes: readonly SideNode[];
  /** Conteúdo exibido no lugar dos nós quando o grupo está vazio. */
  emptyContent?: ReactNode;
}

/**
 * Renderizador da árvore: recebe os nós já montados pelo app, em grupos.
 * Não conhece jogo, kind nem store — o app decide o agrupamento (FFX usa
 * eventos; FF13 usa diretórios) e este componente só desenha.
 */
export function TreeList({
  treeRef,
  groups,
  onNodeContextMenu,
}: {
  treeRef: RefObject<HTMLDivElement | null>;
  groups: readonly TreeGroup[];
  onNodeContextMenu: (event: MouseEvent) => void;
}) {
  return (
    <ScrollArea
      id="entry-tree"
      ref={treeRef}
      className="flex-1 min-h-0"
      onContextMenuCapture={onNodeContextMenu}
    >
      {groups.map((group, index) => (
        <div key={group.id} data-tree-group={group.id}>
          {group.label && group.nodes.length > 0 ? (
            <small
              className={`px-2 pb-1 text-xs font-medium text-muted-foreground ${
                index === 0 ? 'pt-1' : 'pt-3 mt-1'
              }`}
            >
              {group.label}
            </small>
          ) : null}
          {group.nodes.length === 0 && group.emptyContent ? group.emptyContent : null}
          {group.nodes.map((node) => (
            <TreeItem key={node.id} node={node} depth={0} />
          ))}
        </div>
      ))}
    </ScrollArea>
  );
}
