'use client';

import { useEffect, useRef } from 'react';
import { type BoltPath, createBolt } from '@/lib/title-effects/lightning';

/**
 * Texto ou imagem com raios animados recortados NO CONTEÚDO (clip por máscara).
 *
 * DOIS MODOS:
 * - TEXTO: recebe apenas TEXTO PURO — não aceita componentes React/elementos.
 *   O recorte é uma máscara desenhada no canvas com o mesmo `font` do texto,
 *   não com o DOM: ícones, markup, quebras de linha ou fontes misturadas não
 *   têm como ser mascarados. Para animar um componente, renderize-o fora e use
 *   `LightningText` só no trecho de texto.
 * - IMAGEM: com `src`, o recorte é a imagem renderizada no canvas
 *   (`drawImage`) — os raios ficam recortados nos glifos do logo. `alt`
 *   descreve a imagem; `className` controla o tamanho (`h-* w-auto`).
 *
 * Cores aceitam qualquer formato CSS (`#4ade80`, `rgb(74 222 128)`,
 * `green`, `hsl(...)`). Padrão: núcleo azul escuro com brilho azul claro
 * (`#2563eb` + `#93c5fd`).
 *
 * ```tsx
 * <LightningText className="text-xl text-gradient" color="#4ade80">
 *   FINAL FANTASY XIII
 * </LightningText>
 * <LightningText src="/part3-logo.avif" alt="LR:FF13" className="h-7 w-auto" />
 * ```
 */
export type LightningTextProps = {
  /** Cor do núcleo do raio (azul escuro). Padrão: `#2563eb`. */
  color?: string;
  /** Cor do brilho (glow azul claro). Padrão: `#93c5fd`. */
  glowColor?: string;
  /** Liga/desliga a animação sem desmontar. Padrão: `true`. */
  active?: boolean;
  /** Intervalo mínimo entre descargas, em ms. Padrão: `700`. */
  minInterval?: number;
  /** Intervalo máximo entre descargas, em ms. Padrão: `2400`. */
  maxInterval?: number;
  /** Classes do texto (fonte, tamanho, degradê) ou do logo (`h-* w-auto`). */
  className?: string;
} & (
  | { /** Texto puro, em uma linha — é o que recebe o recorte dos raios. */
    children: string;
    src?: undefined;
    alt?: undefined }
  | { /** Fonte da imagem que recebe o recorte dos raios (em vez do texto). */
    children?: undefined;
    src: string;
    /** Texto alternativo do `<img>` (modo imagem). */
    alt?: string }
);

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
  src,
  alt,
  color = '#2563eb',
  glowColor = '#93c5fd',
  active = true,
  minInterval = 700,
  maxInterval = 2400,
  className,
}: LightningTextProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const textEl = textRef.current;
    const imgEl = imgRef.current;
    const canvas = canvasRef.current;
    // Um dos dois existe: o modo é decidido por qual elemento renderizou.
    const boxEl: HTMLElement | null = textEl ?? imgEl;
    if (!boxEl || !canvas || !active) return;
    // Sem movimento: o conteúdo continua normal, sem raios.
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

    /** Mede o conteúdo e redimensiona o canvas (DPR incluso) quando muda. */
    const measure = () => {
      const box = boxEl.getBoundingClientRect();
      const width = Math.max(1, Math.round(box.width));
      const height = Math.max(1, Math.round(box.height));

      // O `font`/espaçamento podem mudar sem mudar o tamanho (ex.: a fonte
      // real terminar de carregar): sempre relê antes do early-return.
      // Só importam no modo texto — a máscara de imagem é o drawImage dela.
      if (textEl) {
        const style = window.getComputedStyle(textEl);
        font = `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
        letterSpacing = style.letterSpacing === 'normal' ? '0px' : style.letterSpacing;
      }

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
      ctx.fillText(children ?? '', 0, cssHeight / 2);
      ctx.globalCompositeOperation = 'source-over';
    };

    /**
     * Aplica a IMAGEM como máscara alfa do que já foi desenhado no canvas:
     * desenha o logo na caixa medida e o `destination-in` recorta os raios
     * nos glifos dele (o alfa do AVIF faz o papel do fillText do modo texto).
     */
    const clipToImage = () => {
      // Imagem quebrada ou ainda decodificando: máscara vazia, sem raios —
      // o listener de `load` re-medindo traz o recorte quando ela chegar.
      if (!imgEl?.complete || imgEl.naturalWidth === 0) return;
      ctx.globalCompositeOperation = 'destination-in';
      ctx.globalAlpha = 1;
      ctx.shadowBlur = 0;
      ctx.shadowColor = 'transparent';
      ctx.drawImage(imgEl, 0, 0, cssWidth, cssHeight);
      ctx.globalCompositeOperation = 'source-over';
    };

    /** A máscara acompanha o modo: `src` define imagem, ausência = texto. */
    const applyMask = () => {
      if (src === undefined) clipToText();
      else clipToImage();
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
        applyMask();
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
    observer.observe(boxEl);
    // No modo imagem, o carregamento muda a caixa (w-auto) e habilita a
    // máscara: re-medir quando o logo chegar.
    imgEl?.addEventListener('load', measure);
    void document.fonts?.ready.then(() => {
      if (!disposed) measure();
    });

    return () => {
      disposed = true;
      if (timer !== undefined) window.clearTimeout(timer);
      if (raf) cancelAnimationFrame(raf);
      observer.disconnect();
      imgEl?.removeEventListener('load', measure);
      strikes = [];
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    };
  }, [active, children, src, color, glowColor, minInterval, maxInterval]);

  return (
    <span className="relative inline-block">
      {src === undefined ? (
        <span ref={textRef} className={className}>
          {children}
        </span>
      ):(
        // biome-ignore lint/performance/noImgElement: app Vite/Wails sem next/image — o logo sai de public/ já otimizado no build.
        <img
          ref={imgRef}
          src={src}
          alt={alt ?? ''}
          draggable={false}
          className={className}
        />
      )}
      <canvas ref={canvasRef} aria-hidden className="pointer-events-none absolute left-0 top-0" />
    </span>
  );
}
