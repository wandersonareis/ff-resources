import { describe, expect, it } from 'vitest';
import type { FilelistNode } from './filelist-common';
import {
  filelistEntryOf,
  filelistLeafLabel,
  filelistOpaqueLabel,
  isFilelistTextLeaf,
} from './filelist-entry';

/** Nó do filelist com os campos que a folha de texto usa. */
function node(over: Partial<FilelistNode> = {}): FilelistNode {
  return {
    name: 'txtres_us.ztr',
    path: 'txtres/resident/book/txtres_us.ztr',
    isDir: false,
    text: true,
    unitId: 'txtres/resident/book/txtres',
    unitLang: 'us',
    container: 'ztr',
    ...over,
  };
}

describe('filelist-entry — folha de texto', () => {
  it('a folha é de TEXTO quando tem texto e id de unidade', () => {
    expect(isFilelistTextLeaf(node())).toBe(true);
    expect(isFilelistTextLeaf(node({ unitId: '' }))).toBe(false);
    expect(isFilelistTextLeaf(node({ text: false }))).toBe(false);
  });

  it('a entry abre a MESMA unidade da árvore /data, com o idioma do arquivo', () => {
    const entry = filelistEntryOf(node(), 'part1', 'root-1');

    expect(entry).toBeDefined();
    // O kind sai do DIRETÓRIO da unidade (a família do texto): `resident`
    // neste caso. É o que faz o id do nó casar com a árvore /data.
    expect(entry?.kind).toBe('resident');
    expect(entry?.id).toBe('txtres/resident/book/txtres');
    expect(entry?.key).toBe('part1/txtres/resident/book/txtres');
    expect(entry?.label).toBe('txtres');
    expect(entry?.lang).toBe('us');
    expect(entry?.container).toBe('ztr');
    // A origem separada: o texto é lido do filelist, não de data/.
    expect(entry?.source).toBe('filelist');
    expect(entry?.filelist).toEqual({
      root: 'root-1',
      path: 'txtres/resident/book/txtres_us.ztr',
    });
  });

  it('o kind da entry casa com o kind da árvore /data para a MESMA unidade', () => {
    // resident (txtres/resident), event (txtres/event), zone, scenario.
    const cases: Array<[string, string]> = [
      ['txtres/resident/book/txtres', 'resident'],
      ['txtres/event/ev_comn_000/txtres', 'event'],
      ['txtres/zone/z000/txtres', 'zone'],
      ['db/ai/npc/pack/auto_c01', 'scenario'],
      ['zone/z004/txtres/ac/ac_bodm_230/txtres', 'ac'],
    ];
    for (const [unitId, want] of cases) {
      const entry = filelistEntryOf(
        node({
          path: `${unitId}_us.ztr`,
          unitId,
          container: want === 'scenario' ? 'wpd' : 'ztr',
        }),
        'part3',
        'root-1',
      );
      expect(entry?.kind).toBe(want);
    }
  });

  it('o idioma fora do catálogo (nome que não diz) fica sem lang', () => {
    const entry = filelistEntryOf(node({ unitLang: '' }), 'part1', 'root-1');
    expect(entry?.lang).toBeUndefined();
  });

  it('nó sem texto não abre entry', () => {
    expect(filelistEntryOf(node({ text: false }), 'part1', 'root-1')).toBeUndefined();
    expect(filelistEntryOf(node({ unitId: '' }), 'part1', 'root-1')).toBeUndefined();
  });

  it('rótulo da folha de texto: o nome do arquivo em minúsculas', () => {
    expect(filelistLeafLabel(node({ name: 'TXTRES_US.ZTR' }))).toBe('txtres_us.ztr');
  });

  it('rótulo de arquivo fora do escopo: o nome do arquivo em minúsculas', () => {
    expect(filelistOpaqueLabel(node({ name: 'system.win32.imgb' }))).toBe('system.win32.imgb');
    expect(filelistOpaqueLabel(node({ name: 'AUDIO_01.OGG' }))).toBe('audio_01.ogg');
  });
});

