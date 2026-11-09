// Tipos do log da UI (internal/platform/uilog no Go).
//
// O FF13 usa o MESMO formato de entrada do FFX: o painel de log pinta a cor
// pelo `level` e mostra o caminho na mesma linha do status.

/** Uma entrada do log. */
export interface Entry {
  /** Momento da entrada (Unix milliseconds). */
  time: number;
  /** Cor da linha: info | success | warn | error. */
  level: string;
  /** Operação ("Import", "Export", "Filelist"…). */
  action?: string;
  /** Texto da linha (ou o detalhe/motivo, nas linhas de arquivo). */
  message: string;
  /** Arquivo envolvido (binário, artefato importado). */
  path?: string;
  /** Resultado na MESMA linha do path ("ok", "falha"…). */
  status?: string;
}
