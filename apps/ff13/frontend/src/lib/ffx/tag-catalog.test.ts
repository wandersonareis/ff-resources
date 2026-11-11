import { describe, expect, it } from 'vitest';
import type { TagCatalog } from '@/lib/ffx/tag-catalog';
import { computeSuggestions, getChipValidator, paletteFromCatalog } from '@/lib/ffx/tag-catalog';

/**
 * Catálogo do FF13 no formato que o backend (FF13Service.GetTagCatalog)
 * serve — um recorte real: 4 cores, 2 ícones, os VAR, um estrutural.
 */
const FF13_CATALOG: TagCatalog = {
  version: 'part1',
  lang: 'us',
  tags: [
    {
      tag: 'CLR',
      prefix: '{CLR:',
      hint: 'Cor do texto',
      values: [
        {
          tag: '{CLR:F9:40:White}',
          label: 'White',
          key: 'F9:40',
          hex: '#ffffff',
          reset: true,
        },
        { tag: '{CLR:F9:42:Gold}', label: 'Gold', key: 'F9:42', hex: '#e8c15a' },
        { tag: '{CLR:F9:41:IceBlue}', label: 'IceBlue', key: 'F9:41', hex: '#a5d8e6' },
        {
          tag: '{CLR:F9:32:Ex00}',
          label: 'Ex00',
          key: 'F9:32',
          hex: '#111111',
          group: 'ex',
          semiTransparent: true,
        },
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
    {
      tag: 'SPACE',
      prefix: '{SPACE:',
      hint: 'Espaço sem quebra (U+00A0)',
      values: [{ tag: '{SPACE:85:60}', label: 'NBSP', key: '85:60' }],
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
    expect(items.map((i) => i.label)).toEqual([
      'CLR',
      'CTR',
      'ICON',
      'BTN',
      'VAR',
      'NL',
      'SPACE',
    ]);
    expect(items.every((i) => i.kind === 'tag')).toBe(true);
    // O insert é o PREFIXO do backend (não um chute do frontend).
    expect(items[0].insert).toBe('{CLR:');
    expect(items[0].detail).toBe('Cor do texto');
    expect(items[2].insert).toBe('{ICON:');
    expect(items[2].detail).toBe('Ícone');
  });

  it('filtra pelo prefixo digitado', () => {
    expect(computeSuggestions('C', FF13_CATALOG).map((i) => i.label)).toEqual(['CLR', 'CTR']);
    expect(computeSuggestions('V', FF13_CATALOG).map((i) => i.label)).toEqual(['VAR']);
    expect(computeSuggestions('BT', FF13_CATALOG).map((i) => i.label)).toEqual(['BTN']);
    expect(computeSuggestions('SP', FF13_CATALOG).map((i) => i.label)).toEqual(['SPACE']);
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
    expect(computeSuggestions('CLR:gol', FF13_CATALOG).map((i) => i.label)).toEqual(['Gold']);
    expect(computeSuggestions('CLR:F9:42', FF13_CATALOG).map((i) => i.label)).toEqual(['Gold']);
    // White é reset: não aparece como cor sugerida.
    expect(computeSuggestions('CLR:F9:40', FF13_CATALOG)).toEqual([]);
  });

  it('classe sem entrada no catálogo não sugere nada', () => {
    expect(computeSuggestions('PC:', FF13_CATALOG)).toEqual([]);
    expect(computeSuggestions('KEY:', FF13_CATALOG)).toEqual([]);
  });

  it('o autocomplete de CLR não sugere Ex nem White (reset)', () => {
    const labels = computeSuggestions('CLR:', FF13_CATALOG).map((i) => i.label);
    expect(labels).not.toContain('Ex00');
    expect(labels).not.toContain('White');
    expect(labels).toContain('Gold');
  });

  it('NL insere uma quebra de linha, não um chip', () => {
    const [item] = computeSuggestions('NL:', FF13_CATALOG);
    expect(item.label).toBe('Nova linha');
    expect(item.hardBreak).toBe(true);
    expect(item.chip).toBeUndefined();
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

  it('SPACE mostra o símbolo visível e não está travado', () => {
    const [item] = computeSuggestions('SPACE:', FF13_CATALOG);
    expect(item.chip?.label).toBe('⍽');
    expect(item.chip?.locked).toBe(false);
  });

  it('classes de valor do jogo ficam travadas (byte calculado pelo jogo)', () => {
    for (const inner of ['VAR:F4:40', 'CTR:F7:40', 'KEY:F6:40', 'PAGE:40:70', 'RAW:81:40', 'B:FF']) {
      expect(isLocked(inner)).toBe(true);
    }
    // Ícone, botão, cor e espaço são conteúdo: o tradutor pode escolher.
    expect(isLocked('ICON:F2:40:Sword')).toBe(false);
    expect(isLocked('BTN:F1:40:A')).toBe(false);
    expect(isLocked('CLR:F9:40:White')).toBe(false);
    expect(isLocked('SPACE:85:60')).toBe(false);
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

  it('cor só-nome não valida no FF13; a forma com bytes sim', () => {
    const validate = getChipValidator(FF13_CATALOG);
    expect(validate('CLR:White')).toBe(false);
    expect(validate('CLR:F9:40:White')).toBe(true);
  });
});

describe('paleta do catálogo', () => {
  it('vem da classe CLR, com hex, tag e grupo ex', () => {
    const palette = paletteFromCatalog(FF13_CATALOG, 'part1');
    expect(palette?.[0]).toEqual({
      label: 'White',
      name: 'White',
      hex: '#ffffff',
      tag: '{CLR:F9:40:White}',
      reset: true,
    });
    expect(palette?.find((c) => c.name === 'Ex00')).toMatchObject({
      group: 'ex',
      semiTransparent: true,
      tag: '{CLR:F9:32:Ex00}',
    });
  });

  it('FF13 sem catálogo devolve null (nunca paleta de outro idioma)', () => {
    expect(paletteFromCatalog(null, 'part1')).toBeNull();
  });

  it('não infla a paleta: dedupe por tag', () => {
    const values = Array.from({ length: 43 }, (_, i) => ({
      tag: `{CLR:F9:${(0x40 + i).toString(16).toUpperCase()}:C${i}}`,
      label: `C${i}`,
      key: `F9:${(0x40 + i).toString(16).toUpperCase()}`,
      hex: '#000000',
    }));
    const catalog: TagCatalog = {
      version: 'part1',
      tags: [{ tag: 'CLR', prefix: '{CLR:', hint: '', values: [...values, values[0]] }],
    };
    expect(paletteFromCatalog(catalog, 'part1')).toHaveLength(43);
  });
});

/** Import local para o lock (o mesmo que o popup usa). */
function isLocked(inner: string): boolean {
  // espelha isLockedTagInner sem importar ciclo no teste
  const upper = inner.trim().toUpperCase();
  const locked = ['VAR:', 'CTR:', 'KEY:', 'RAW:', 'B:', 'PAGE:'];
  return locked.some((p) => upper.startsWith(p));
}
