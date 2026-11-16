'use client';

import { cn } from 'cn';
import { PanelLeft, Save, Search } from 'lucide-react';
import { Badge } from '../../ui/badge';
import { Button } from '../../ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '../../ui/tooltip';
import type { EntryProgress } from '../../../app/entry-progress';
import { SHORTCUTS, shortcutTip } from '../../../app/shortcuts';

/**
 * Barra de título da tabela: tipo + entrada + badge de progresso, e o
 * botão Salvar (e o de buscar neste arquivo). Fica FORA do rolante (o
 * painel é flex-col: barra em cima, tabela rolando embaixo), então não
 * precisa de sticky — fixa por construção, independente do tamanho do texto.
 *
 * Responsivo: no estreito o badge de progresso some (a barra de status já o
 * mostra), o rótulo da entrada trunca e o toggle da sidebar aparece quando o
 * app passa `onToggleSidebar` (sidebar em gaveta).
 */
export function TableBar({
  kindLabel,
  entryLabel,
  progress,
  hasDirty,
  saving,
  onSave,
  onOpenFind,
  onToggleSidebar,
}: {
  kindLabel: string;
  entryLabel?: string | null;
  progress?: EntryProgress | null;
  hasDirty: boolean;
  saving: boolean;
  onSave: () => void;
  /** Abre a busca & substituição local (ausente em kinds sem rows, p/ images). */
  onOpenFind?: () => void;
  /** Abre a gaveta da sidebar (só no layout estreito). */
  onToggleSidebar?: () => void;
}) {
  return (
    <div className="flex shrink-0 items-center justify-between gap-2 sm:gap-4">
      <h3 className="flex min-w-0 items-center gap-2 text-base text-primary font-malva-medium md:text-lg">
        <span className="shrink-0">{kindLabel}</span>
        {/* O span volta ao Malva padrão (Regular, família base), só desanimado. */}
        {entryLabel ? <span className="truncate opacity-70"> · {entryLabel}</span> : null}
        {progress ? (
          <Badge
            variant="outline"
            className={cn(
              'max-sm:hidden',
              progress.translated >= progress.total && progress.total > 0
                ? 'text-success border-success/40'
                : 'text-muted-foreground',
            )}
          >
            {progress.translated}/{progress.total} linhas traduzidas ({progress.pct}%)
          </Badge>
        ) : null}
      </h3>
      <div className="flex shrink-0 items-center gap-1">
        {onToggleSidebar ? (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Abrir conteúdo"
            onClick={onToggleSidebar}
          >
            <PanelLeft size={18} />
          </Button>
        ) : null}
        {onOpenFind ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Buscar neste arquivo"
                onClick={onOpenFind}
              >
                <Search size={18} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {shortcutTip(SHORTCUTS.findInFile.label, SHORTCUTS.findInFile.keys)}
            </TooltipContent>
          </Tooltip>
        ) : null}
        <Tooltip>
          <TooltipTrigger asChild>
            {/* O botão desabilitado não dispara pointer events, então o
                trigger é o span envoltório (padrão Radix p/ disabled). */}
            <span className="inline-flex">
              <Button disabled={!hasDirty || saving} onClick={onSave}>
                <Save size={18} />
                {saving ? 'Salvando…' : 'Salvar'}
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>{shortcutTip(SHORTCUTS.save.label, SHORTCUTS.save.keys)}</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
