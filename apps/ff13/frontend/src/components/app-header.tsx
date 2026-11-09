'use client';

import { Settings } from 'lucide-react';
import { ExportFormatMenu } from '@/components/export-format-menu';
import { ExportLangsMenu } from '@/components/export-langs-menu';
import { LightningText } from '@/components/lightning-text';
import { LogWindowToggle } from '@/components/log-window-toggle';
import { ThemeToggle } from '@/components/theme-toggle';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { GameVersionId } from '@/lib/ffx/game-version';
import { SHORTCUTS, shortcutTip } from '@/lib/ffx/shortcuts';
import { partUi } from '@/lib/parts';

const TITLE = 'FINAL FANTASY XIII Trilogy Resources Editor';
const TITLE_CLASS = 'text-xl font-malva-medium text-gradient';

/**
 * Barra superior (título + tema + configurações + formato e idiomas de
 * exportação). Só precisa do callback de abrir Configurações: formato e
 * idiomas vêm das stores módulo (`exportSelection` / `exportLangs`),
 * então nada disso atravessa props.
 *
 * `version` decide o efeito do título: cada part liga o seu em
 * `lib/parts/partN.ts` (`titleEffect`).
 */
export function AppHeader({
  version,
  onOpenConfig,
}: {
  version: GameVersionId;
  onOpenConfig: () => void;
}) {
  const effect = partUi(version).titleEffect;

  return (
    <header className="sticky top-0 z-10 flex items-center gap-0.5 bg-chrome px-4 py-2 border-b">
      {effect === 'lightning' ? (
        <LightningText className={TITLE_CLASS}>{TITLE}</LightningText>
      ) : (
        <span className={TITLE_CLASS}>{TITLE}</span>
      )}
      <span className="flex-1" />
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
    </header>
  );
}
