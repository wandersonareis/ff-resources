import { describe, expect, it } from 'vitest';
import {
  filelistLabel,
  languageLabel,
  languageOf,
  lowerName,
  mergeLabel,
  sourceLabel,
  stripExtension,
} from './filelist-common';

describe('filelist-common — nomes', () => {
  it('tira a extensão final (e o segmento de plataforma)', () => {
    expect(stripExtension('filelistu.win32.bin')).toBe('filelistu');
    expect(stripExtension('filelist_scrc.x360.bin')).toBe('filelist_scrc');
    expect(stripExtension('FILELIST_patch.PS3.BIN')).toBe('FILELIST_patch');
    expect(stripExtension('filelistu.bin')).toBe('filelistu');
    expect(stripExtension('filelistu.bin2')).toBe('filelistu');
    expect(stripExtension('zone/filelist_z000u.win32.bin')).toBe('filelist_z000u');
  });

  it('minúsculas', () => {
    expect(lowerName('FilelistU')).toBe('filelistu');
  });

  it('idioma pelo FINAL do nome sem extensão', () => {
    expect(languageOf('filelistu.win32.bin')).toBe('u');
    expect(languageOf('filelistc.win32.bin')).toBe('c');
    expect(languageOf('filelist_scru.win32.bin')).toBe('u');
    expect(languageOf('filelist_scrc.win32.bin')).toBe('c');
    expect(languageOf('filelist_z000u.win32.bin')).toBe('u');
    expect(languageOf('filelist_patch.win32.bin')).toBe('');
  });

  it('dicionários de idioma e plataforma', () => {
    expect(languageLabel('u')).toBe('inglês');
    expect(languageLabel('c')).toBe('japonês');
    expect(languageLabel('x')).toBe('');
    expect(languageLabel(undefined)).toBe('');
    expect(sourceLabel('win32')).toBe('PC');
    expect(sourceLabel('x360')).toBe('Xbox 360');
    expect(sourceLabel('ps3')).toBe('PlayStation 3');
    expect(sourceLabel(undefined)).toBe('');
  });

  it('rótulo do índice: nome sem extensão, minúsculas, com o idioma', () => {
    expect(filelistLabel({ name: 'filelistu.win32.bin' })).toBe('filelistu (inglês)');
    expect(filelistLabel({ name: 'filelistc.win32.bin' })).toBe('filelistc (japonês)');
    expect(filelistLabel({ name: 'filelist_scru.win32.bin' })).toBe('filelist_scru (inglês)');
    expect(filelistLabel({ name: 'filelist_scrc.win32.bin' })).toBe('filelist_scrc (japonês)');
    expect(filelistLabel({ name: 'filelist_patch.win32.bin' })).toBe('filelist_patch');
  });

  it('o idioma do DTO manda sobre o nome (o backend já resolveu)', () => {
    expect(filelistLabel({ name: 'filelistu.win32.bin', language: 'c' })).toBe(
      'filelistu (japonês)',
    );
    expect(filelistLabel({ name: 'filelistu.win32.bin', language: '' })).toBe('filelistu (inglês)');
  });

  it('rótulo do nó: a cadeia fundida quando existe, senão o nome', () => {
    expect(
      mergeLabel({
        name: 'empty',
        path: 'empty/empty',
        mergeChain: ['empty', 'empty'],
        isDir: true,
      }),
    ).toBe('empty/empty');
    expect(mergeLabel({ name: 'txtres', path: 'txtres', isDir: true })).toBe('txtres');
  });
});
