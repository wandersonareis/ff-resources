'use client';

import { useState } from 'react';
import type * as dto from '@ff-resources/ui/dto';
import type { EntryView } from '@/components/entry-view/entry-view-store';
import { parseError } from '@/lib/ffx/error-handler';
import { exportOutput } from '@/lib/ffx/export-output';
import { invalidateFilelistCache } from '@/lib/ffx/filelist';
import { loggedToast as toast } from '@/lib/ffx/toast-logged';
import type { EntryRow } from '@/lib/ffx/tree-data';
import {
  extractImageFiles,
  imageExportPath,
  importImage,
  importImageGroup,
  importImages,
  previewImportImages,
  refreshImageDuplicates,
  restoreFilelistInserts,
  saveImage,
  saveImageTo,
  selectImageExportDir,
  selectImageFile,
} from '@/lib/ffx/tree-data';
import {
  extractVbfImagesSelection,
  importVbfImage,
  invalidateVbfImage,
  saveVbfImage,
  selectVbfImageExtractDir,
  vbfImageExportPath,
} from '@/lib/ffx/vbf';
import { buildImageImportSelection, nextAllSelected } from './import-selection';

export type ImageBusy =
  | 'extract'
  | 'extractTo'
  | 'export'
  | 'import'
  | 'importBatch'
  | 'refresh'
  | null;

/**
 * Fluxos imperativos do painel: salvar, importar (com passo de alcance
 * quando há cópias), reanalisar duplicatas e abrir ação de imagem.
 *
 * Todo o "chama binding → grava → toast → recarrega" vive aqui; o JSX fica
 * só com o que pintar. `busy` trava os botões durante a gravação e
 * `pendingImport` é o .dds já escolhido aguardando a escolha de alcance.
 */
