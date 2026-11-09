import type * as dto from '../dto';
import type { EntrySummary } from '../api/GameApi';
import { getCurrentApi } from '../api/registry';
import { type EntryKind, resolveEntryLabel } from './display-names';
import type { ExportFormat } from './export-selection';

// A UI é agnóstica de jogo: a versão é um id (string) e o acesso ao backend
// passa pelo GameApi registrado no boot (registry).
type GameVersionId = string;
const api = () => getCurrentApi();
export interface EntryRow {
  kind: EntryKind;
  id: string;
  key: string;
  label: string;
  /** Categoria de conteúdo vinda do backend (system, mission, event, ...). */
  type?: string;
  /**
   * false = arquivo que o app NÃO decodifica (imagem, script, índice…): a
   * árvore mostra a folha como "fora do escopo" e ela não é selecionável.
   * Ausente = texto (comportamento padrão).
   */
  text?: boolean;
  /**
   * Presente quando a entrada veio do navegador de .vbf (não da árvore
   * data/): `root` é o container absoluto e `path` o caminho interno do
   * arquivo. É o que muda a carga para os bindings VBF e mantém a entrada
   * SOMENTE LEITURA (nada aqui escreve no .vbf).
   */
  vbf?: { root: string; path: string };
  /** Idiomas presentes na entrada (FF13: a unidade de texto). */
  langs?: string[];
  /**
   * Idioma que a FOLHA representa (FF13: o idioma do arquivo clicado no
   * filelist, "us"/"jp"/…). Ausente = a entrada é o idioma default (editável).
   */
  lang?: string;
  /** Arquivo editável da entrada (FF13: o "_us" da unidade). */
  primary?: string;
  /** Idioma -> arquivo (FF13: os idiomas da unidade). */
  files?: Record<string, string>;
  /** Tipo do arquivo editável (FF13: "ztr" | "wpd"). */
  container?: string;
  /**
   * FF13: a unidade já tem tradução em `mods/partN`. A árvore usa para
   * preferir a cópia traduzida quando o mesmo caminho visual aparece no
   * `data/` global e no de zona (as duas são idênticas no original).
   */
  modded?: boolean;
}

/**
 * Extrai o código de idioma do caminho do arquivo dentro do container
 * (padrão `new_<code>pc`: "new_jppc" → "jp", "new_sppc" → "sp").
 * Null quando o caminho não tem raiz de localização.
 */
export function locFromVbfPath(path: string): string | null {
  const m = /\/new_([a-z]{2})pc\//.exec(path.replace(/\\/g, '/'));
  return m ? m[1] : null;
}

// Kinds servidos por versão na sidebar. As expansões não têm dicionário
// próprio: eternalcalm (sob o ffx) mostra só events (scene*.bin); lastmiss
// (sob o ffx2) mostra events + objects (kernel lm_*). O lockit (kit de
// localização do menu/launcher) existe em FFX e FFX-2. Os painéis de ajuda
// (.sps2 em help/) são FFX-only: a árvore do ffx2 não tem a pasta help/.
export function entryKindsFor(version: GameVersionId): EntryKind[] {
  // battletext (battle/btl) e cloud (cloudsave) existem em FFX e FFX-2;
  // tutorial (menu/tutorial.msb) é só FFX-2. As expansões dividem a árvore
  // do jogo-pai mas não têm essas folhas próprias.
  if (version === 'eternalcalm') return ['events'];
  if (version === 'lastmiss') return ['events', 'objects'];
  if (version === 'ffx')
    return ['events', 'objects', 'macro', 'lockit', 'help', 'battletext', 'cloud', 'menumain'];
  return ['events', 'objects', 'macro', 'lockit', 'battletext', 'cloud', 'tutorial'];
}

/**
 * Kinds de IMAGEM (.dds.phyre). Só ffx e ffx2: as expansões (eternalcalm
 * sob o ffx, lastmiss sob o ffx2) dividem a árvore do jogo-pai mas não
 * têm texturas próprias.
 *
 * Fica separado de entryKindsFor de propósito — esse é o conjunto de kinds
 * de TEXTO (escopo do Exportar/JSON, save-all e preload), e imagem não
 * participa de nada disso.
 */
