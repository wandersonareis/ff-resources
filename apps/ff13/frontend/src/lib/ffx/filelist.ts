import {
  ListFilelistDir,
  ListFilelistRoots,
  ListFilelistZone,
} from '@ff13-bindings/ffresources/apps/ff13/ff13service';
import type { FilelistNode, FilelistRoot } from './filelist-common';

/**
 * Acesso ao filelist do FF13 (SOMENTE LEITURA), no espírito do navegador de
 * .vbf do FFX: a árvore sai do índice e nada é decodificado. Nesta fase todo
 * arquivo é "formato incompatível" ao clicar (sem leitura de formato).
 *
 * Este módulo é a camada de DADOS: ele só fala com o backend e guarda o
 * cache. A formatação dos rótulos é do `filelist-tree` e as primitivas de
 * nome do `filelist-common`.
 */

/** Raiz virtual de uma zona no protocolo do backend ("zone:z000"). */
export const ZONE_ROOT_PREFIX = 'zone:';

/** Raiz virtual de uma zona (o id da zona depois do prefixo). */
export function zoneRoot(zone: string): string {
  return `${ZONE_ROOT_PREFIX}${zone}`;
}

const rootsCache = new Map<string, Promise<FilelistRoot[]>>();
const dirCache = new Map<string, Promise<FilelistNode[]>>();

/** Filelists descobertos no game root ou na árvore extraída. */
export function loadFilelistRoots(part: string): Promise<FilelistRoot[]> {
  let pending = rootsCache.get(part);
  if (!pending) {
    pending = ListFilelistRoots(part).then((roots) => roots ?? []);
    rootsCache.set(part, pending);
  }
  return pending;
}

/**
 * Filhos imediatos de um diretório do índice (diretórios primeiro).
 *
 * root é o caminho do índice ("indice.bin"), o de um índice embutido
 * ("indice.bin#zone/filelist_x") ou a raiz virtual de uma zona
 * ("zone:z000").
 */
export function loadFilelistDir(part: string, root: string, dir: string): Promise<FilelistNode[]> {
  const key = `${part}|${root}|${dir.toLowerCase()}`;
  const cached = dirCache.get(key);
  if (cached) return cached;
  const pending = ListFilelistDir(part, root, dir).then((nodes) => nodes ?? []);
  dirCache.set(key, pending);
  return pending;
}

/**
 * Conteúdo de uma ZONA: a união do índice de zona com o resto da part.
 *
 * O backend faz a união (índice de zona + entradas "zone/<z>/" dos demais
 * índices); o cache usa a mesma chave da raiz virtual para não duplicar
 * requisição.
 */
export function loadFilelistZone(part: string, zone: string): Promise<FilelistNode[]> {
  const key = `${part}|${zoneRoot(zone)}|`;
  const cached = dirCache.get(key);
  if (cached) return cached;
  const pending = ListFilelistZone(part, zone).then((nodes) => nodes ?? []);
  dirCache.set(key, pending);
  return pending;
}

/** Descarta o cache (troca de jogo/perfil ou reload da árvore). */
export function invalidateFilelistCache(): void {
  rootsCache.clear();
  dirCache.clear();
}
