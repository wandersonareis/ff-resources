// Contrato único de UI (Wails-agnostic). Cada app ffx/ff13 injeta um adapter
// que implementa esta interface. Este pacote NUNCA importa bindings/* gerados
// pelo Wails.
//
// Os tipos são estruturais: o adapter converte o DTO do backend no formato
// declarado aqui, de modo que os componentes compartilhados não dependam do
// modelo gerado por cada app.
//
// Os DTOs de imagem ainda são `any`: a camada de imagens do FFX é a próxima a
// ser tipada/movida; os métodos existem para a árvore e o painel compilarem e
// o FF13 respondê-los com stubs.

export interface TagSuggestion {
  tag: string;
  label: string;
  key: string;
  /** Swatch representativo (só CLR); nunca vira byte. */
  hex?: string;
  /** Grupo de exibição (ex.: "ex" para as semitransparentes). */
  group?: string;
  /** Cor semitransparente no jogo (Ex*). */
  semiTransparent?: boolean;
  /** White: reset para a cor natural do texto. */
  reset?: boolean;
}

export interface TagCatalogEntry {
  tag: string;
  prefix: string;
  hint: string;
  values: TagSuggestion[];
}

export interface TagCatalog {
  version: string;
  /** Idioma da paleta/tabela servida (default "us"). */
  lang?: string;
  tags: TagCatalogEntry[];
}

export interface EntrySummary {
  kind: string;
  id: string;
  /** Chave canônica do backend (ex.: ffx/event/...), quando houver. */
  key?: string;
  /** Categoria de CONTEÚDO (system, mission, event, ...) — não traduzida. */
  type?: string;
  title?: string;
  count?: number;
  /**
   * false = o arquivo existe mas o app NÃO sabe decodificar (imagem, script,
   * índice…). A árvore mostra a folha como "fora do escopo" e ela não entra
   * em seleção/exportação. Ausente = texto (comportamento padrão).
   */
  text?: boolean;
  /**
   * Rótulo de EXIBIÇÃO da entrada, quando o backend manda um (FF13: a
   * unidade de texto, sem o sufixo de idioma). Ausente = a árvore resolve o
   * rótulo pelo id.
   */
  label?: string;
  /**
   * "loading" = o arquivo existe na árvore (descoberta concluída) mas o
   * conteúdo ainda está sendo lido no backend. A folha aparece com spinner;
   * abrir espera a carga em vez de dar "não encontrado".
   */
  status?: 'loading' | 'ready';
  /** Idiomas presentes na entrada (FF13: a unidade de texto). */
  langs?: string[];
  /** Arquivo editável da entrada (FF13: o "_us" da unidade). */
  primary?: string;
  /** Idioma -> arquivo (FF13: os idiomas da unidade). */
  files?: Record<string, string>;
  /** Tipo do arquivo editável (FF13: "ztr" | "wpd") ou container de imagem. */
  container?: string;
  /** FF13: a entrada é imagem/container de imagem. */
  image?: boolean;
  /** FF13: referência opaca da imagem/container. */
  imageRef?: string;
  /** FF13: nó expansível de container de imagem. */
  imageContainer?: boolean;
  /** FF13: binário .imgb pareado. */
  pair?: string;
  /**
   * FF13: a unidade já tem tradução em `mods/partN`. A árvore prefere essa
   * cópia quando o mesmo caminho visual aparece no `data/` global e no de
   * zona.
   */
  modded?: boolean;
}

export interface FileEntry {
  kind: string;
  id: string;
  metadata?: Record<string, unknown>;
}

export interface Collection {
  metadata?: Record<string, unknown>;
  rows?: Array<Record<string, unknown>>;
}

export interface ImportSummary {
  applied?: number;
  changed?: number;
  missing?: Array<Record<string, unknown>>;
  errors?: string[];
}

export interface ExportItem {
  kind: string;
  ids: string[];
  complete: boolean;
}

export interface Language {
  code: string;
  name: string;
}

