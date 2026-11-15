'use client';

import { Settings } from 'lucide-react';
import { ExportFormatMenu } from '@/components/export-format-menu';
import { ExportLangsMenu } from '@/components/export-langs-menu';
import { LogWindowToggle } from '@/components/log-window-toggle';
import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { SHORTCUTS, shortcutTip } from '@/lib/ffx/shortcuts';

/**
 * Botões do header. `ml-auto`/`justify-end` mantêm o grupo no canto direito
 * no desktop e também quando o header quebra a linha no mobile.
 */
export function AppHeaderActions({ onOpenConfig }: { onOpenConfig: () => void }) {
  return (
    <div className="ml-auto flex flex-wrap items-center justify-end gap-0.5">
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
