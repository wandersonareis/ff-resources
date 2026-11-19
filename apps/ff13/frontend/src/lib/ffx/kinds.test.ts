import { describe, expect, it } from 'vitest';
import { compareKindLabels, entryKindsAlpha, entryKindsFor } from '@/lib/ffx/kinds';

describe('ordem de exibição dos kinds', () => {
  it('part1/part2: alfabética pelo rótulo (Eventos primeiro)', () => {
    expect(entryKindsAlpha('part1')).toEqual(['event', 'ac', 'resident', 'zone']);
    expect(entryKindsAlpha('part2')).toEqual(['event', 'ac', 'resident', 'zone']);
  });

  it('part3: Cenários entra antes de Eventos (sem `ac`)', () => {
    expect(entryKindsAlpha('part3')).toEqual(['scenario', 'event', 'resident', 'zone']);
  });

  it('não altera a ordem canônica usada pela árvore', () => {
    expect(entryKindsFor('part1')).toEqual(['ac', 'resident', 'zone', 'event']);
  });

  it('compara pelo rótulo, não pela chave do kind', () => {
    expect(compareKindLabels('ac', 'event')).toBeGreaterThan(0);
  });
});