export function imageKindsFor(version: GameVersionId): EntryKind[] {
  if (version === 'lastmiss' || version === 'eternalcalm') return [];
  return ['images'];
}

/** Todos os kinds da árvore da versão (texto + imagem), em ordem canônica. */
export function allKindsFor(version: GameVersionId): EntryKind[] {
  return [...entryKindsFor(version), ...imageKindsFor(version)];
}

/**
 * Fluxo novo (DTO): o backend envia o sumário canônico (id + key) e o DTO
 * estruturado sob demanda. Os artefatos de export/import são JSON + .strings
 * em mods/edits; não há mais árvore de diretórios nem .txt.
 */
export async function loadKindEntries(
  kind: EntryKind,
  version: GameVersionId,
): Promise<EntryRow[]> {
  const summaries: EntrySummary[] = await api().listEntries(
    kind,
    version,
  );
  return (summaries ?? [])
    .map((s) => ({
      kind,
      id: s.id,
      key: s.key,
      type: s.type,
      text: s.text,
      // O backend pode mandar o rótulo (FF13: a unidade de texto, sem o
      // sufixo de idioma). Sem ele, a árvore resolve pelo id (FFX).
      label: s.label ?? resolveEntryLabel(kind, s.id),
      langs: s.langs,
      primary: s.primary,
      files: s.files,
      container: s.container,
      modded: s.modded,
    }))
    .sort((a, b) => a.label.localeCompare(b.label));
}

/** Busca no texto decodificado de data/ e mods/ (idioma us), sem trazer DTOs. */
export function searchTextEntries(
  version: GameVersionId,
  query: string,
): Promise<any> {
  return api().searchTextEntries(version, query);
}

/**
 * Pré-aquece o índice de busca da versão em background (chamado depois que a
 * árvore carrega): a primeira consulta do modal não paga a construção.
 */
export function warmTextSearch(version: GameVersionId): void {
  api().warmTextSearch(version);
}

export async function loadEntry(
  kind: EntryKind,
  id: string,
  version: GameVersionId,
  lang?: string,
): Promise<dto.FileEntry> {
  return api().getEntry(kind, id, version, lang);
}

// ---- Imagens (.dds.phyre) ----
// kind=images não é texto: não tem rows, não participa de export/import de
// texto. Cada chamada abaixo espelha um binding do App.

/** Carrega a textura e devolve a pré-visualização (data URL) + metadados. */
export async function loadImage(id: string, version: GameVersionId): Promise<dto.ImageEntry> {
  return api().getImageEntry('images', id, version);
}

/** Reempacota um .dds sobre o container pristine e grava em mods/. */
export function importImage(id: string, ddsPath: string, version: GameVersionId): Promise<void> {
  return api().importImage('images', id, ddsPath, version);
}

/**
 * A textura ainda existe (data/ OU mods/)? Usada na revalidação da seleção
 * após reload: id apagado pelo delete NÃO deve ser re-carregado — o erro
 * "não encontrada em data/ nem em mods/" só faria confusão. Falha de
 * binding = não existe (não serve para exibição).
 */
export async function imageExists(id: string, version: GameVersionId): Promise<boolean> {
  try {
    return await api().imageExists('images', id, version);
  } catch {
    return false;
  }
}

/**
 * Reempacota o MESMO .dds na textura e nas cópias idênticas dela (payload
 * original igual — as réplicas da otimização do DVD). O backend valida o
 * lote antes de gravar: alvo fora do grupo é recusado inteiro.
 */
export function importImageGroup(
  id: string,
  ddsPath: string,
  targets: string[],
  version: GameVersionId,
): Promise<dto.ImageImportResult> {
  return api().importImageGroup(
    'images',
    id,
    ddsPath,
    targets,
    version,
  );
}

/**
 * Varre <TranslateLocation>/images e devolve o resumo do import: uma entrada
 * por .dds.phyre encontrado (novas, iguais ao binário e inválidas), com as
 * cópias ausentes do grupo anexadas à primeira entrada exibida de cada grupo.
 */
export function previewImportImages(version: GameVersionId): Promise<dto.ImageImportPreview> {
  return api().previewImportImages(version);
}