export interface GameApi {
  /* ---- texto ---- */
  getTagCatalog(version: string, lang?: string): Promise<TagCatalog | null>;
  listEntries(kind: string, version: string): Promise<EntrySummary[]>;
  getEntry(kind: string, id: string, version: string, lang?: string): Promise<FileEntry>;
  getCollection(kind: string, ids: string[], version: string): Promise<Collection>;
  applyCollection(kind: string, c: Collection, version: string): Promise<void>;
  /** Aplica o lote de uma sessão .vbf (FFX; FF13 não usa). */
  applyVbfCollection(vbfPath: string, kind: string, version: string, c: Collection): Promise<void>;
  exportEntry(kind: string, id: string, version: string, langs: string[]): Promise<string[]>;
  exportArtifact(
    items: ExportItem[],
    version: string,
    langs: string[],
    format: string,
  ): Promise<string[]>;
  importEntry(kind: string, id: string, version: string): Promise<string[]>;
  importFile(path: string, version: string): Promise<number>;
  importTranslated(version: string, kinds: string[]): Promise<any>;
  previewImport(path: string, version: string): Promise<any>;
  previewImportTranslated(version: string, kinds: string[]): Promise<any>;
  previewSavePrune(kind: string, version: string): Promise<any[]>;
  countChangedTexts(imported: Collection, store: Collection): Promise<number>;
  searchTextEntries(version: string, query: string): Promise<any>;
  warmTextSearch(version: string): Promise<void>;
  revealEntryFile(kind: string, id: string, version: string): Promise<void>;
  getMetadata(query: string): Promise<any>;
  selectImportFile(): Promise<string>;
  listLanguages(): Promise<Language[]>;

  /* ---- imagens (DTO tipado numa fase posterior) ---- */
  getImageEntry(kind: string, id: string, version: string): Promise<any>;
  /** FF13: preview de tela .ykd. Opcional para os outros apps. */
  getYkdScreen?(kind: string, id: string, version: string): Promise<any>;
  /** FF13: edita a caixa de um texto .ykd e grava em mods/. */
  applyYkdText?(kind: string, id: string, version: string, patch: any): Promise<any>;
  imageExists(kind: string, id: string, version: string): Promise<boolean>;
  imageDuplicates(kind: string, id: string, version: string): Promise<any>;
  imageSelectionCopies(kind: string, ids: string[], version: string): Promise<string[]>;
  imageExportPath(kind: string, id: string, format: string, version: string): Promise<string>;
  extractImage(kind: string, id: string, version: string): Promise<string[]>;
  extractImageGroup(kind: string, id: string, targets: string[], version: string): Promise<any>;
  extractImageSelection(kind: string, ids: string[], version: string): Promise<any>;
  importImage(kind: string, id: string, ddsPath: string, version: string): Promise<void>;
  importImageGroup(
    kind: string,
    id: string,
    ddsPath: string,
    targets: string[],
    version: string,
  ): Promise<any>;
  importImages(version: string, selection: any): Promise<any>;
  previewImportImages(version: string): Promise<any>;
  refreshImageDuplicates(kind: string, version: string): Promise<void>;
  replicateImage(kind: string, id: string, targets: string[], version: string): Promise<any>;
  deleteImages(
    kind: string,
    id: string,
    targets: string[],
    scope: string,
    version: string,
  ): Promise<any>;
  deleteImageSelection(
    kind: string,
    ids: string[],
    withCopies: boolean,
    scope: string,
    version: string,
  ): Promise<any>;
  saveImage(kind: string, id: string, format: string, destPath: string, version: string): Promise<void>;
  saveImageTo(kind: string, id: string, format: string, destRoot: string, version: string): Promise<string>;
  selectImageExportDir(version?: string): Promise<string>;
  selectImageFile(id: string, version: string): Promise<string>;
  selectImageSavePath(
    format: string,
    suggestedName: string,
    id: string,
    version: string,
  ): Promise<string>;

  /* ---- log ---- */
  writeLog(level: string, message: string, fields?: Record<string, unknown>): Promise<void>;
}
