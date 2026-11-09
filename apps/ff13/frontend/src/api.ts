// Adapter do app FF13: implementa o contrato GameApi do pacote compartilhado
// sobre o adaptor local (bindings/ffxresources/textservice), que por sua vez
// fala com o FF13Service do Go. Imagens/busca/import ainda são stubs.
import * as text from '@/bindings/ffxresources/textservice';
import type { GameApi } from '@ff-resources/ui/api/GameApi';

const emptySummary = { entryCount: 0, changedTexts: 0, warnings: [], errors: [] };
const emptyBatch = {
  root: '',
  files: [],
  entryCount: 0,
  changedTexts: 0,
  warnings: [],
  errors: [],
};
const emptyImageResult = { imported: 0, skipped: 0, errors: [] };
const emptyBatchResult = { ok: 0, failed: 0 };

export const ff13Api: GameApi = {
  /* ---- texto ---- */
  getTagCatalog: (version) => text.GetTagCatalog(partOf(version)) as Promise<any>,
  listEntries: async (kind, version) => {
    // A COLEÇÃO de data/partN: o inventário inteiro do que foi extraído. Só
    // as unidades de TEXTO viram folha — os arquivos fora do escopo ficam no
    // inventário do backend (o filelist é o lugar de buscar arquivo).
    const entries = (await text.ListDataEntries(version)) as Array<{
      id: string;
      label?: string;
      type?: string;
      kind?: string;
      text?: boolean;
      langs?: string[] | null;
      primary?: string;
      files?: Record<string, string> | null;
      container?: string;
      modded?: boolean;
    }>;
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
  getEntry: (kind, id, version, lang) => text.GetTextEntry(kind, id, version, lang) as Promise<any>,
  getCollection: async () => ({}),
  applyCollection: (kind, c, version) =>
    text.ApplyTextCollection(kind, version, c as never) as Promise<void>,
  applyVbfCollection: async () => {},
  exportEntry: (kind, id, version, langs) =>
    text.ExportEntry(kind, id, version, langs) as Promise<string[]>,
  exportArtifact: (items, version, langs, format) =>
    text.ExportArtifactBatch(items as never, version, langs, format) as Promise<string[]>,
  importEntry: async () => [],
  importFile: async () => 0,
  importTranslated: async () => emptyBatch,
  previewImport: async () => emptySummary,
  previewImportTranslated: async () => emptyBatch,
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
  selectImportFile: async () => '',
  listLanguages: () => text.ListLanguages() as Promise<any>,

  /* ---- imagens (fase posterior) ---- */
  getImageEntry: async () => ({ metadata: {}, preview: null }),
  imageExists: async () => false,
  imageDuplicates: async () => ({ ids: [] }),
  imageSelectionCopies: async () => [],
  imageExportPath: async () => '',
  extractImage: async () => [],
  extractImageGroup: async () => emptyBatchResult,
  extractImageSelection: async () => emptyBatchResult,
  importImage: async () => {},
  importImageGroup: async () => emptyImageResult,
  importImages: async () => emptyImageResult,
  previewImportImages: async () => ({ entries: [], errors: [] }),
  refreshImageDuplicates: async () => {},
  replicateImage: async () => emptyBatchResult,
  deleteImages: async () => emptyBatchResult,
  deleteImageSelection: async () => emptyBatchResult,
  saveImage: async () => {},
  saveImageTo: async () => '',
  selectImageExportDir: async () => '',
  selectImageFile: async () => '',
  selectImageSavePath: async () => '',

  /* ---- log ---- */
  writeLog: async () => {},
};
