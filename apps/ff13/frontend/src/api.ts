// Adapter do app FF13: implementa o contrato GameApi do pacote compartilhado
// sobre o adaptor local (bindings/ffxresources/textservice), que por sua vez
// fala com o FF13Service do Go. Imagens/busca/import ainda são stubs.
import * as text from '@/bindings/ffxresources/textservice';
import * as image from '@/bindings/ffxresources/imageservice';
import type { GameApi } from '@ff-resources/ui/api/GameApi';

const emptyBatchResult = { ok: 0, failed: 0 };

export const ff13Api: GameApi = {
  /* ---- texto ---- */
  // O adaptador de bindings já converte a version para a part (partOf):
  // passar o valor cru — repetir a conversão aqui chamaria um helper que
  // não existe neste módulo (ReferenceError na montagem do editor).
  getTagCatalog: (version, lang) => text.GetTagCatalog(version, lang) as Promise<any>,
  listEntries: async (kind, version) => {
    // A COLEÇÃO de data/partN: o inventário inteiro do que foi extraído. Só
    // as unidades de TEXTO viram folha — os arquivos fora do escopo ficam no
    // inventário do backend (o filelist é o lugar de buscar arquivo).
    const entries = ((await text.ListDataEntries(version)) ?? []) as Array<{
      id: string;
      label?: string;
      type?: string;
      kind?: string;
      text?: boolean;
      langs?: string[] | null;
      primary?: string;
      files?: Record<string, string> | null;
      container?: string;
      image?: boolean;
      imageRef?: string;
      imageContainer?: boolean;
      pair?: string;
      modded?: boolean;
    }>;
    if (kind === 'images') {
      return entries
        .filter((e) => e.image === true)
        .map((e) => ({
          kind: 'images' as const,
          id: e.imageRef ?? e.id,
          key: e.imageRef ?? e.id,
          type: e.type,
          text: false,
          title: e.id,
          label: e.label ?? e.id,
          image: true,
          imageRef: e.imageRef,
          imageContainer: e.imageContainer === true,
          container: e.container,
          pair: e.pair,
          source: 'data' as const,
        }));
    }
    return (
      entries
        .filter((e) => e.text === true)
        // O kind pedido filtra pela FAMÍLIA (a pasta): `ztr`/vazio = todos.
        // O backend manda o kind de cada unidade; o `ac` do part1 (que só
        // existe nas zonas) e o `scenario` do part3 caem no filtro da part.
        .filter((e) => kind === 'ztr' || !kind || e.kind === kind)
        .map((e) => ({
          kind: (e.kind ?? kind) as typeof kind,
          id: e.id,
          key: e.id,
          type: e.type,
          text: true,
          title: e.id,
          // Rótulo da UNIDADE (nome-base, sem o sufixo de idioma) e os idiomas
          // que ela tem: a árvore usa o rótulo e o diálogo de tradução usa os
          // idiomas como REFERÊNCIAS somente-leitura.
          label: e.label,
          langs: e.langs ?? undefined,
          primary: e.primary,
          files: e.files ?? undefined,
          container: e.container,
          // A cópia que já tem tradução em mods/ — a árvore a prefere no dedupe.
          modded: e.modded === true,
        }))
    );
  },
  getEntry: (kind, id, version, lang, origin) =>
    text.GetTextEntry(kind, id, version, lang, origin) as Promise<any>,
  getCollection: async () => ({}),
  applyCollection: (kind, c, version) =>
    text.ApplyTextCollection(kind, version, c as never) as Promise<void>,
  applyVbfCollection: async () => {},
  exportEntry: (kind, id, version, langs) =>
    text.ExportEntry(kind, id, version, langs) as Promise<string[]>,
  exportArtifact: (items, version, langs, format) =>
    text.ExportArtifactBatch(items as never, version, langs, format) as Promise<string[]>,
  importEntry: async () => [],
  importFile: (path, version, options) => text.ImportFile(path, version, options),
  importTranslated: (version, kinds, options) =>
    text.ImportTranslated(version, kinds, options) as Promise<any>,
  previewImport: (path, version, options) =>
    text.PreviewImport(path, version, options) as Promise<any>,
  previewImportTranslated: (version, kinds) =>
    text.PreviewImportTranslated(version, kinds) as Promise<any>,
  previewSavePrune: async () => [],
  countChangedTexts: async () => 0,
  searchTextEntries: async (_version, query) => ({
    results: [],
    totalRows: 0,
    totalFiles: 0,
    truncated: false,
    query,
  }),
  warmTextSearch: async () => {},
  revealEntryFile: async () => {},
  getMetadata: async (query) => ({ key: query }),
  selectImportFile: (version) => text.SelectImportFile(version),
  listLanguages: () => text.ListLanguages() as Promise<any>,

  /* ---- imagens ---- */
  getImageEntry: (_kind, id, version) => image.GetImageEntry(version, id) as Promise<any>,
  getYkdScreen: (_kind, id, version) => image.GetYkdScreen(version, id) as Promise<any>,
  applyYkdText: (_kind, id, version, patch) =>
    image.ApplyYkdText(version, id, patch) as Promise<any>,
  imageExists: (_kind, id, version) => image.ImageExists(version, id),
  imageDuplicates: async () => ({ duplicates: [], dupPayload: 0 }),
  imageSelectionCopies: async () => [],
  imageExportPath: (_kind, id, format, version) =>
    image.ImageExportPath(version, id, format) as Promise<string>,
  extractImage: (_kind, id, version) =>
    image.ExtractImageSelection(version, [id], '') as Promise<any>,
  extractImageGroup: (_kind, id, targets, version) =>
    image.ExtractImageSelection(version, [id, ...(targets ?? [])], '') as Promise<any>,
  extractImageSelection: (_kind, ids, version) =>
    image.ExtractImageSelection(version, ids, '') as Promise<any>,
  importImage: (_kind, id, ddsPath, version, options) =>
    image.ImportImage(version, id, ddsPath, options),
  importImageGroup: (_kind, id, ddsPath, targets, version, options) =>
    image.ImportImageGroup(version, id, ddsPath, targets, options),
  importImages: (version, selection) => image.ImportImages(version, selection),
  previewImportImages: (version) => image.PreviewImportImages(version),
  restoreFilelistInserts: (version) => image.RestoreFilelistInserts(version),
  refreshImageDuplicates: async () => {},
  replicateImage: async () => emptyBatchResult,
  deleteImages: (_kind, id, targets, scope, version) =>
    image.DeleteImages(version, [id, ...(targets ?? [])], scope) as Promise<any>,
  deleteImageSelection: (_kind, ids, _withCopies, scope, version) =>
    image.DeleteImageSelection(version, ids, scope) as Promise<any>,
  saveImage: (_kind, id, format, destPath, version) =>
    image.SaveImage(version, id, format, destPath),
  saveImageTo: async (_kind, id, format, destRoot, version) => {
    const path = await image.ImageExportPath(version, id, format);
    if (!destRoot) {
      await image.SaveImage(version, id, format, path);
      return path;
    }
    // O export já inclui <part>; o destino escolhido é a raiz DA PART.
    // Removemos "…/edits/images/<part>/" e recompomos sob destRoot sem
    // duplicar a part.
    const inside = path.replace(/^.*[\\/]edits[\\/]images[\\/][^\\/]+[\\/]/, '');
    const sep = destRoot.includes('\\') ? '\\' : '/';
    const target = `${destRoot.replace(/[\\/]+$/, '')}${sep}${inside.replace(/[\\/]/g, sep)}`;
    await image.SaveImage(version, id, format, target);
    return target;
  },
  selectImageExportDir: (version) => image.SelectImageExportDir(version ?? 'part1'),
  selectImageFile: (_id, version) => image.SelectImageFile(version, _id),
  selectImageSavePath: (format, suggestedName, id, version) =>
    image.SelectImageSavePath(version, format, suggestedName, id),

  /* ---- log ---- */
  writeLog: async () => {},
};
