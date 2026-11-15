// Adapter do app FFX: implementa o contrato GameApi do pacote compartilhado
// sobre os bindings Wails gerados para este app.
import * as app from '@/bindings/ffresources/apps/ffx/backend/service/app';
import * as image from '@/bindings/ffresources/apps/ffx/backend/service/imageservice';
import * as text from '@/bindings/ffresources/apps/ffx/backend/service/textservice';
import type { GameApi } from '@ff-resources/ui/api/GameApi';

type V = Parameters<typeof text.ListTextEntries>[1];

const v = (version: string) => version as V;

export const ffxApi: GameApi = {
  /* ---- texto ---- */
  getTagCatalog: (version) => text.GetTagCatalog(v(version)) as never,
  listEntries: async (kind, version) => {
    const rows = (await text.ListTextEntries(kind, v(version))) as Array<{ id: string; key: string }>;
    return rows.map((r) => ({ kind, id: r.id, key: r.key, type: kind, title: r.id }));
  },
  getEntry: (kind, id, version) => text.GetTextEntry(kind, id, v(version)) as never,
  getCollection: (kind, ids, version) => text.GetTextCollection(kind, v(version), ids) as never,
  applyCollection: (kind, c, version) =>
    text.ApplyTextCollection(kind, v(version), c as never) as never,
  applyVbfCollection: (vbfPath, kind, version, c) =>
    text.ApplyVbfTextCollection(vbfPath, kind, v(version), c as never) as never,
  exportEntry: (kind, id, version, langs) =>
    text.ExportEntry(kind, id, v(version), langs) as never,
  exportArtifact: (items, version, langs, format) =>
    text.ExportArtifactBatch(items as never, v(version), langs, format) as never,
  importEntry: (kind, id, version) => text.ImportEntry(kind, id, v(version)) as never,
  importFile: (path, version) => text.ImportFile(path, v(version)) as never,
  importTranslated: (version, kinds) => text.ImportTranslated(v(version), kinds) as never,
  previewImport: (path, version) => text.PreviewImport(path, v(version)) as never,
  previewImportTranslated: (version, kinds) =>
    text.PreviewImportTranslated(v(version), kinds) as never,
  previewSavePrune: (kind, version) => text.PreviewSavePrune(kind, v(version)) as never,
  countChangedTexts: (imported, store) =>
    text.CountChangedTexts(imported as never, store as never) as never,
  searchTextEntries: (version, query) => text.SearchTextEntries(v(version), query) as never,
  warmTextSearch: (version) => text.WarmTextSearch(v(version)) as never,
  revealEntryFile: (kind, id, version) => text.RevealEntryFile(kind, id, v(version)) as never,
  getMetadata: (query) => text.GetMetadata(query) as never,
  selectImportFile: () => text.SelectImportFile() as never,
  listLanguages: () => text.ListLanguages() as never,

  /* ---- imagens ---- */
  getImageEntry: (kind, id, version) => image.GetImageEntry(kind, id, v(version)) as never,
  imageExists: (kind, id, version) => image.ImageExists(kind, id, v(version)) as never,
  imageDuplicates: (kind, id, version) => image.ImageDuplicates(kind, id, v(version)) as never,
  imageSelectionCopies: (kind, ids, version) =>
    image.ImageSelectionCopies(kind, ids, v(version)) as never,
  imageExportPath: (kind, id, format, version) =>
    image.ImageExportPath(kind, id, format, v(version)) as never,
  extractImage: (kind, id, version) => image.ExtractImage(kind, id, v(version)) as never,
  extractImageGroup: (kind, id, targets, version) =>
    image.ExtractImageGroup(kind, id, targets, v(version)) as never,
  extractImageSelection: (kind, ids, version) =>
    image.ExtractImageSelection(kind, ids, v(version)) as never,
  importImage: (kind, id, ddsPath, version) =>
    image.ImportImage(kind, id, ddsPath, v(version)) as never,
  importImageGroup: (kind, id, ddsPath, targets, version) =>
    image.ImportImageGroup(kind, id, ddsPath, targets, v(version)) as never,
  importImages: (version, selection) => image.ImportImages(v(version), selection) as never,
  previewImportImages: (version) => image.PreviewImportImages(v(version)) as never,
  refreshImageDuplicates: (kind, version) =>
    image.RefreshImageDuplicates(kind, v(version)) as never,
  replicateImage: (kind, id, targets, version) =>
    image.ReplicateImage(kind, id, targets, v(version)) as never,
  deleteImages: (kind, id, targets, scope, version) =>
    image.DeleteImages(kind, id, targets, scope, v(version)) as never,
  deleteImageSelection: (kind, ids, withCopies, scope, version) =>
    image.DeleteImageSelection(kind, ids, withCopies, scope, v(version)) as never,
  saveImage: (kind, id, format, destPath, version) =>
    image.SaveImage(kind, id, format, destPath, v(version)) as never,
  saveImageTo: (kind, id, format, destRoot, version) =>
    image.SaveImageTo(kind, id, format, destRoot, v(version)) as never,
  selectImageExportDir: () => image.SelectImageExportDir() as never,
  selectImageFile: (id, version) => image.SelectImageFile(id, v(version)) as never,
  selectImageSavePath: (format, suggestedName, id, version) =>
    image.SelectImageSavePath(format, suggestedName, id, v(version)) as never,

  /* ---- log ---- */
  writeLog: (level, message, fields) => app.WriteLog(level, message, fields ?? {}) as never,
};
