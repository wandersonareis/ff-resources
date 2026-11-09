// Tipos do backend (DTOs Go) do app FF13.
//
// Os tipos de TEXTO/IMAGEM vivem no pacote compartilhado (frontend/shared/
// src/dto.ts) — é de lá que a UI FFX e a FF13 tiram as formas. Aqui ficam
// só os DTOs que o pacote compartilhado ainda não expõe (importação e
// navegador .vbf), com os MESMOS nomes snake_case do Go.
//
// O caminho é RELATIVO de propósito: o tsconfig do app_ff13 mapeia poucos
// alias do pacote compartilhado, e o import relativo resolve igual no Vite
// e no tsc.
export * from '../../../../../../../frontend/shared/src/dto';

/** Resumo de uma importação de UM arquivo (modal de confirmação). */
export interface ImportSummary {
  path: string;
  format: string;
  kind: string;
  version: string;
  languages: string[];
  entry_count: number;
  total_indices: number;
  changed_texts: number;
  saves_binary: boolean;
  entries: ImportEntryInfo[];
  usages: ImportUsage[];
  warnings: string[];
  skipped: ImportRowIssue[];
  errors: string[];
}

/** Comparação entre uma entrada do artefato importado e a store. */
export interface ImportEntryInfo {
  id: string;
  key: string;
  index_count: number;
  store_index_count: number;
  changed_texts: number;
  skipped_rows: number;
  error?: string;
}

/** Row do artefato que ficou fora da importação. */
export interface ImportRowIssue {
  id: string;
  index: number;
  hash: string;
  text: string;
  reason: string;
}

/** Uso pós-rebuild do limite uint16 de um binário. */
export interface ImportUsage {
  id: string;
  kind: string;
  used: number;
  limit: number;
  over: boolean;
}

/** Importação em LOTE (vários arquivos). */
export interface ImportBatch {
  root: string;
  kinds: string[];
  files: string[];
  entry_count: number;
  changed_texts: number;
  errors: string[];
  warnings: string[];
}

/** Raiz de árvore do navegador de container .vbf. */
export interface VbfRoot {
  name: string;
  path: string;
  version: string;
  size: number;
  entries: number;
}

/** Filho imediato de um diretório de container .vbf. */
export interface VbfNode {
  name: string;
  path: string;
  isDir: boolean;
  size?: number;
  kind?: string;
  id?: string;
  version?: string;
  macro?: boolean;
  image?: boolean;
}

/** Resumo antes de extrair a seleção do .vbf. */
export interface VbfExtractPreview {
  files: number;
  bytes: number;
  existing: number;
  destRoot: string;
}
