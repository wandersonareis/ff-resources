import { Loader2 } from 'lucide-react';

/**
 * Placeholder de um arquivo AINDA EM CARGA no backend.
 *
 * A lista é servida de uma vez (o universo de ids sai da descoberta), mas o
 * conteúdo é lido arquivo por arquivo. Enquanto o arquivo clicado não chega,
 * abrir fica em espera - este painel mostra o progresso em vez de uma tabela
 * velha ou de um erro "não encontrado".
 */
export function EntryTableLoading({
  label,
  loaded,
  total,
}: {
  label?: string;
  loaded?: number;
  total?: number;
}) {
  const detail = total && total > 0 ? ` - ${loaded ?? 0} de ${total} arquivos` : '';
  return (
    <div
      className="mt-2 rounded-md border border-dashed px-3 py-6 text-center text-sm text-muted-foreground"
      role="status"
      aria-live="polite"
    >
      <Loader2 size={18} className="mr-2 inline animate-spin align-text-bottom" aria-hidden />
      <span>{label ? `Carregando ${label}${detail}…` : `Carregando arquivo${detail}…`}</span>
      <div className="mt-1 text-xs">O conteúdo chega por arquivo; a árvore já está completa.</div>
    </div>
  );
}
