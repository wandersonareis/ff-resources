'use client';

import { LightningText } from '@/components/lightning-text';
import type { GameVersionId } from '@/lib/ffx/game-version';
import { partUi } from '@/lib/parts';

const TITLE = 'FINAL FANTASY XIII Trilogy Resources Editor';
const TITLE_CLASS = 'text-xl font-malva-medium text-gradient';
/** Logo do título (part3): altura do `text-xl`, largura pelo aspect ratio. */
const TITLE_LOGO_CLASS = 'h-7 w-auto select-none';

/**
 * Título do header (fica à esquerda; o wrap do header cuida do mobile).
 *
 * `version` decide o efeito do título: cada part liga o seu em
 * `lib/parts/partN.ts` (`titleEffect`). O part3 troca o texto pelo logo
 * (`titleLogoSrc`) — os raios recortam na imagem em vez do texto.
 */
export function AppHeaderTitle({ version }: { version: GameVersionId }) {
  const { titleEffect, titleLogoSrc } = partUi(version);

  if (titleEffect === 'lightning') {
    return titleLogoSrc ? (
      <LightningText src={titleLogoSrc} alt={TITLE} className={TITLE_LOGO_CLASS} />
    ) : (
      <LightningText className={TITLE_CLASS}>{TITLE}</LightningText>
    );
  }

  return <span className={TITLE_CLASS}>{TITLE}</span>;
}
