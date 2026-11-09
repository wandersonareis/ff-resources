import { describe, expect, it } from 'vitest';
import { createBolt } from './lightning';

/** RNG fixo: deixa a geometria determinística nos testes. */
const constant = (value: number) => () => value;

describe('createBolt', () => {
  it('segue do start com o número de segmentos pedido', () => {
    const bolt = createBolt({
      start: { x: 0, y: 10 },
      angle: 0,
      segments: 6,
      segmentLength: 10,
      spread: 0,
      branchChance: 0,
      rng: constant(0.5),
    });

    expect(bolt.trunk).toHaveLength(7);
    expect(bolt.trunk[0]).toEqual({ x: 0, y: 10 });
    for (let i = 1; i < bolt.trunk.length; i++) {
      expect(bolt.trunk[i].x).toBeCloseTo(bolt.trunk[i - 1].x + 10);
      expect(bolt.trunk[i].y).toBeCloseTo(10);
    }
    expect(bolt.branches).toHaveLength(0);
  });

  it('gera galhos a partir do tronco quando a chance manda', () => {
    const bolt = createBolt({
      start: { x: 0, y: 0 },
      angle: 0,
      segments: 8,
      segmentLength: 10,
      spread: 0.7,
      branchChance: 1,
      rng: constant(0),
    });

    expect(bolt.branches).toHaveLength(1);
    for (const branch of bolt.branches) {
      expect(branch.length).toBeGreaterThanOrEqual(2);
      expect(branch.every((point) => Number.isFinite(point.x) && Number.isFinite(point.y))).toBe(
        true,
      );
    }
  });
});
