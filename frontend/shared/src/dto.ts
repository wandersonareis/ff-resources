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
  files?: Record<string, string | undefined>;
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
  identical: boolean;
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
  assetPath?: string;
  pngData: string;
  ddsData?: string;
  flipped?: boolean;
  ddsPath?: string;
  pngPath?: string;
  modded: boolean;
  duplicates: ImageDuplicate[];
  dupPayload: number;
}

export interface ImageImportPreviewCopy {
  id: string;
  display_path: string;
  in_mods: boolean;
  identical: boolean;
  selected: boolean;
}

export interface ImageImportPreviewEntry {
  id: string;
  display_path: string;
  differs: boolean;
  copy_count: number;
  copy_missing_count: number;
  copies?: ImageImportPreviewCopy[];
  validation_error?: string;
  selected: boolean;
}

export interface ImageImportPreview {
  root: string;
  entries: ImageImportPreviewEntry[];
  max: number;
  new_count: number;
  identical_count: number;
  errors: string[];
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

/* ---- YKD (telas de UI do FF13) ---- */

export interface YkdHeader {
  unknown1: number;
  u2: number;
  u3: number;
  u4: number;
  u5: number;
}

export interface YkdText {
  width: number;
  height: number;
  fontSize: number;
  flags: number;
  color1: number;
  color2: number;
  color3: number;
  mode: number;
  originalWidth: number;
  maxWidth: number;
}

export interface YkdTextPatch {
  blockIndex: number;
  width: number;
}

export interface YkdBlock {
  type: number;
  index: number;
  parentIndex: number;
  unknown: number;
  scale: number[];
  rotation: number[];
  position: number[];
  pivot: number[];
  extra: number[];
  resourceIndexes?: number[] | null;
  text?: YkdText | null;
  entries?: string[] | null;
}

export interface YkdResource {
  index: number;
  type: number;
  typeName: string;
  name: string;
  sourceX: number;
  sourceY: number;
  sourceWidth: number;
  sourceHeight: number;
  viewWidth: number;
  viewHeight: number;
  flags: number;
  colors: number[];
  texture?: string | null;
  missing: boolean;
}

export interface YkdItem {
  kind: 'sprite' | 'text';
  blockIndex: number;
  blockType: number;
  nodeIndex: number;
  x: number;
  y: number;
  scaleX: number;
  scaleY: number;
  rotation: number;
  pivotX: number;
  pivotY: number;
  flipX: boolean;
  flipY: boolean;
  resourceIndex: number;
  texture?: string | null;
  missing: boolean;
  sourceX: number;
  sourceY: number;
  sourceWidth: number;
  sourceHeight: number;
  viewWidth: number;
  viewHeight: number;
  flags: number;
  text?: YkdText | null;
}

export interface YkdTexture {
  pngData: string;
  width: number;
  height: number;
  source: string;
}

export interface YkdStats {
  blocks: number;
  resources: number;
  items: number;
  textures: number;
  missingTextures: number;
  textNodes: number;
  texturedItems: number;
}

export interface YkdScreenRef {
  name: string;
  ref: string;
}

export interface YkdScreen {
  canvasWidth: number;
  canvasHeight: number;
  header: YkdHeader;
  blocks?: YkdBlock[] | null;
  resources?: YkdResource[] | null;
  items?: YkdItem[] | null;
  textures: Record<string, YkdTexture>;
  missingTextures?: string[] | null;
  siblings?: YkdScreenRef[] | null;
  stats: YkdStats;
}
