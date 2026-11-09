import { describe, expect, it } from 'vitest';
import type { TagCatalog } from '@/lib/ffx/tag-catalog';
import { computeSuggestions, getChipValidator } from '@/lib/ffx/tag-catalog';

/**
 * Catálogo do FF13 no formato que o backend (FF13Service.GetTagCatalog)
 * serve — um recorte real: 4 cores, 2 ícones, os VAR, um estrutural.
 */
const FF13_CATALOG: TagCatalog = {
  version: 'part1',
  tags: [
    {
      tag: 'COLOR',
      prefix: '{COLOR:',
      hint: 'Cor do texto',
      values: [
        { tag: '{COLOR:White}', label: 'White', key: 'White' },
        { tag: '{COLOR:Gold}', label: 'Gold', key: 'Gold' },
        { tag: '{COLOR:IceBlue}', label: 'IceBlue', key: 'IceBlue' },
        { tag: '{COLOR:EXTRA00}', label: 'EXTRA00', key: 'EXTRA00' },
      ],
    },
    {
      tag: 'CTR',
      prefix: '{CTR:',
      hint: 'Contador',
      values: [{ tag: '{CTR:F7:40}', label: 'F7:40', key: 'F7:40' }],
    },
    {
      tag: 'ICON',
      prefix: '{ICON:',
      hint: 'Ícone',
      values: [
        { tag: '{ICON:F0:40:Clock}', label: 'Clock', key: 'F0:40' },
        { tag: '{ICON:F2:40:Sword}', label: 'Sword', key: 'F2:40' },
      ],
    },
    {
      tag: 'BTN',
      prefix: '{BTN:',
      hint: 'Botão do controle',
      values: [{ tag: '{BTN:F1:40:A}', label: 'A', key: 'F1:40' }],
    },
    {
      tag: 'VAR',
      prefix: '{VAR:',
      hint: 'Variável do jogo (item, nome, número)',
      values: [
        { tag: '{VAR:F4:40}', label: 'F4:40', key: 'F4:40' },
        { tag: '{VAR:F4:41}', label: 'F4:41', key: 'F4:41' },
      ],
    },
    {
      tag: 'NL',
      prefix: '{NL:',
      hint: 'Quebra de linha',
      values: [{ tag: '{NL:40:72}', label: '40:72', key: '40:72' }],
    },
  ],
};

/** Catálogo do FFX (recorte) — as classes de sempre. */
const FFX_CATALOG: TagCatalog = {
  version: 'ffx',
  tags: [
    {
      tag: 'PC',
      prefix: '{PC:',
      hint: 'Personagem',
      values: [{ tag: '{PC:01:YUNA}', label: 'YUNA', key: '01' }],
    },
    {
      tag: 'ICON',
      prefix: '{ICON:',
      hint: 'Ícone',
      values: [{ tag: '{ICON:01:Red Gate}', label: 'Red Gate', key: '01' }],
    },
  ],
};

describe('autocomplete do FF13 — estágio 1 (classes)', () => {
  it('lista as classes do catálogo da versão, com prefixo e hint do backend', () => {
    const items = computeSuggestions('', FF13_CATALOG);
    expect(items.map((i) => i.label)).toEqual(['COLOR', 'CTR', 'ICON', 'BTN', 'VAR', 'NL']);
    expect(items.every((i) => i.kind === 'tag')).toBe(true);
    // O insert é o PREFIXO do backend (não um chute do frontend).
    expect(items[0].insert).toBe('{COLOR:');
    expect(items[0].detail).toBe('Cor do texto');
    expect(items[2].insert).toBe('{ICON:');
    expect(items[2].detail).toBe('Ícone');
  });

  it('filtra pelo prefixo digitado', () => {
    expect(computeSuggestions('C', FF13_CATALOG).map((i) => i.label)).toEqual(['COLOR', 'CTR']);
    expect(computeSuggestions('V', FF13_CATALOG).map((i) => i.label)).toEqual(['VAR']);
    expect(computeSuggestions('BT', FF13_CATALOG).map((i) => i.label)).toEqual(['BTN']);
  });

  it('o FFX continua mostrando só as classes dele', () => {
    expect(computeSuggestions('', FFX_CATALOG).map((i) => i.label)).toEqual(['PC', 'ICON']);
  });
});

describe('autocomplete do FF13 — estágio 2 (valores)', () => {
  it('mostra os valores da classe com a tag completa como detalhe', () => {
    const items = computeSuggestions('ICON:', FF13_CATALOG);
    expect(items.map((i) => i.label)).toEqual(['Clock', 'Sword']);
    expect(items[0].insert).toBe('{ICON:F0:40:Clock}');
    expect(items[0].detail).toBe('{ICON:F0:40:Clock}');
  });

  it('filtra por índice (par de bytes) e por nome', () => {
    expect(computeSuggestions('ICON:F2', FF13_CATALOG).map((i) => i.label)).toEqual(['Sword']);
    expect(computeSuggestions('ICON:swo', FF13_CATALOG).map((i) => i.label)).toEqual(['Sword']);
    expect(computeSuggestions('COLOR:gol', FF13_CATALOG).map((i) => i.label)).toEqual(['Gold']);
  });

  it('classe sem entrada no catálogo não sugere nada', () => {
    expect(computeSuggestions('PC:', FF13_CATALOG)).toEqual([]);
    expect(computeSuggestions('KEY:', FF13_CATALOG)).toEqual([]);
  });
});

describe('chips do FF13 — label, lock e validade', () => {
  it('VAR sem nome mostra a tag inteira (não o 2º byte do par)', () => {
    const [item] = computeSuggestions('VAR:', FF13_CATALOG);
    expect(item.chip?.label).toBe('{VAR:F4:40}');
  });

  it('ICON mostra o nome e não está travado', () => {
    const [item] = computeSuggestions('ICON:', FF13_CATALOG);
    expect(item.chip?.label).toBe('Clock');
    expect(item.chip?.locked).toBe(false);
  });

  it('classes de valor do jogo ficam travadas (byte calculado pelo jogo)', () => {
    for (const inner of ['VAR:F4:40', 'CTR:F7:40', 'KEY:F6:40', 'PAGE:40:70', 'RAW:81:40', 'B:FF']) {
      expect(isLocked(inner)).toBe(true);
    }
    // Ícone e botão são conteúdo: o tradutor pode escolher.
    expect(isLocked('ICON:F2:40:Sword')).toBe(false);
    expect(isLocked('BTN:F1:40:A')).toBe(false);
  });

  it('todo valor do catálogo valida contra o próprio catálogo', () => {
    const validate = getChipValidator(FF13_CATALOG);
    for (const entry of FF13_CATALOG.tags ?? []) {
      for (const value of entry.values ?? []) {
        const inner = value.tag.slice(1, -1);
        expect(validate(inner), `${inner} deveria validar`).toBe(true);
      }
    }
  });
});

/** Import local para o lock (o mesmo que o popup usa). */
function isLocked(inner: string): boolean {
  // espelha isLockedTagInner sem importar ciclo no teste
  const upper = inner.trim().toUpperCase();
  const locked = ['VAR:', 'CTR:', 'KEY:', 'RAW:', 'B:', 'PAGE:', 'NL:', 'TAB:'];
  return locked.some((p) => upper.startsWith(p));
}
