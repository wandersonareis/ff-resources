// Tipos canônicos do DTO de texto (espelham internal/shared/dto no Go).
//
// São estruturais: os bindings gerados de cada app (classes) e estes
// interfaces são compatíveis, então a UI compartilhada não depende do modelo
// gerado pelo Wails de nenhum app.

export interface TextRow {
  index: number;
  character?: string;
  kind?: string;
  name?: string;
  hash?: Record<string, string>;
  text: Record<string, string>;
  original?: Record<string, string>;
  source?: string;
  target?: string;
  missingInOriginal?: boolean;
  missingInTranslated?: boolean;
  divergent?: boolean;
}

export interface RefLink {
  text: string;
  original?: string;
  originals?: Record<string, string>;
  sourceId?: string;
  sourceIndex?: number;
  sourceName?: string;
}

export interface Metadata {
  key: string;
  row_count?: number;
  id?: string;
  is_dir?: boolean;
  /** Layout do recurso (FF13: standard | mission | system). */
  layout?: string;
  /** idioma -> caminho físico (FF13: txt_XX.ztr). */
  files?: Record<string, string>;
}

export interface FileEntry {
  metadata: Metadata;
  rows: TextRow[];
  refs?: Record<string, RefLink>;
}

export type Collection = Record<string, FileEntry>;

// Fábricas (equivalentes ao `createFrom` dos bindings do Wails): preenchem os
// campos obrigatórios para o objeto poder virar um DTO sem cast.

export function textRow(source: Partial<TextRow> = {}): TextRow {
  return { index: 0, text: {}, ...source } as TextRow;
}

export function fileEntry(source: Partial<FileEntry> = {}): FileEntry {
  return { metadata: metadata(), rows: [], ...source } as FileEntry;
}

export function metadata(source: Partial<Metadata> = {}): Metadata {
  return { key: '', ...source } as Metadata;
}

/* ---- imagens (FFX .dds.phyre; a UI compartilhada só transporta) ---- */

export interface ImageDuplicate {
  id: string;
  key: string;
  vbfPath?: string;
  modded: boolean;
}

export interface ImageDuplicates {
  duplicates: ImageDuplicate[];
  dupPayload: number;
}

export interface ImageEntry {
  metadata: Metadata;
  source: string;
  format: string;
  width: number;
  height: number;
  mipmapCount: number;
  maxMipmapLevel: number;
  preview?: string;
}

export interface ImageImportPreviewCopy {
  id: string;
  display_path: string;
  in_mods: boolean;
  identical: boolean;
}

export interface ImageImportPreviewEntry {
  id: string;
  display_path: string;
  differs: boolean;
  copies?: ImageImportPreviewCopy[];
}

export interface ImageImportPreview {
  root: string;
  entries: ImageImportPreviewEntry[];
  max: number;
  new_count: number;
}

export interface ImageImportResult {
  updated: string[];
  failed: string[];
  total: number;
}

export interface ImageImportSelectionEntry {
  id: string;
  image: boolean;
  copies?: string[];
}

export interface ImageImportSelection {
  entries: ImageImportSelectionEntry[];
}

export interface BatchResult {
  done: string[];
  failed: string[];
  total: number;
}