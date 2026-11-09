import type {
  FilelistNode,
  FilelistRoot,
} from '@ff13-bindings/ffresources/apps/ff13/common/models';

/**
 * Contrato do navegador de filelist: os tipos do DTO (do backend) e as
 * PRIMITIVAS de nome que o frontend usa para formatar os rótulos.
 *
 * O backend entrega só FATOS (nome cru, caminho, códigos de idioma e
 * plataforma); toda a formatação de texto mora aqui — é o ponto de partida
 * para um framework de i18n (os dicionários LANGUAGE_LABELS/SOURCE_LABELS).
 */

// Os tipos são reexportados: a store da árvore fala com o navegador de
// filelist e não deve conhecer o caminho dos bindings.
export type { FilelistNode, FilelistRoot };

/** Idiomas dos índices (o que muda é o áudio do part1). */
export const LANGUAGE_LABELS: Readonly<Record<string, string>> = {
  u: 'inglês',
  c: 'japonês',
};

/** Plataformas possíveis de um índice (derivadas do nome do arquivo). */
export const SOURCE_LABELS: Readonly<Record<string, string>> = {
  win32: 'PC',
  x360: 'Xbox 360',
  ps3: 'PlayStation 3',
};

/**
 * Tira a extensão final do nome: ".win32.bin", ".x360.bin", ".ps3.bin",
 * ".bin" e ".bin2".
 *
 *   filelistu.win32.bin  -> filelistu
 *   filelist_scrc.bin    -> filelist_scrc
 */
export function stripExtension(name: string): string {
  const base = baseName(name);
  return base.replace(/\.(win32|x360|ps3)\.bin2?$/i, '').replace(/\.bin2?$/i, '');
}

/** Nome em minúsculas (a árvore do filelist é toda em minúsculas). */
export function lowerName(name: string): string {
  return name.toLowerCase();
}

/**
 * Idioma do índice pelo NOME: "u" quando o nome (sem extensão) termina em
 * "u" (inglês) e "c" quando termina em "c" (japonês). Regra mecânica: um
 * nome que termina em "c" é japonês mesmo que seja por coincidência.
 */
export function languageOf(name: string): string {
  const stem = stripExtension(name);
  if (!stem) return '';
  const last = stem.slice(-1);
  return last === 'u' || last === 'c' ? last : '';
}

/** Rótulo do idioma ("inglês"/"japonês"); vazio quando o código é desconhecido. */
export function languageLabel(code: string | undefined | null): string {
  if (!code) return '';
  return LANGUAGE_LABELS[code] ?? '';
}

/** Rótulo da plataforma ("PC"/"Xbox 360"/"PlayStation 3"). */
export function sourceLabel(code: string | undefined | null): string {
  if (!code) return '';
  return SOURCE_LABELS[code] ?? '';
}

/**
 * Rótulo de um índice de filelist: o nome do arquivo sem extensão, em
 * minúsculas, com o idioma entre parênteses.
 *
 *   filelistu.win32.bin     -> filelistu (inglês)
 *   filelist_scrc.win32.bin -> filelist_scrc (japonês)
 */
export function filelistLabel(root: { name: string; language?: string | null }): string {
  const label = lowerName(stripExtension(root.name));
  const language = root.language || languageOf(root.name);
  const text = languageLabel(language);
  return text ? `${label} (${text})` : label;
}

/**
 * Rótulo de um nó da árvore: a cadeia fundida ("empty/empty") quando o
 * backend fundiu diretórios com filho único, senão o nome do nó.
 */
export function mergeLabel(node: FilelistNode): string {
  const chain = node.mergeChain;
  if (chain && chain.length > 0) return lowerName(chain.join('/'));
  return lowerName(node.name);
}

/** baseName devolve o último segmento de um caminho ("/" e "\" separiam). */
function baseName(name: string): string {
  const normalized = name.replace(/\\/g, '/');
  return normalized.slice(normalized.lastIndexOf('/') + 1);
}
