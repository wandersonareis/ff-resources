import { describe, expect, it } from 'vitest';
import { filelistLabel } from '@/lib/ffx/filelist-common';
import { shortFilelistName } from './part-ui';
import { part1, part1FilelistOrder } from './part1';

describe('part1 — rótulos dos índices de filelist', () => {
  it('o rótulo vem do common: nome sem extensão, minúsculas, com o idioma', () => {
    expect(filelistLabel({ name: 'filelistu.win32.bin' })).toBe('filelistu (inglês)');
    expect(filelistLabel({ name: 'filelistc.win32.bin' })).toBe('filelistc (japonês)');
    expect(filelistLabel({ name: 'filelist_scru.win32.bin' })).toBe('filelist_scru (inglês)');
    expect(filelistLabel({ name: 'filelist_scrc.win32.bin' })).toBe('filelist_scrc (japonês)');
  });

  it('tira o prefixo "filelist" e a extensão (nome curto)', () => {
    expect(shortFilelistName('filelistu.win32.bin')).toBe('u');
    expect(shortFilelistName('filelist_scru.win32.bin')).toBe('scru');
    expect(shortFilelistName('zone/filelist_z000u.win32.bin')).toBe('z000u');
  });
});

describe('part1 — a zona no topo', () => {
  it('as zonas viram o nó "Zone" (o conteúdo vive só ali)', () => {
    expect(part1.filelistZoneLabel).toBe('Zone');
    expect(part1.zoneAsTopLevel).toBe(true);
  });
});

describe('part1 — ordem dos índices', () => {
  const names = [
    'filelist_scrc.win32.bin',
    'filelist_z105u.win32.bin',
    'filelistu.win32.bin',
    'filelist_z000c.win32.bin',
    'filelistc.win32.bin',
    'filelist_scru.win32.bin',
    'filelist_z000u.win32.bin',
    'filelist_z105c.win32.bin',
  ];

  it('globais primeiro, inglês antes do japonês, zonas em ordem numérica', () => {
    const sorted = [...names].sort((a, b) =>
      part1FilelistOrder(a).localeCompare(part1FilelistOrder(b)),
    );
    expect(sorted).toEqual([
      'filelistu.win32.bin',
      'filelistc.win32.bin',
      'filelist_scru.win32.bin',
      'filelist_scrc.win32.bin',
      'filelist_z000u.win32.bin',
      'filelist_z000c.win32.bin',
      'filelist_z105u.win32.bin',
      'filelist_z105c.win32.bin',
    ]);
  });
});
