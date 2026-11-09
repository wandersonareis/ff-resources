// Tipos dos SERVIÇOS do backend (o FF13 usa os mesmos do FFX: a busca de
// texto e a poda de linhas órfãs são as mesmas estruturas).
//
// O FF13 ainda não expõe busca/prune no Go — os tipos ficam aqui para a UI
// compartilhada compilar.

/** Uma linha casada pela busca. */
export interface TextSearchMatch {
  index: number;
  name?: string;
  data: boolean;
  mods: boolean;
  snippetBefore?: string;
  snippetHit?: string;
  snippetAfter?: string;
}

/** Resultado da busca em UM arquivo. */
export interface TextSearchResult {
  kind: string;
  id: string;
  rows: TextSearchMatch[];
}

/** Linha que sobrou sem par depois da poda de órfãs. */
export interface PrunedRowView {
  /** Chave canônica da entrada (ex.: ffx2/menu/macrodic.dcp). */
  file: string;
  /** Bloco (chunk) quando o formato tem; ausente = é o índice. */
  block?: number | null;
  /** Índice da linha dentro do arquivo/bloco. */
  index: number;
  /** Conteúdo da linha (o texto em branco que ficou). */
  text?: string;
}
