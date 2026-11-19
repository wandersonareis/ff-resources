'use client';

import { type MouseEvent, useCallback, useMemo, useRef } from 'react';
import type { TreeCtxNode } from '@/lib/ffx/content-tree/menu-target';
import {
  buildNodeIndex,
  isKindRootNode,
  nodeEntries,
  type SideNode,
  unloadedImageContainers,
} from '@/lib/ffx/content-tree/model';
import { parseError } from '@/lib/ffx/error-handler';
import { exportLangs } from '@/lib/ffx/export-langs';
import { exportSelection } from '@/lib/ffx/export-selection';
import { progressWasShown } from '@/lib/ffx/progress-store';
import { loggedToast as toast } from '@/lib/ffx/toast-logged';
import type { EntryRow } from '@/lib/ffx/tree-data';
import { exportOutput } from '@/lib/ffx/export-output';
import {
  exportArtifactBatch,
  extractImageFiles,
  extractImageSelection,
} from '@/lib/ffx/tree-data';
import {
  exportVbfSelection,
  extractVbfImagesSelection,
  invalidateVbfImage,
  selectVbfImageExtractDir,
} from '@/lib/ffx/vbf';
import { vbfSelection } from '@/lib/ffx/vbf-selection';
import type { EntryView } from '../entry-view-store';
import type { VbfExtractRequest } from './vbf-extract-dialog';

/**
 * Operações da árvore: marcar (checkbox), exportar, extrair e apagar.
 *
 * Saiu do componente porque nada disso é UI — tudo escreve em store, chama o
 * backend ou monta toast. Lê o estado de que precisa por parâmetro, então o
 * ContentTree continua sendo o único dono das assinaturas.
 */
