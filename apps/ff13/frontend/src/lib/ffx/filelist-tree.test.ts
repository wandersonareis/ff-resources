import { describe, expect, it } from 'vitest';
import { part1 } from '@/lib/parts/part1';
import { part2 } from '@/lib/parts/part2';
import type { FilelistNode, FilelistRoot } from './filelist-common';
import { buildFilelistChildren, buildFilelistRoots } from './filelist-tree';

function root(over: Partial<FilelistRoot> = {}): FilelistRoot {
  return {
    name: 'filelistu.win32.bin',
    path: 'C:/game/white_data/sys/filelistu.win32.bin',
    part: 'part1',
    sourceDir: 'white_data',
    entries: 5,
    source: 'win32',
    language: 'u',
    kind: 'disk',
    ...over,
  };
}

function node(over: Partial<FilelistNode> = {}): FilelistNode {
  return {
    name: 'file.bin',
    path: 'file.bin',
    isDir: false,
    kind: 'file',
    ...over,
  };
}

const ZONE_U = root({
  name: 'filelist_z000u.win32.bin',
  path: 'C:/game/white_data/sys/filelistu.win32.bin#zone/filelist_z000u.win32.bin',
  group: 'zone',
  zone: 'z000',
  kind: 'embedded',
});
const ZONE_C = root({
  name: 'filelist_z000c.win32.bin',
  path: 'C:/game/white_data/sys/filelistc.win32.bin#zone/filelist_z000c.win32.bin',
  group: 'zone',
  zone: 'z000',
  language: 'c',
  kind: 'embedded',
});
const ZONE_2 = root({
  name: 'filelist_z002u.win32.bin',
  path: 'C:/game/white_data/sys/filelistu.win32.bin#zone/filelist_z002u.win32.bin',
  group: 'zone',
  zone: 'z002',
  kind: 'embedded',
});

describe('filelist-tree — raízes', () => {
  it('nomeia os índices globais e põe a plataforma no tooltip', () => {
    const nodes = buildFilelistRoots(
      [
        root({ name: 'filelist_scrc.win32.bin', path: 'p/scrc', language: 'c' }),
        root(),
        root({ name: 'filelistc.win32.bin', path: 'p/c', language: 'c' }),
        root({ name: 'filelist_scru.win32.bin', path: 'p/scru' }),
      ],
      part1,
    );

    expect(nodes.map((n) => n.label)).toEqual([
      'filelistu (inglês)',
      'filelistc (japonês)',
      'filelist_scru (inglês)',
      'filelist_scrc (japonês)',
    ]);
    expect(nodes[0].hint).toBe('PC · white_data');
    expect(nodes[0].filelistRoot).toBe('C:/game/white_data/sys/filelistu.win32.bin');
  });

  it('a lista pode ter índices de plataformas diferentes', () => {
    const nodes = buildFilelistRoots(
      [
        root(),
        root({
          name: 'filelistu.x360.bin',
          path: 'p/x360',
          source: 'x360',
          sourceDir: 'data/part1',
        }),
      ],
      part1,
    );
    expect(nodes.map((n) => n.hint)).toEqual(['PC · white_data', 'Xbox 360 · data/part1']);
  });

  it('part1: as zonas viram o nó "Zone" com um filho por zona', () => {
    const nodes = buildFilelistRoots([ZONE_U, ZONE_C, ZONE_2], part1);
    expect(nodes).toHaveLength(1);
    const zone = nodes[0];
    expect(zone.id).toBe('filelist:zone|');
    expect(zone.label).toBe('Zone');
    expect(zone.children?.map((c) => c.label)).toEqual(['z000', 'z002']);
    expect(zone.children?.[0].filelistZone).toBe('z000');
    expect(zone.children?.[0].filelistRoot).toBe('zone:z000');
    expect(zone.hint).toBe('PC · white_data');
  });

  it('part2/part3: as zonas continuam no grupo "zone", com os índices dentro', () => {
    const nodes = buildFilelistRoots([ZONE_U, ZONE_C, ZONE_2], part2);
    const zone = nodes[nodes.length - 1];
    expect(zone.label).toBe('zone');
    expect(zone.children?.map((c) => c.label)).toEqual([
      'filelist_z000c (japonês)',
      'filelist_z000u (inglês)',
      'filelist_z002u (inglês)',
    ]);
  });

  it('as raízes começam fechadas (a carga é sob demanda)', () => {
    const nodes = buildFilelistRoots([root()], part1);
    expect(nodes[0].children).toBeUndefined();
    expect(nodes[0].expandable).toBe(true);
  });
});