/**
 * Aplica a SELEÇÃO do modal: grava em mods/ apenas as imagens marcadas e as
 * cópias marcadas de cada uma (inclusive a regravação de uma idêntica ao
 * binário).
 */
export function importImages(
  version: GameVersionId,
  selection: dto.ImageImportSelection,
): Promise<dto.ImageImportResult> {
  return api().importImages(version, selection);
}

/**
 * Descarta o índice de duplicatas em cache (rebuild na próxima abertura da
 * textura) — existe para edição EXTERNA em hex editor, que o cache não vê.
 */
export function refreshImageDuplicates(version: GameVersionId): Promise<void> {
  return api().refreshImageDuplicates('images', version);
}

/**
 * Cópias da textura SEM carregar a imagem (só o índice memoizado): é o que
 * o menu de contexto e os diálogos de extrair/replicar/deletar pedem para
 * agir sobre o NÓ CLICADO, que pode não ser a entry selecionada.
 */
export function imageDuplicates(id: string, version: GameVersionId): Promise<dto.ImageDuplicates> {
  return api().imageDuplicates('images', id, version);
}

/** União de cópias adicionais dos ids selecionados, sem carregar imagens. */
export function imageSelectionCopies(ids: string[], version: GameVersionId): Promise<string[]> {
  return api().imageSelectionCopies('images', ids, version);
}

/** Extrai .dds + .png desta textura e das cópias escolhidas (lote). */
export function extractImageGroup(
  id: string,
  targets: string[],
  version: GameVersionId,
): Promise<dto.BatchResult> {
  return api().extractImageGroup(
    'images',
    id,
    targets,
    version,
  );
}

/** Extrai somente as imagens marcadas, sem incluir cópias não selecionadas. */
export function extractImageSelection(
  ids: string[],
  version: GameVersionId,
): Promise<dto.BatchResult> {
  return api().extractImageSelection(
    'images',
    ids,
    version,
  );
}

/**
 * Reempacota a IMAGEM ABERTA em mods/ das cópias escolhidas (o "dupe"): a
 * fonte é o próprio conteúdo servido ao painel, sem diálogo de arquivo.
 */
export function replicateImage(
  id: string,
  targets: string[],
  version: GameVersionId,
): Promise<dto.BatchResult> {
  return api().replicateImage('images', id, targets, version);
}

/**
 * Apaga os containers escolhidos no escopo pedido e SEMPRE os artefatos
 * derivados (.dds/.png extraídos). Escopo vem explícito do diálogo.
 */
export function deleteImages(
  id: string,
  targets: string[],
  scope: 'data' | 'mods' | 'both',
  version: GameVersionId,
): Promise<dto.BatchResult> {
  return api().deleteImages('images', id, targets, scope, version);
}

/** Delete em lote das imagens marcadas; a opção de cópias vale para cada id. */
export function deleteImageSelection(
  ids: string[],
  withCopies: boolean,
  scope: 'data' | 'mods' | 'both',
  version: GameVersionId,
): Promise<dto.BatchResult> {
  return api().deleteImageSelection(
    'images',
    ids,
    withCopies,
    scope,
    version,
  );
}

/**
 * "Abrir até o arquivo": abre o explorador já com o arquivo da entrada
 * selecionado, resolvido mods-first (o arquivo que está valendo).
 */
export function revealEntry(kind: EntryKind, id: string, version: GameVersionId): Promise<void> {
  return api().revealEntryFile(kind, id, version);
}

/** Salva .dds ou .png no caminho escolhido pelo usuário. */
export function saveImage(
  id: string,
  format: 'dds' | 'png',
  destPath: string,
  version: GameVersionId,
): Promise<void> {
  return api().saveImage('images', id, format, destPath, version);
}

/** Seletor nativo para escolher um .dds a importar (abre em mods/images/<raiz>/<dir do id>). */
export function selectImageFile(id: string, version: GameVersionId): Promise<string> {
  return api().selectImageFile(id, version);
}

/** Diálogo "Salvar como" para .dds/.png, já no diretório de trabalho da textura. */
export function selectImageSavePath(
  format: 'dds' | 'png',
  suggestedName: string,
  id: string,
  version: GameVersionId,
): Promise<string> {
  return api().selectImageSavePath(
    format,
    suggestedName,
    id,
    version,
  );
}

