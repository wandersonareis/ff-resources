import { describe, expect, it } from 'vitest';
import { part3FilelistOrder } from './part3';

/**
 * Ordem das raízes de filelist do part3 (LR): o par é "a" (inglês) e "v"
 * (japonês) — o inglês vem antes do japonês dentro da mesma família, e as
 * famílias globais vêm antes dos scripts, que vêm antes das demais.
 */
describe('part3 — ordem dos índices de filelist', () => {
  it('o par "a"/"v" vira o par "u"/"c" do resto do app', () => {
    expect(part3FilelistOrder('filelist_a.win32.bin')).toBe('0||0');
    expect(part3FilelistOrder('filelist_v.win32.bin')).toBe('0||1');
  });

  it('scripts ("scr") vêm depois dos globais', () => {
    expect(part3FilelistOrder('filelist_scra.win32.bin')).toBe('1|scr|0');
    expect(part3FilelistOrder('filelist_scrv.win32.bin')).toBe('1|scr|1');
  });

  it('as demais famílias vêm depois de scripts e globais', () => {
    expect(part3FilelistOrder('filelist_p0000000img_a.win32.bin')).toBe('2|p0000000img_|0');
    expect(part3FilelistOrder('filelist_p0000000img_v.win32.bin')).toBe('2|p0000000img_|1');
  });

  it('o inglês vem antes do japonês dentro da mesma família', () => {
    const english = part3FilelistOrder('filelist_a.win32.bin');
    const japanese = part3FilelistOrder('filelist_v.win32.bin');
    expect(english < japanese).toBe(true);

    const scrEnglish = part3FilelistOrder('filelist_scra.win32.bin');
    const scrJapanese = part3FilelistOrder('filelist_scrv.win32.bin');
    expect(scrEnglish < scrJapanese).toBe(true);
  });

  it('a ordem das famílias: globais, scripts, demais', () => {
    const global = part3FilelistOrder('filelist_a.win32.bin');
    const script = part3FilelistOrder('filelist_scra.win32.bin');
    const other = part3FilelistOrder('filelist_p0000000img_a.win32.bin');
    expect(global < script).toBe(true);
    expect(script < other).toBe(true);
  });
});
