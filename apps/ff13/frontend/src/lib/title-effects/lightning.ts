/**
 * Geometria dos raios do efeito de título — sem DOM, sem canvas.
 *
 * Cada raio é uma linha "quebrada": a cada segmento o ângulo sofre um
 * desvio aleatório e o comprimento varia. O tronco atravessa a caixa do
 * texto; os galhos nascem de pontos dele e morrem sozinhos (sem recursão),
 * o suficiente para o recorte nas letras parecer elétrico.
 */

/** Ponto em coordenadas CSS (px). */
export type Point = { x: number; y: number };

/** Caminho de um raio: tronco + galhos. */
export type BoltPath = {
  trunk: Point[];
  branches: Point[][];
};

export type BoltOptions = {
  /** Onde o raio começa. */
  start: Point;
  /** Ângulo inicial em radianos (0 = para a direita). */
  angle: number;
  /** Número de segmentos do tronco. */
  segments: number;
  /** Comprimento médio de cada segmento, em px. */
  segmentLength: number;
  /** Desvio angular máximo por segmento, em radianos. Padrão: 0.7. */
  spread?: number;
  /** Chance de nascer um galho a cada 4 pontos. Padrão: 0.15. */
  branchChance?: number;
  /** Fonte de aleatoriedade (injetável nos testes). Padrão: Math.random. */
  rng?: () => number;
};

/** Linha quebrada a partir de `start`, girando em torno do ângulo anterior. */
function jagged(
  start: Point,
  angle: number,
  segments: number,
  segmentLength: number,
  spread: number,
  rng: () => number,
): Point[] {
  const points: Point[] = [start];
  let x = start.x;
  let y = start.y;

  for (let i = 0; i < segments; i++) {
    // Desvio em torno do ângulo-base (não acumula): o tronco segue na
    // direção pedida em vez de vagar como um passeio aleatório.
    const current = angle + (rng() - 0.5) * spread;
    const step = segmentLength * (0.55 + rng() * 0.9);
    x += Math.cos(current) * step;
    y += Math.sin(current) * step;
    points.push({ x, y });
  }

  return points;
}

/** Gera o caminho completo de um raio (tronco + galhos). */
export function createBolt(options: BoltOptions): BoltPath {
  const rng = options.rng ?? Math.random;
  const spread = options.spread ?? 0.7;
  const branchChance = options.branchChance ?? 0.15;

  const trunk = jagged(
    options.start,
    options.angle,
    options.segments,
    options.segmentLength,
    spread,
    rng,
  );

  const branches: Point[][] = [];
  for (let i = 4; i < trunk.length - 1; i += 4) {
    if (rng() >= branchChance) continue;

    const direction = options.angle + (rng() < 0.5 ? -1 : 1) * (0.5 + rng() * 0.6);
    branches.push(
      jagged(
        trunk[i],
        direction,
        Math.max(2, Math.round(options.segments / 5)),
        options.segmentLength * 0.7,
        spread,
        rng,
      ),
    );
  }

  return { trunk, branches };
}