/** Caminho padrão de exportação (.dds ou .png) em mods/images para export direto sem diálogo. */
export function imageExportPath(
  id: string,
  format: 'dds' | 'png',
  version: GameVersionId,
): Promise<string> {
  return api().imageExportPath('images', id, format, version);
}

/**
 * Exporta .dds/.png para a pasta escolhida preservando o caminho interno da
 * textura (raiz da versão + id) sob destRoot. Devolve o caminho escrito.
 */
export function saveImageTo(
  id: string,
  format: 'dds' | 'png',
  destRoot: string,
  version: GameVersionId,
): Promise<string> {
  return api().saveImageTo('images', id, format, destRoot, version);
}

/** Seletor nativo de pasta do "Exportar para…" (default mods/images). "" = cancelou. */
export function selectImageExportDir(): Promise<string> {
  return api().selectImageExportDir();
}

/** Exporta a entrada em JSON + .strings (mods/edits). */
export async function exportEntry(
  kind: EntryKind,
  id: string,
  version: GameVersionId,
  langs: string[] = [],
): Promise<string[]> {
  return api().exportEntry(kind, id, version, langs);
}

/** Lê o artefato JSON padrão em mods/edits e aplica no binário. */
export async function importEntry(
  kind: EntryKind,
  id: string,
  version: GameVersionId,
): Promise<string[]> {
  return api().importEntry(kind, id, version);
}

/** Exporta o lote como UMA operação agrupada: um item {kind, ids} por ação
 * (ids vazio = tudo do kind), no formato pedido (json | strings | csv |
 * xlsx | po). O modal de progresso (steps por kind, barra própria por ação) é
 * dirigido pelos eventos do backend — o caller não mexe em toast.
 *
 * `complete: true` marca a seleção TOTAL do kind (checkbox da raiz na
 * árvore): o backend devolve o artefato bulk ÚNICO com prefixo `tree_`,
 * em vez de um arquivo por entrada. */
export async function exportArtifactBatch(
  items: { kind: EntryKind; ids: string[]; complete?: boolean }[],
  version: GameVersionId,
  langs: string[] = [],
  format: ExportFormat,
): Promise<string[]> {
  return api().exportArtifact(
    items.map((it) => ({
      kind: it.kind,
      ids: it.ids,
      complete: it.complete ?? false,
    })),
    version,
    langs,
    format,
  );
}

/**
 * Resumo do destino do export para o toast: os artefatos saem em mods/edits
 * (todos os kinds), então mostra a contagem + diretório (ou o caminho único).
 */
export function exportDestination(paths: string[]): string {
  const list = paths ?? [];
  if (list.length === 0) return '';
  if (list.length === 1) return list[0];
  const dir = list[0].replace(/[\\/][^\\/]*$/, '');
  return `${list.length} arquivos em ${dir}`;
}

/** Seletor nativo de arquivo (.json/.strings/.csv/.xlsx/.po). "" = cancelado. */
export function selectImportFile(): Promise<string> {
  return api().selectImportFile();
}

/** Valida o arquivo e devolve o resumo do modal (sem aplicar nada). */
export async function previewImport(
  path: string,
  version: GameVersionId,
): Promise<dto.ImportSummary> {
  return api().previewImport(path, version);
}

/** Aplica o arquivo validado; devolve quantos textos 'us' mudaram. */
export async function importFile(path: string, version: GameVersionId): Promise<number> {
  return api().importFile(path, version);
}

/**
 * Varre <TranslateLocation>/text e resume o lote SEM aplicar — alimenta o
 * diálogo de confirmação do Importar em lote.
 */
export async function previewImportTranslated(
  version: GameVersionId,
  kinds: EntryKind[],
): Promise<dto.ImportBatch> {
  return api().previewImportTranslated(version, kinds);
}

/**
 * Aplica o lote de <TranslateLocation>/text, arquivo a arquivo; devolve o
 * resumo agregado (arquivos, entradas, textos alterados e falhas).
 */
export async function importTranslated(
  version: GameVersionId,
  kinds: EntryKind[],
): Promise<dto.ImportBatch> {
  return api().importTranslated(version, kinds);
}
