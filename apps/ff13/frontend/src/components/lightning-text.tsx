'use client';

import { useEffect, useRef } from 'react';
import { type BoltPath, createBolt } from '@/lib/title-effects/lightning';

/**
 * Texto com raios animados recortados NAS LETRAS (clip por glifos).
 *
 * Recebe apenas TEXTO PURO — não aceita componentes React/elementos. O
 * recorte é uma máscara desenhada no canvas com o mesmo `font` do texto, não
 * com o DOM: ícones, markup, quebras de linha ou fontes misturadas não têm
 * como ser mascarados. Para animar um componente, renderize-o fora e use
 * `LightningText` só no trecho de texto.
 *
 * Cores aceitam qualquer formato CSS (`#4ade80`, `rgb(74 222 128)`,
 * `green`, `hsl(...)`).
 *
 * ```tsx
 * <LightningText className="text-xl text-gradient" color="#4ade80">
 *   FINAL FANTASY XIII
 * </LightningText>
 * ```
 */
export type LightningTextProps = {
  /** Texto puro, em uma linha — é o que recebe o recorte dos raios. */
  children: string;
  /** Cor do núcleo do raio. Padrão: `#4ade80` (verde). */
  color?: string;
  /** Cor do brilho (glow). Padrão: `#10b981` (esmeralda). */
  glowColor?: string;
  /** Liga/desliga a animação sem desmontar. Padrão: `true`. */
  active?: boolean;
  /** Intervalo mínimo entre descargas, em ms. Padrão: `700`. */
  minInterval?: number;
  /** Intervalo máximo entre descargas, em ms. Padrão: `2400`. */
  maxInterval?: number;
  /** Classes do texto (fonte, tamanho, degradê). */
  className?: string;
};

type Strike = {
  path: BoltPath;
  /** performance.now() em que a descarga começa. */
  born: number;
  /** Duração total do traço, em ms. */
  life: number;
  /** Espessura em px. */
  width: number;
};

export function LightningText({
  children,
  color = '#B1C8FC',
  glowColor = '#4ade80',
  active = true,
  minInterval = 700,
  maxInterval = 2400,
  className,
}: LightningTextProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const textEl = textRef.current;
    const canvas = canvasRef.current;
    if (!textEl || !canvas || !active) return;
    // Sem movimento: o texto continua normal, sem raios.
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let disposed = false;
    let raf = 0;
    let timer: number | undefined;
    let strikes: Strike[] = [];
    let cssWidth = 0;
    let cssHeight = 0;
    let font = '';
    let letterSpacing = '0px';

    /** Mede o texto e redimensiona o canvas (DPR incluso) quando muda. */
    const measure = () => {
      const box = textEl.getBoundingClientRect();
      const width = Math.max(1, Math.round(box.width));
      const height = Math.max(1, Math.round(box.height));

      // O `font`/espaçamento podem mudar sem mudar o tamanho (ex.: a fonte
      // real terminar de carregar): sempre relê antes do early-return.
      const style = window.getComputedStyle(textEl);
      font = `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
      letterSpacing = style.letterSpacing === 'normal' ? '0px' : style.letterSpacing;

      if (width === cssWidth && height === cssHeight) return;

      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      cssWidth = width;
      cssHeight = height;
      canvas.width = Math.round(width * dpr);
      canvas.height = Math.round(height * dpr);
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      strikes = [];
      ctx.clearRect(0, 0, width, height);
    };

    /** Aplica o texto como máscara alfa do que já foi desenhado no canvas. */
    const clipToText = () => {
      ctx.globalCompositeOperation = 'destination-in';
      ctx.globalAlpha = 1;
      ctx.shadowBlur = 0;
      ctx.shadowColor = 'transparent';
      ctx.font = font;
      const spaced = ctx as CanvasRenderingContext2D & { letterSpacing?: string };
      if (spaced.letterSpacing !== undefined) spaced.letterSpacing = letterSpacing;
      ctx.textAlign = 'left';
      ctx.textBaseline = 'middle';
      ctx.fillStyle = '#fff';
      ctx.fillText(children, 0, cssHeight / 2);
      ctx.globalCompositeOperation = 'source-over';
    };

    const strokePath = (points: BoltPath['trunk'], width: number, alpha: number) => {
      ctx.globalAlpha = alpha;
      ctx.lineWidth = width;
      ctx.strokeStyle = color;
      ctx.shadowBlur = 9;
      ctx.shadowColor = glowColor;
      ctx.lineCap = 'round';
      ctx.lineJoin = 'round';
      ctx.beginPath();
      ctx.moveTo(points[0].x, points[0].y);
      for (let i = 1; i < points.length; i++) ctx.lineTo(points[i].x, points[i].y);
      ctx.stroke();
    };

    const frame = (now: number) => {
      raf = 0;
      ctx.clearRect(0, 0, cssWidth, cssHeight);
      strikes = strikes.filter((strike) => now - strike.born < strike.life);

      if (strikes.length > 0) {
        for (const strike of strikes) {
          if (now < strike.born) continue;
          const progress = (now - strike.born) / strike.life;
          // Segura o raio cheio na primeira metade e desvanece na segunda.
          const alpha = progress < 0.45 ? 1 : 1 - (progress - 0.45) / 0.55;
          // Halo largo + núcleo fino, ambos recortados nas letras.
          strokePath(strike.path.trunk, strike.width * 3.2, alpha * 0.3);
          strokePath(strike.path.trunk, strike.width, alpha);
          for (const branch of strike.path.branches) {
            strokePath(branch, strike.width * 0.6, alpha * 0.8);
          }
        }
        clipToText();
        raf = requestAnimationFrame(frame);
      }
    };

    /** Uma descarga = 1–2 raios atravessando a caixa do texto. */
    const strike = () => {
      if (cssWidth < 4 || cssHeight < 4) return;

      const now = performance.now();
      const count = 1 + (Math.random() < 0.35 ? 1 : 0) + (Math.random() < 0.08 ? 1 : 0);

      for (let i = 0; i < count; i++) {
        const y = cssHeight * (0.3 + Math.random() * 0.4);
        const path = createBolt({
          start: { x: -6, y },
          angle: (Math.random() - 0.5) * 0.3,
          segments: Math.max(8, Math.round(cssWidth / 9)),
          segmentLength: 7 + Math.random() * 4,
          spread: 0.9,
          branchChance: 0.15,
        });
        strikes.push({
          path,
          born: now + i * 70,
          life: 650 + Math.random() * 350,
          width: 1.6 + Math.random() * 1.0,
        });
      }

      if (!raf) raf = requestAnimationFrame(frame);
    };

    const schedule = () => {
      const max = Math.max(minInterval, maxInterval);
      const delay = minInterval + Math.random() * (max - minInterval);
      timer = window.setTimeout(() => {
        if (disposed) return;
        strike();
        schedule();
      }, delay);
    };

    measure();
    strike();
    schedule();

    const observer = new ResizeObserver(measure);
    observer.observe(textEl);
    void document.fonts?.ready.then(() => {
      if (!disposed) measure();
    });

    return () => {
      disposed = true;
      if (timer !== undefined) window.clearTimeout(timer);
      if (raf) cancelAnimationFrame(raf);
      observer.disconnect();
      strikes = [];
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    };
  }, [active, children, color, glowColor, minInterval, maxInterval]);

  return (
    <span className="relative inline-block">
      <span ref={textRef} className={className}>
        {children}
      </span>
      <canvas ref={canvasRef} aria-hidden className="pointer-events-none absolute left-0 top-0" />
    </span>
  );
}
