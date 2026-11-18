'use client';

import { Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { SHORTCUTS, shortcutTip } from '@/lib/ffx/shortcuts';
import { ExportFormatMenu } from './export-format-menu';
import { ExportLangsMenu } from './export-langs-menu';
import { LogWindowToggle } from './log-window-toggle';
import { ThemeToggle } from './theme-toggle';

/**
 * Botões do header. No mobile a barra inverte (row-reverse + justify-start):
 * o Tema fica na extremidade direita e os demais botões seguem para a esquerda,
 * inclusive quando o header quebra a linha. A partir de `sm` volta à ordem e
 * ao alinhamento normais (`justify-end`), mantendo o grupo no canto direito.
 */
export function AppHeaderActions({ onOpenConfig }: { onOpenConfig: () => void }) {
  return (
    <div className="flex flex-row-reverse flex-wrap items-center justify-start gap-0.5 sm:flex-row sm:justify-end">
      <ThemeToggle />
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" onClick={onOpenConfig} aria-label="Configurações">
            <Settings size={22} />
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {shortcutTip(SHORTCUTS.config.label, SHORTCUTS.config.keys)}
        </TooltipContent>
      </Tooltip>
      <ExportFormatMenu />
      <ExportLangsMenu />
      <LogWindowToggle />
    </div>
  );
}
