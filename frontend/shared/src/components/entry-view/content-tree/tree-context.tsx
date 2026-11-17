'use client';

import { createContext, type ReactNode, useContext } from 'react';
import type { SideNode } from '../../../app/content-tree/model';
import type { EntryKind } from '../../../app/display-names';

/**
 * Estado e ações compartilhados por TODA a árvore. Existe só para a
 * recursão do TreeItem não arrastar 8 props nível a nível — o dono dos
 * dados continua sendo o store da view (index.tsx apenas o projeta aqui).
 *
 * O dono MEMOIZA o valor (useMemo com deps estáveis) e o TreeItem é memo:
 * com contexto estável, nó intocado re-renderiza só quando o próprio nó muda.
 */
export interface ContentTreeValue {
  expanded: ReadonlySet<string>;
  selectedId: string | null;
  selectedByKind: ReadonlyMap<EntryKind, ReadonlySet<string>>;
  vbfSelectionByRoot: ReadonlyMap<string, readonly string[]>;
  /** Rótulos i18n por `node.type` (folder/file/system/mission/event/...). */
  typeLabels?: Readonly<Record<string, string>>;
  /** Ícones por `node.type`; o frontend decide (i18n/design). */
  typeIcons?: Readonly<Record<string, ReactNode>>;
  /**
   * Render prop do chamador: recebe o nó, a profundidade e o conteúdo padrão
   * da linha; devolve o que quiser (pode envolver, trocar ou só devolver o
   * padrão). O chamador controla o visual sem o pacote conhecer regra de jogo.
   */
  renderItem?: (node: SideNode, depth: number, defaultContent: ReactNode) => ReactNode;
  onToggle: (node: SideNode) => void;
  onSelect: (node: SideNode) => void;
  /**
   * Marca/desmarca o nó. Pode ser assíncrono: um grupo ainda não carregado
   * (diretório, container, índice) carrega os filhos antes de marcar.
   */
  onCheck: (node: SideNode, checked: boolean) => void | Promise<void>;
}

const ContentTreeContext = createContext<ContentTreeValue | null>(null);

export function ContentTreeProvider({
  value,
  children,
}: {
  value: ContentTreeValue;
  children: ReactNode;
}) {
  return <ContentTreeContext.Provider value={value}>{children}</ContentTreeContext.Provider>;
}

export function useContentTree(): ContentTreeValue {
  const value = useContext(ContentTreeContext);
  if (!value) {
    throw new Error('useContentTree precisa estar dentro de ContentTreeProvider');
  }
  return value;
}
