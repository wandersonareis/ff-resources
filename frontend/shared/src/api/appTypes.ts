// Tipos de apoio da camada de UI (espelham os modelos de serviço do FFX, mas
// estruturais para os dois apps).

/** Linha que o save excluiria por não existir no original. */
export interface PrunedRowView {
  file: string;
  block?: number | null;
  index: number;
  name: string;
  text?: string;
  langs?: string[];
  version: string;
  kind: string;
}

/** Ocorrência da busca global dentro de uma row. */
export interface TextSearchMatch {
  index: number;
  name?: string;
  data: boolean;
  mods: boolean;
  snippetBefore?: string;
  snippetHit?: string;
  snippetAfter?: string;
}

/** Resultado da busca global por arquivo. */
export interface TextSearchResult {
  kind: string;
  id: string;
  rows: TextSearchMatch[];
}

/** Resposta da busca global. */
export interface TextSearchResponse {
  results: TextSearchResult[];
  totalRows: number;
  totalFiles: number;
  truncated: boolean;
}
