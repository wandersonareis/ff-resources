'use client';

import type { GameVersionId } from '@/lib/ffx/game-version';
import { AppHeaderActions } from './actions';
import { AppHeaderTitle } from './title';

/**
 * Barra superior (título + tema + configurações + formato e idiomas de
 * exportação). Título e botões são componentes separados, num container que
 * permite wrap: no mobile os botões descem de linha e continuam à direita.
 * Só precisa do callback de abrir Configurações: formato e idiomas vêm das
 * stores módulo (`exportSelection` / `exportLangs`), então nada disso
 * atravessa props.
 *
 * ALTURA FIXA a partir de `lg`: o logo de cada part tem alturas diferentes e
 * antes empurrava o header (49px no part3, 100px no part1/2) ao trocar de
 * aba. Com `lg:h-16` o header não muda mais; o logo que passar da altura é
 * recortado pelo `overflow-hidden` (o NOME fica sempre dentro — ver
 * `LOGO_HEIGHT` em title.tsx). Abaixo de `lg` o header segue com wrap e a
 * altura vem do slot de 40px do título.
 */
export function AppHeader({
  version,
  onOpenConfig,
}: {
  version: GameVersionId;
  onOpenConfig: () => void;
}) {
  return (
    <header className="sticky top-0 z-10 flex flex-wrap items-center justify-between gap-x-2 gap-y-1 overflow-hidden bg-chrome px-4 py-2 border-b lg:h-16 lg:flex-nowrap lg:gap-y-0 lg:py-0">
      <AppHeaderTitle version={version} />
      <AppHeaderActions onOpenConfig={onOpenConfig} />
    </header>
  );
}
