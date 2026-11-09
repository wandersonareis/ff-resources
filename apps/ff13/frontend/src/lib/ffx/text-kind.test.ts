import { describe, expect, it } from 'vitest';
import { entryKindOfUnit, textKindOfPath } from './text-kind';

describe('textKindOfPath — o kind sai do DIRETÓRIO (não da extensão)', () => {
  it('txtres global: a pasta abaixo de txtres é a família', () => {
    expect(textKindOfPath('txtres/event/ev_comn_000/txtres_us.ztr')).toBe('event');
    expect(textKindOfPath('txtres/resident/book/txtres_us.ztr')).toBe('resident');
    expect(textKindOfPath('txtres/resident/system/txtres_us.ztr')).toBe('resident');
    expect(textKindOfPath('txtres/zone/z000/txtres_us.ztr')).toBe('zone');
    expect(textKindOfPath('txtres/ac/ac_aaea_007/txtres_us.ztr')).toBe('ac');
  });

  it('txtres de ZONA (part1): o mesmo ac vive dentro das zonas', () => {
    expect(textKindOfPath('zone/z004/txtres/ac/ac_bodm_230/txtres_us.ztr')).toBe('ac');
    expect(textKindOfPath('zone/z004/txtres/event/ev_bodm_200/txtres_us.ztr')).toBe('event');
    expect(textKindOfPath('zone/z004/txtres/zone/z004/txtres_us.ztr')).toBe('zone');
  });

  it('scenario é o db/ai/npc/pack (container, não .ztr solto)', () => {
    expect(textKindOfPath('db/ai/npc/pack/auto_c01_us.bin')).toBe('scenario');
    expect(textKindOfPath('db/ai/npc/pack/quest1008_us.bin')).toBe('scenario');
    // Fora do pack não é scenario.
    expect(textKindOfPath('db/ext/z0161pack.bin')).toBeUndefined();
    expect(textKindOfPath('db/resident/wdbpack.bin')).toBeUndefined();
  });

  it('caminhos fora do txtres/pack não casam', () => {
    expect(textKindOfPath('gui/resident/tutorial/x_us.ztr')).toBeUndefined();
    expect(textKindOfPath('sys/script/btl/x.bin')).toBeUndefined();
  });

  it('não distingue caixa nem barras invertidas', () => {
    expect(textKindOfPath('TXTRES/AC/AC_AAEA_007/TXTRES_US.ZTR')).toBe('ac');
    expect(textKindOfPath('zone\\z004\\txtres\\ac\\x\\txtres_us.ztr')).toBe('ac');
  });
});

describe('entryKindOfUnit — fallback', () => {
  it('sem pasta conhecida cai no kind legado ztr', () => {
    expect(entryKindOfUnit('gui/x/y')).toBe('ztr');
    expect(entryKindOfUnit('txtres/event/ev_a/txtres')).toBe('event');
  });
});