describe('filelist-tree — filhos', () => {
  it('diretório vira pasta expansível, com o rótulo da cadeia fundida', () => {
    const children = buildFilelistChildren(
      'C:/game/white_data/sys/filelistu.win32.bin',
      [
        node({
          name: 'empty',
          path: 'empty/empty',
          isDir: true,
          kind: 'dir',
          mergeChain: ['empty', 'empty'],
        }),
      ],
      part1,
      'part1',
    );
    expect(children).toHaveLength(1);
    expect(children[0].label).toBe('empty/empty');
    expect(children[0].type).toBe('folder');
    expect(children[0].filelistPath).toBe('empty/empty');
    expect(children[0].expandable).toBe(true);
  });

  it('arquivo fora do escopo vira folha "fora do escopo"; índice embutido vira folha expansível', () => {
    const children = buildFilelistChildren(
      'C:/game/white_data/sys/filelistu.win32.bin',
      [
        node({ name: 'system.win32.imgb', path: 'gui/system.win32.imgb' }),
        node({
          name: 'filelist_z000u.win32.bin',
          path: 'zone/filelist_z000u.win32.bin',
          kind: 'index',
          language: 'u',
        }),
      ],
      part1,
      'part1',
    );
    expect(children[0].label).toBe('system.win32.imgb');
    expect(children[0].unsupported).toBe(true);
    expect(children[0].entry).toBeUndefined();
    expect(children[0].expandable).toBe(false);
    expect(children[1].label).toBe('filelist_z000u (inglês)');
    expect(children[1].nestedFilelist).toBe(true);
    expect(children[1].expandable).toBe(true);
    expect(children[1].unsupported).toBe(false);
  });

  it('texto (.ztr) vira a entrada da unidade, com o nome do arquivo no rótulo', () => {
    const children = buildFilelistChildren(
      'C:/game/white_data/sys/filelistu.win32.bin',
      [
        node({
          name: 'txtres_us.ztr',
          path: 'txtres/resident/book/txtres_us.ztr',
          text: true,
          unitId: 'txtres/resident/book/txtres',
          container: 'ztr',
        }),
      ],
      part1,
      'part1',
    );
    expect(children[0].label).toBe('txtres_us.ztr');
    expect(children[0].unsupported).toBe(false);
    expect(children[0].entry).toEqual({
      kind: 'resident',
      id: 'txtres/resident/book/txtres',
      key: 'part1/txtres/resident/book/txtres',
      label: 'txtres',
      container: 'ztr',
    });
  });

  it('a folha de texto carrega o IDIOMA do arquivo (a tabela abre o idioma clicado)', () => {
    const children = buildFilelistChildren(
      'C:/game/white_data/sys/filelistu.win32.bin',
      [
        node({
          name: 'txtres_jp.ztr',
          path: 'txtres/resident/book/txtres_jp.ztr',
          text: true,
          unitId: 'txtres/resident/book/txtres',
          unitLang: 'jp',
          container: 'ztr',
        }),
      ],
      part1,
      'part1',
    );
    expect(children[0].entry?.id).toBe('txtres/resident/book/txtres');
    expect(children[0].entry?.lang).toBe('jp');
  });

  it('container WPD (.bin) vira a entrada da unidade e mantém a extensão no rótulo', () => {
    const children = buildFilelistChildren(
      'C:/game/white_data/sys/filelistu.win32.bin',
      [
        node({
          name: 'auto_c01_us.bin',
          path: 'db/ai/npc/pack/auto_c01_us.bin',
          text: true,
          unitId: 'db/ai/npc/pack/auto_c01',
          container: 'wpd',
        }),
      ],
      part1,
      'part1',
    );
    expect(children[0].label).toBe('auto_c01_us.bin');
    expect(children[0].unsupported).toBe(false);
    expect(children[0].entry?.id).toBe('db/ai/npc/pack/auto_c01');
    expect(children[0].entry?.container).toBe('wpd');
  });

  it('part1: o diretório "zone" dentro de um índice fica oculto', () => {
    const children = buildFilelistChildren(
      'C:/game/white_data/sys/filelistu.win32.bin',
      [
        node({ name: 'zone', path: 'zone', isDir: true, kind: 'dir' }),
        node({ name: 'txtres', path: 'txtres', isDir: true, kind: 'dir' }),
      ],
      part1,
      'part1',
    );
    expect(children.map((c) => c.label)).toEqual(['txtres']);
  });

  it('part2: o diretório "zone" continua visível', () => {
    const children = buildFilelistChildren(
      'C:/game/alba_data/sys/filelistu.win32.bin',
      [node({ name: 'zone', path: 'zone', isDir: true, kind: 'dir' })],
      part2,
      'part2',
    );
    expect(children.map((c) => c.label)).toEqual(['zone']);
  });
});
