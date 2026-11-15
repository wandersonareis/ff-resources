'use client';

import { AppHeaderActions } from '@/components/app-header-actions';
import { AppHeaderTitle } from '@/components/app-header-title';
import type { GameVersionId } from '@/lib/ffx/game-version';

/**
 * Barra superior (título + tema + configurações + formato e idiomas de
 * exportação). Título e botões são componentes separados, num container que
 * permite wrap: no mobile os botões descem de linha e continuam à direita.
 * Só precisa do callback de abrir Configurações: formato e idiomas vêm das
 * stores módulo (`exportSelection` / `exportLangs`), então nada disso
 * atravessa props.
 */
export function AppHeader({
  version,
  onOpenConfig,
}: {
  version: GameVersionId;
  onOpenConfig: () => void;
}) {
  return (
    <header className="sticky top-0 z-10 flex flex-wrap items-center gap-x-2 gap-y-1 bg-chrome px-4 py-2 border-b">
      <AppHeaderTitle version={version} />
      <AppHeaderActions onOpenConfig={onOpenConfig} />
    </header>
  );
}