export function useImageActions({
  view,
  entry,
  duplicates,
}: {
  view: EntryView;
  /** null = nenhuma textura selecionada (o painel não renderiza nada). */
  entry: EntryRow | null;
  duplicates: dto.ImageDuplicate[];
}) {
  const { version, actions } = view;
  const [busy, setBusy] = useState<ImageBusy>(null);
  /**
   * .dds já escolhido aguardando o usuário decidir o alcance do import
   * (null = sem diálogo aberto). Só aparece quando a imagem tem cópias.
   */
  const [pendingImport, setPendingImport] = useState<string | null>(null);

  /**
   * Import de <TranslateLocation>/images: `pendingPreview` guarda o resumo
   * da varredura (null = diálogo fechado). A seleção (imagens e cópias
   * marcadas) vive dentro do preview — quem aplica é o confirm, com o
   * payload montado por buildImageImportSelection.
   */
  const [pendingPreview, setPendingPreview] = useState<dto.ImageImportPreview | null>(null);

  /**
   * Destinos do import (Comprimir/Inserir). Compartilhado entre o menu do
   * import avulso e o modal do lote: "Comprimir" nasce ligado (é a conversão
   * para mods/, o comportamento de sempre) e "Inserir" desligado (modifica a
   * instalação do jogo).
   */
  const [importOptions, setImportOptions] = useState<dto.ImageImportOptions>({
    write_mods: true,
    insert_filelist: false,
  });

  const destinationLabel = (opts: dto.ImageImportOptions): string => {
    const dests: string[] = [];
    if (opts.write_mods) dests.push('mods/');
    if (opts.insert_filelist) dests.push('filelist');
    return dests.length === 0 ? 'nenhum destino' : dests.join(' e ');
  };

  const afterImportReload = async (opts: dto.ImageImportOptions) => {
    // O filelist mudou em disco: o cache do frontend também precisa cair.
    if (opts.insert_filelist) invalidateFilelistCache();
    await actions.reload();
  };

  /**
   * Textura aberta pelo navegador de .vbf: o container é somente leitura;
   * extrair usa o caminho interno e importar/salvar grava fora dele. Delete e
   * replicate de grupos continuam restritos à árvore data/.
   */
  const readOnly = Boolean(entry?.vbf) || entry?.source === 'filelist';

  const refuseVbf = (): boolean => {
    if (!readOnly) return false;
    toast.message(
      'Exclusão e replicação estão disponíveis apenas para imagens da árvore de data/.',
      { duration: 4000 },
    );
    return true;
  };

  /**
   * Ações de imagem (extrair / replicar / deletar) abrem o diálogo único da
   * aba (ImageActionDialogs) com a textura aberta como alvo — ali ficam a
   * escolha de alcance, o aviso de irreversível e a contagem de cópias.
   */
  const openAction = (type: 'extract' | 'replicate' | 'delete') => {
    // O hook é chamado antes do painel decidir que não há textura; sem alvo
    // não há ação a abrir.
    if (!entry) return;
    if (readOnly) {
      if (type === 'extract' && entry.vbf) {
        setBusy('extract');
        void extractVbfImagesSelection(entry.vbf.root, [entry.vbf.path], '')
          .then((result) => {
            if (result.failed.length > 0) {
              toast.warning(`Extraídas ${result.done.length} de ${result.total} imagens.`, {
                description: result.failed.slice(0, 3).join('; '),
              });
            } else if (result.total > 0) {
              toast.success('Imagem extraída como .dds e .png.');
            }
            if (result.done.length > 0 && entry.vbf) {
              invalidateVbfImage(entry.vbf.root, entry.vbf.path);
              void actions.selectEntry(entry);
            }
          })
          .catch((error) => toast.error(parseError(error)))
          .finally(() => setBusy(null));
        return;
      }
      if (type === 'replicate' && entry.vbf) {
        actions.openImageAction({
          type: 'replicate',
          id: entry.id,
          label: entry.label,
          duplicates,
          vbf: entry.vbf,
        });
        return;
      }
      refuseVbf();
      return;
    }
    actions.openImageAction({
      type,
      id: entry.id,
      label: entry.label,
      duplicates,
    });
  };

  const onExport = async (format: 'dds' | 'png' | 'dds+png') => {
    if (!entry) return;
    const opts = exportOutput.getSnapshot();
    if (!opts.decode && !opts.extract) {
      toast.info('Marque Descompactar ou Extrair para escolher a saída.');
      return;
    }
    setBusy('export');
    try {
      const formats = format === 'dds+png' ? (['dds', 'png'] as const) : [format];
      const dests: string[] = [];
      let extracted = 0;
      let failed: string[] = [];

      if (opts.decode) {
        for (const fmt of formats) {
          let dest: string;
          if (entry.vbf) {
            dest = await vbfImageExportPath(entry.vbf.root, entry.vbf.path, fmt);
            await saveVbfImage(entry.vbf.root, entry.vbf.path, fmt, dest);
          } else {
            dest = await imageExportPath(entry.id, fmt, version);
            await saveImage(entry.id, fmt, dest, version);
          }
          dests.push(dest);
        }
      }

      if (opts.extract) {
        if (entry.source !== 'filelist') {
          toast.warning('Extrair só vale para imagens do filelist.');
        } else {
          const result = await extractImageFiles(version, [entry.id]);
          extracted = result.done.length;
          failed = result.failed;
        }
      }

      if (opts.decode && opts.extract) {
        toast.success(
          `Imagem descompactada (${dests.length}) e binário extraído (${extracted}).` +
            (failed.length > 0 ? ` ${failed.length} falha(s).` : ''),
          { description: failed.slice(0, 3).join('; ') || undefined },
        );
      } else if (opts.decode) {
        toast.success(
          dests.length === 1
            ? `Imagem exportada: ${dests[0]}`
            : `Imagem exportada como DDS e PNG: ${dests.join(', ')}`,
        );
      } else {
        if (failed.length > 0) {
          toast.warning(`Extraído ${extracted} binário(s).`, {
            description: failed.slice(0, 3).join('; '),
          });
        } else {
          toast.success(`Binário extraído para data/: ${extracted} arquivo(s).`);
        }
      }
      if (extracted > 0) {
        await actions.reload();
        await actions.selectEntry(entry);
      }
    } catch (error) {
      toast.error(parseError(error));
    } finally {
      setBusy(null);
    }
  };

  const onExtractTo = async (format: 'dds' | 'png' | 'dds+png') => {
    if (!entry) return;
    const opts = exportOutput.getSnapshot();
    if (!opts.decode && !opts.extract) {
      toast.info('Marque Descompactar ou Extrair para escolher a saída.');
      return;
    }
    setBusy('extractTo');
    try {
      // "Exportar para…" escolhe a pasta dos FORMATOS (DDS/PNG). O binário
      // do filelist sempre vai para data/partN (como os binários de data).
      const destRoot = opts.decode
        ? entry.vbf
          ? await selectVbfImageExtractDir()
          : await selectImageExportDir(version)
        : '';
      if (opts.decode && !destRoot) return; // usuário cancelou
      const fmtLabel = format === 'dds+png' ? '.dds e .png' : `.${format}`;
      if (entry.vbf) {
        // O extract do .vbf grava DDS e PNG juntos; o formato escolhido só
        // rotula o resumo. Extrair binário do filelist não se aplica.
        const result = await extractVbfImagesSelection(entry.vbf.root, [entry.vbf.path], destRoot);
        if (result.failed.length > 0) {
          toast.warning(`Extraídas ${result.done.length} de ${result.total} imagens.`, {
            description: result.failed.slice(0, 3).join('; '),
          });
        } else if (result.total > 0) {
          toast.success(`Imagem extraída como ${fmtLabel} em ${destRoot}`);
        }
        return;
      }
      if (opts.decode) {
        const formats = format === 'dds+png' ? (['dds', 'png'] as const) : [format];
        const dests: string[] = [];
        for (const fmt of formats) {
          dests.push(await saveImageTo(entry.id, fmt, destRoot, version));
        }
        toast.success(`Imagem exportada como ${fmtLabel}: ${dests.join(', ')}`);
      }
      if (opts.extract) {
        if (entry.source !== 'filelist') {
          toast.warning('Extrair só vale para imagens do filelist.');
        } else {
          const result = await extractImageFiles(version, [entry.id]);
          if (result.failed.length > 0) {
            toast.warning(`Extraído ${result.done.length} binário(s).`, {
              description: result.failed.slice(0, 3).join('; '),
            });
          } else if (result.done.length > 0) {
            toast.success(`Binário extraído: ${result.done[0]}`);
          }
          await actions.reload();
          await actions.selectEntry(entry);
        }
      }
    } catch (error) {
      toast.error(parseError(error));
    } finally {
      setBusy(null);
    }
  };

  /**
   * Importa o .dds escolhido. `all=true` propaga para as cópias idênticas
   * (grupo de payload original — o backend recusa alvo fora dele).
   */
  const runImport = async (path: string, all: boolean) => {
    if (!entry) return;
    setBusy('import');
    try {
      if (entry.vbf) {
        await importVbfImage(entry.vbf.root, entry.vbf.path, path);
        invalidateVbfImage(entry.vbf.root, entry.vbf.path);
        toast.success(`Textura importada em mods/: ${entry.label}`);
        await actions.selectEntry(entry);
        return;
      }
      if (!all) {
        await importImage(entry.id, path, version, importOptions);
        toast.success(
          `Textura importada (${destinationLabel(importOptions)}): ${path}`,
        );
      } else {
        const targets = duplicates.map((d) => d.id);
        const result = await importImageGroup(entry.id, path, targets, version, importOptions);
        const warnings = result.warnings ?? [];
        if (result.failed.length > 0) {
          toast.warning(`Importadas ${result.updated.length} de ${result.total} texturas.`, {
            description: result.failed.slice(0, 3).join('; '),
          });
        } else if (warnings.length > 0) {
          toast.warning(
            `Importadas ${result.updated.length} de ${result.total} texturas (${warnings.length} aviso(s)).`,
            { description: warnings.slice(0, 3).join('; ') },
          );
        } else {
          toast.success(
            `Textura importada em ${result.updated.length} textura(s) idêntica(s) (${destinationLabel(importOptions)}).`,
          );
        }
      }
      // Recarrega árvore + textura: o payload em mods/ e/ou filelist mudou.
      await afterImportReload(importOptions);
      await actions.selectEntry(entry);
    } catch (error) {
      toast.error(parseError(error));
    } finally {
      // O diálogo fica aberto durante a gravação (botões travados) e fecha
      // no fim — sucesso ou erro.
      setPendingImport(null);
      setBusy(null);
    }
  };

  const onImport = async () => {
    if (!entry) return;
    if (!importOptions.write_mods && !importOptions.insert_filelist) {
      toast.info('Marque Comprimir ou Inserir para escolher o destino do import.');
      return;
    }
    try {
      const path = await selectImageFile(entry.id, version);
      if (!path) return;
      if (readOnly) {
        await runImport(path, false);
        return;
      }
      // Com cópias idênticas o usuário escolhe o alcance antes de gravar.
      if (duplicates.length > 0) {
        setPendingImport(path);
        return;
      }
      await runImport(path, false);
    } catch (error) {
      toast.error(parseError(error));
    }
  };

  /** Reconstrói o índice de duplicatas (edição externa no hex editor). */
  const onRefresh = async () => {
    if (refuseVbf()) return;
    setBusy('refresh');
    try {
      await refreshImageDuplicates(version);
      toast.success('Duplicatas reanalisadas.');
      // Árvore junto: o índice é quem decide quais cópias ficam ocultas.
      await actions.reload();
    } catch (error) {
      toast.error(parseError(error));
    } finally {
      setBusy(null);
    }
  };

  /**
   * Import de mods/translated/images: abre varrendo a raiz (sem seleção
   * prévia) e mostra o sumário. Cada marcação mexe SÓ no estado do preview
   * (imagens e cópias) — nada regrava o binário aqui.
   */
  const openImportImages = async () => {
    if (!importOptions.write_mods && !importOptions.insert_filelist) {
      toast.info('Marque Comprimir ou Inserir para escolher o destino do import.');
      return;
    }
    try {
      setPendingPreview(await previewImportImages(version));
    } catch (error) {
      toast.error(parseError(error));
    }
  };

  /**
   * Aplica a seleção do modal: monta o payload (imagens + cópias marcadas,
   * com a regra da imagem igual ao binário sendo reimportada junto com as
   * cópias) e chama o backend. Só o que está marcado é gravado.
   */
  const confirmImportImages = async () => {
    if (!pendingPreview) return;
    const options = importOptions;
    const selection = buildImageImportSelection(pendingPreview, options);
    setBusy('importBatch');
    const toastId = 'importImages';
    toast.loading('Importando imagens…', { id: toastId });
    try {
      const result = await importImages(version, selection);
      setPendingPreview(null);
      const dest = destinationLabel(options);
      if (result.updated.length === 0) {
        const motivo = result.failed.length > 0 ? ` (${result.failed.length} falharam)` : '';
        toast.info(`Nada foi gravado (${dest}).${motivo}`, { id: toastId });
        return;
      }
      const warnings = result.warnings ?? [];
      const falhas = result.failed.length > 0 ? ` (${result.failed.length} falharam)` : '';
      const avisos = warnings.length > 0 ? ` (${warnings.length} aviso(s))` : '';
      const details = [...result.failed, ...warnings].slice(0, 3).join('; ');
      toast.success(
        `${result.updated.length} de ${result.total} imagem(ns) processada(s) em ${dest}.${falhas}${avisos}`,
        {
          id: toastId,
          ...(details ? { description: details } : {}),
        },
      );
      await afterImportReload(options);
      if (entry) await actions.selectEntry(entry);
    } catch (error) {
      toast.error(parseError(error), { id: toastId });
    } finally {
      setBusy(null);
    }
  };

  /**
   * Desfaz TODAS as inserções da part: o backend restaura os filelists do
   * backup e trunca os white bins ao tamanho original. Funciona mesmo depois
   * de fechar e reabrir o app (o manifesto fica em mods/partN).
   */
  const onRestoreInserts = async () => {
    const toastId = 'restoreFilelist';
    setBusy('import');
    toast.loading('Revertendo inserções no filelist…', { id: toastId });
    try {
      const result = await restoreFilelistInserts(version);
      if (result.applied === 0 && result.warnings.length === 0) {
        toast.info('Nenhuma inserção no filelist para reverter.', { id: toastId });
        return;
      }
      invalidateFilelistCache();
      await actions.reload();
      const avisos = result.warnings.length > 0 ? ` (${result.warnings.length} aviso(s))` : '';
      toast.success(
        `Inserções revertidas: ${result.bins} white bin(s), ${result.filelists} filelist(s), ` +
          `${result.bytes} byte(s) devolvidos.${avisos}`,
        {
          id: toastId,
          ...(result.warnings.length > 0
            ? { description: result.warnings.slice(0, 3).join('; ') }
            : {}),
        },
      );
    } catch (error) {
      toast.error(parseError(error), { id: toastId });
    } finally {
      setBusy(null);
    }
  };

  /**
   * "Todas": marcado/indeterminate marca tudo (imagens e cópias
   * selecionáveis — inclusive as iguais ao binário, que forçam a
   * regravação); marcado desmarca tudo.
   */
  const toggleAllEntries = () => {
    setPendingPreview((prev) => {
      if (!prev) return prev;
      const next = nextAllSelected(prev);
      return {
        ...prev,
        entries: prev.entries.map((e) =>
          e.validation_error
            ? e
            : {
                ...e,
                selected: next,
                copies: (e.copies ?? []).map((c) => ({ ...c, selected: next })),
              },
        ),
      };
    });
  };

  /** Checkbox da linha: controla SÓ a imagem. */
  const toggleEntry = (entryId: string) => {
    setPendingPreview((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        entries: prev.entries.map((e) => (e.id === entryId ? { ...e, selected: !e.selected } : e)),
      };
    });
  };

  /** Checkbox do collapse: controla SÓ a cópia. */
  const toggleCopy = (entryId: string, copyId: string) => {
    setPendingPreview((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        entries: prev.entries.map((e) =>
          e.id === entryId
            ? {
                ...e,
                copies: (e.copies ?? []).map((c) =>
                  c.id === copyId ? { ...c, selected: !c.selected } : c,
                ),
              }
            : e,
        ),
      };
    });
  };

  return {
    busy,
    readOnly,
    pendingImport,
    setPendingImport,
    pendingPreview,
    importOptions,
    setImportOptions,
    openImportImages,
    confirmImportImages,
    closeImportImages: () => setPendingPreview(null),
    openAction,
    onExport,
    onExtractTo,
    onImport,
    runImport,
    onRefresh,
    onRestoreInserts,
    toggleAllEntries,
    toggleEntry,
    toggleCopy,
  };
}
