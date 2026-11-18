'use client';

import { LightningText } from '@/components/lightning-text';
import type { GameVersionId } from '@/lib/ffx/game-version';
import { partUi } from '@/lib/parts';

const TITLE = 'FINAL FANTASY XIII Trilogy Resources Editor';
const TITLE_CLASS = 'text-base sm:text-lg font-malva-medium text-gradient';

/** Altura do slot do título no header (mobile/tablet): o header não pula. */
const LOGO_SLOT_CLASS = 'relative flex h-10 shrink-0 items-center lg:h-full';
/** Dimensões comuns: altura vem de `LOGO_HEIGHT`, largura pelo aspect ratio. */
const LOGO_BASE_CLASS = 'w-auto select-none';
/**
 * Glow pulsante do logo, no tom tirado da própria arte de cada part:
 * - part1 é jade/menta (amostras `#A0C0B0`, `#80A090`, `#D0F0D0`);
 * - part2 é vermelho→rosa.
 * O part3 não entra aqui: os raios do `LightningText` fazem o efeito.
 */
const LOGO_GLOW: Partial<Record<GameVersionId, string>> = {
  part1: 'part1-logo-glow',
  part2: 'part2-logo-glow',
};

/**
 * Altura do logo por part. O header tem altura FIXA a partir de `lg`
 * (`h-16`) — antes cada logo empurrava a barra (49px no part3, 100px no
 * part1/2) e trocar de aba redimensionava tudo.
 *
 * O logo pode ser mais alto que o header: ele é recortado pelo
 * `overflow-hidden` da barra, mas o NOME (a faixa de baixo da arte) fica
 * sempre dentro dos 64px. Alturas medidas na arte (faixa do nome):
 * - part1  220×102 — linhas 44..83 (0,814): 70px ⇒ nome termina em ~54px;
 * - part2  512×169 — linhas 70..166 (0,982): 60px ⇒ nome termina em ~61px;
 * - part3 1024×128 — duas linhas até a 98 (0,766) e MUITO larga (8:1):
 *   36px ⇒ nome termina em ~42px e 288px de largura (com 64px de altura a
 *   arte virava 512px e dominava a barra). No mobile todos usam o slot de
 *   40px, então a barra mede o mesmo nas 3 abas.
 */
const LOGO_HEIGHT: Partial<Record<GameVersionId, string>> = {
  part1: 'h-10 lg:h-[70px]',
  part2: 'h-10 lg:h-[60px]',
  part3: 'h-8 lg:h-9',
};

/**
 * Título do header (fica à esquerda; o wrap do header cuida do mobile).
 *
 * `version` decide o título via `lib/parts/partN.ts`: quando a part define
 * `titleLogoSrc`, o texto é substituído pelo AVIF. Com `titleEffect: 'lightning'`
 * (part3) os raios recortam na imagem; sem efeito (part1/part2) o logo aparece
 * com glow pulsante.
 */
export function AppHeaderTitle({ version }: { version: GameVersionId }) {
  const { titleEffect, titleLogoSrc } = partUi(version);
  const logoClass = `${LOGO_BASE_CLASS} ${LOGO_HEIGHT[version] ?? 'h-10 lg:h-16'}`;
  const glow = LOGO_GLOW[version];

  if (titleLogoSrc) {
    return (
      <span className={LOGO_SLOT_CLASS}>
        {titleEffect === 'lightning' ? (
          <LightningText src={titleLogoSrc} alt={TITLE} className={logoClass} />
        ) : (
          // biome-ignore lint/performance/noImgElement: app Vite/Wails sem next/image — o AVIF vem de public/ já otimizado.
          <img
            src={titleLogoSrc}
            alt={TITLE}
            draggable={false}
            className={glow ? `${logoClass} ${glow}` : logoClass}
          />
        )}
      </span>
    );
  }

  if (titleEffect === 'lightning') {
    return <LightningText className={TITLE_CLASS}>{TITLE}</LightningText>;
  }

  return <span className={TITLE_CLASS}>{TITLE}</span>;
}