export function useContentTreeActions({
  view,
  ctxNode,
  selectedEntry,
  selectedImageIds,
  setCtxNode,
  openExtractRequest,
}: {
  view: EntryView;
  ctxNode: TreeCtxNode | null;
  selectedEntry: EntryRow | null;
  selectedImageIds: string[];
  setCtxNode: (node: TreeCtxNode | null) => void;
  openExtractRequest: (request: VbfExtractRequest) => void;
}) {
  const { version, actions } = view;
  const extractRequestCounter = useRef(0);

  // O menu do .vbf opera somente sobre os caminhos marcados no container
  // clicado — o nó sob o cursor apenas determina qual container recebe o
  // menu, não troca a seleção por esse nó.
  const vbfPathsForContext = useMemo(
    () =>
      ctxNode?.vbfRoot ? (ctxNode.selectedVbfPaths ?? vbfSelection.pathsOf(ctxNode.vbfRoot)) : [],
    [ctxNode],
  );

  /**
   * Checkbox da árvore. Folha = as entradas do próprio nó; grupo = todas as
   * folhas abaixo (o kind de cada uma, porque um diretório do filelist pode
   * misturar texto e imagem). A raiz de kind de data/ marca o kind INTEIRO
   * (`whole`), que deduplica a exportação para o artefato bulk único.
   *
   * Grupo ainda NÃO CARREGADO (diretório, container de imagem/texto, índice
   * de filelist aninhado) busca as folhas antes de marcar — só os formatos
   * compatíveis entram (folhas sem `entry` ficam de fora). Containers de
   * imagem fechados vão num LOTE ÚNICO antes de marcar, senão marcar uma
   * pasta deixaria de fora tudo o que está dentro dos WPD/TRB.
   *
   * Checkbox do .vbf: guarda o caminho (diretório = subárvore, expandida
   * pelo backend); inclui imagens e formatos fora do escopo de texto para
   * permitir a extração dos binários originais.
   */
  const checkNode = useCallback(
    async (node: SideNode, checked: boolean): Promise<void> => {
      if (node.vbf) {
        const root = node.vbfRoot ?? node.entry?.vbf?.root;
        const path = node.vbfPath ?? node.entry?.vbf?.path ?? '';
        if (root) vbfSelection.set(root, path, checked);
        return;
      }
      const whole = isKindRootNode(node) && checked;
      const nodeIndex = () =>
        buildNodeIndex(
          view.store.state.roots,
          view.store.state.vbfRoots,
          view.store.state.filelistRoots,
        );
      if (checked) {
        const containers = unloadedImageContainers(node);
        if (containers.length > 0) await actions.loadImageContainers(containers);
        // Grupo não-imagem sem folhas listadas (índice de filelist, diretório
        // do .vbf): busca o próprio nó antes de marcar.
        const current = nodeIndex().get(node.id) ?? node;
        if (!current.imageContainer && current.expandable && nodeEntries(current).length === 0) {
          await actions.loadNodeChildren(current);
        }
      }
      const entries = nodeEntries(nodeIndex().get(node.id) ?? node);
      if (entries.length === 0) return;
      exportSelection.setManyByKind(version, entries, checked, whole);
    },
    [actions, version, view],
  );

  const onExtractSelectedImages = useCallback(async () => {
    if (selectedImageIds.length === 0) {
      toast.message('Nenhuma imagem marcada para extrair.');
      return;
    }
    const opts = exportOutput.getSnapshot();
    if (!opts.decode && !opts.extract) {
      toast.info('Marque Descompactar ou Extrair para escolher a saída.');
      return;
    }
    try {
      if (opts.decode) {
        const result = await extractImageSelection(selectedImageIds, version);
        if (result.failed.length > 0) {
          toast.warning(`Extraídas ${result.done.length} de ${result.total} imagens.`, {
            description: result.failed.slice(0, 3).join('; '),
          });
        } else {
          toast.success(`Extraídas ${result.done.length} imagens selecionadas.`);
        }
      }
      if (opts.extract) {
        const result = await extractImageFiles(version, selectedImageIds);
        if (result.failed.length > 0) {
          toast.warning(`Extraído ${result.done.length} binário(s) do filelist.`, {
            description: result.failed.slice(0, 3).join('; '),
          });
        } else if (result.done.length > 0) {
          toast.success(`Extraído ${result.done.length} binário(s) para data/.`);
        }
      }
      await actions.reload();
    } catch (error) {
      toast.error(parseError(error));
    }
  }, [actions, selectedImageIds, version]);

  const onDeleteSelectedImages = useCallback(() => {
    if (selectedImageIds.length === 0) return;
    actions.openImageAction({
      type: 'delete-selection',
      id: selectedImageIds[0],
      ids: selectedImageIds,
      label: `${selectedImageIds.length} imagens selecionadas`,
      duplicates: null,
    });
  }, [actions, selectedImageIds]);

  // Exporta só o que está marcado na árvore de data/ no FORMATO SELECIONADO
  // no topo (JSON | Strings | CSV | XLSX) — leitura imperativa do store
  // compartilhado. Uma ação = modal de progresso com o resumo no final.
  //
  // Seleção marcada pela RAIZ do kind (`complete`) deduplica para o
  // artefato bulk único (prefixo tree_ no backend): um arquivo em vez de N.
  const idsForContext = useCallback((): string[] => {
    const kind = ctxNode?.kind;
    if (!kind) return [];
    const marked = exportSelection.idsOf(version, kind);
    if (marked.length > 0) return marked;
    // Sem checkbox: o nó clicado é o alvo (um item).
    if (ctxNode?.entryId) return [ctxNode.entryId];
    if (kind === 'images') {
      const container = ctxNode?.id?.startsWith('imagecontainer:')
        ? ctxNode.id.slice('imagecontainer:'.length)
        : '';
      if (container) return [container];
    }
    return [];
  }, [ctxNode, version]);

  const onCtxExport = useCallback(async () => {
    const kind = ctxNode?.kind;
    if (!kind) return;
    const opts = exportOutput.getSnapshot();
    if (!opts.decode && !opts.extract) {
      toast.info('Marque Descompactar ou Extrair para escolher a saída.');
      return;
    }
    const ids = idsForContext();
    if (ids.length === 0) {
      toast.message(`Nenhum arquivo marcado para exportar (${kind}).`);
      return;
    }
    const format = exportSelection.formatOf();
    const complete = exportSelection.completeOf(version, kind);
    // A folha do filelist exporta a fonte validada; a de data/ usa a coleção
    // de data/. O backend cai em data/ se o jogo não estiver configurado.
    const source = ctxNode?.filelistRoot ? 'filelist' : 'data';
    try {
      if (kind === 'images') {
        if (opts.decode) {
          const result = await extractImageSelection(ids, version);
          if (result.total === 0) toast.info('A seleção não contém imagens compatíveis.');
        }
        if (opts.extract) {
          const result = await extractImageFiles(version, ids);
          if (result.failed.length > 0) {
            toast.warning(`Extraído ${result.done.length} binário(s).`, {
              description: result.failed.slice(0, 3).join('; '),
            });
          }
        }
        return;
      }
      const paths = await exportArtifactBatch(
        [{ kind, ids, complete, source }],
        version,
        exportLangs.langsOf(),
        format,
        opts,
      );
      if ((paths ?? []).length === 0) {
        toast.info('Nada para exportar.');
      }
      // Modal de progresso mostrou o resumo — sem toast de sucesso.
    } catch (error) {
      if (!progressWasShown()) toast.error(parseError(error));
    }
  }, [ctxNode, idsForContext, version]);

  /**
   * "Extrair binário em data/" do menu de contexto: só filelist. Usa a
   * seleção/entry do nó e a mesma régua de kinds do export.
   */
  const onCtxExtract = useCallback(async () => {
    const kind = ctxNode?.kind;
    if (!kind) return;
    if (!ctxNode?.filelistRoot) {
      toast.info('Extrair só vale para itens do filelist.');
      return;
    }
    const ids = idsForContext();
    if (ids.length === 0) return;
    try {
      if (kind === 'images') {
        const result = await extractImageFiles(version, ids);
        if (result.failed.length > 0) {
          toast.warning(`Extraído ${result.done.length} binário(s).`, {
            description: result.failed.slice(0, 3).join('; '),
          });
        } else if (result.done.length > 0) {
          toast.success(`Extraído ${result.done.length} binário(s) para data/.`);
        }
        return;
      }
      const paths = await exportArtifactBatch(
        [
          {
            kind,
            ids,
            complete: exportSelection.completeOf(version, kind),
            source: 'filelist',
          },
        ],
        version,
        [],
        exportSelection.formatOf(),
        { decode: false, extract: true },
      );
      if ((paths ?? []).length === 0) toast.info('Nada para extrair.');
    } catch (error) {
      if (!progressWasShown()) toast.error(parseError(error));
    }
  }, [ctxNode, idsForContext, version]);

  const onVbfExtract = useCallback(
    (mode: 'data' | 'choose') => {
      if (!ctxNode?.vbfRoot || vbfPathsForContext.length === 0) return;
      openExtractRequest({
        requestId: ++extractRequestCounter.current,
        root: ctxNode.vbfRoot,
        paths: [...vbfPathsForContext],
        mode,
      });
      setCtxNode(null);
    },
    [ctxNode, openExtractRequest, setCtxNode, vbfPathsForContext],
  );

  const onVbfExtractImages = useCallback(
    (mode: 'default' | 'choose') => {
      if (!ctxNode?.vbfRoot || vbfPathsForContext.length === 0) return;
      const root = ctxNode.vbfRoot;
      const paths = [...vbfPathsForContext];
      setCtxNode(null);
      void (async () => {
        let destRoot = '';
        if (mode === 'choose') {
          destRoot = await selectVbfImageExtractDir();
          if (!destRoot) return;
        }
        try {
          const result = await extractVbfImagesSelection(root, paths, destRoot);
          if (result.total === 0) {
            toast.info('A seleção não contém imagens compatíveis para extrair.');
          } else if (result.failed.length > 0) {
            toast.warning(`Extraídas ${result.done.length} de ${result.total} imagens.`, {
              description: result.failed.slice(0, 3).join('; '),
            });
          } else {
            toast.success(`Extraídas ${result.done.length} imagens do .vbf.`);
          }
          if (selectedEntry?.vbf && result.done?.includes(selectedEntry.vbf.path)) {
            invalidateVbfImage(selectedEntry.vbf.root, selectedEntry.vbf.path);
            await actions.selectEntry(selectedEntry);
          }
        } catch (error) {
          toast.error(parseError(error));
        }
      })();
    },
    [actions, ctxNode, selectedEntry, setCtxNode, vbfPathsForContext],
  );

  const onVbfExport = useCallback(() => {
    if (!ctxNode?.vbfRoot || vbfPathsForContext.length === 0) return;
    const root = ctxNode.vbfRoot;
    const paths = [...vbfPathsForContext];
    setCtxNode(null);
    void (async () => {
      const format = exportSelection.formatOf();
      try {
        const exported = await exportVbfSelection(root, format, paths, exportLangs.langsOf());
        if (exported.length === 0) {
          toast.info('A seleção não contém textos compatíveis para extrair.');
          return;
        }
        // O backend agrupa por tipo com um modal de progresso por ação —
        // o resumo (arquivos + destino) aparece lá, sem toast paralelo.
      } catch (error) {
        if (!progressWasShown()) toast.error(parseError(error));
      }
    })();
  }, [ctxNode, setCtxNode, vbfPathsForContext]);

  /**
   * Clique direito na árvore → estado do menu. O atributo `data-node-id`
   * decide o alvo; no .vbf a seleção congelada no clique tem prioridade
   * sobre a marcação atual, e sem marcação o menu opera só no arquivo
   * clicado (uma pasta precisa ser marcada para expandir sua subárvore).
   */
  const onNodeContextMenu = useCallback(
    (event: MouseEvent) => {
      const row = (event.target as HTMLElement).closest('[data-node-id]');
      if (!(row instanceof HTMLElement)) {
        setCtxNode(null);
        return;
      }
      const kind = row.getAttribute('data-node-kind') as TreeCtxNode['kind'] | null;
      const isVbf = row.getAttribute('data-node-vbf') === 'true';
      const id = row.getAttribute('data-node-id') ?? '';
      const label = row.getAttribute('data-node-label') ?? '';
      const readOnly = row.getAttribute('data-node-readonly') === 'true';
      if (isVbf) {
        const vbfRoot = row.getAttribute('data-node-vbf-root') ?? '';
        const selectedVbfPaths = vbfSelection.pathsOf(vbfRoot);
        const clickedPath = row.getAttribute('data-node-vbf-path') ?? '';
        const isFile = row.getAttribute('data-node-vbf-file') === 'true';
        const paths =
          vbfSelection.countOf(vbfRoot) > 0
            ? selectedVbfPaths
            : isFile && clickedPath
              ? [clickedPath]
              : [];
        setCtxNode(
          vbfRoot && paths.length > 0
            ? { id, label, vbfRoot, vbfPath: clickedPath, selectedVbfPaths: paths }
            : null,
        );
        return;
      }
      const entryId = row.getAttribute('data-node-entry-id') ?? undefined;
      const filelistRoot = row.getAttribute('data-node-filelist-root') ?? undefined;
      const filelistPath = row.getAttribute('data-node-filelist-path') ?? undefined;
      setCtxNode(
        kind
          ? { id, kind, label, readOnly, entryId, filelistRoot, filelistPath }
          : null,
      );
    },
    [setCtxNode],
  );

  return {
    checkNode,
    onCtxExport,
    onCtxExtract,
    onExtractSelectedImages,
    onDeleteSelectedImages,
    onVbfExtract,
    onVbfExtractImages,
    onVbfExport,
    onNodeContextMenu,
  };
}
