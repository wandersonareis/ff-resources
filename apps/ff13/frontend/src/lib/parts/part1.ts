import { type PartUi, shortFilelistName } from './part-ui';
import { buildUnitTree } from './unit-tree';

/**
 * part1 = FINAL FANTASY XIII.
 *
 * - /data: árvore por diretório interno do `.ztr` (txtres/system, zone/...);
 * - filelist: índices globais (u = inglês, c = japonês) e, no topo, o nó
 *   "Zone" com uma entrada por zona (z000, z002…).
 *
 * Os índices do part1 vêm em pares de idioma: o sufixo `u` é o conteúdo em
 * INGLÊS e o `c` o de JAPONÊS. O mesmo texto traduzido vale para os dois
 * idiomas (basta gravar nos dois arquivos), então a árvore mostra os dois —
 * a repetição é resolvida pelo dedupe na hora de aplicar, não escondendo
 * metade do jogo.
 *
 * O RÓTULO das raízes não é montado aqui: ele vem do `filelist-common` (nome
 * do arquivo sem extensão, em minúsculas, com o idioma entre parênteses).
 */

/** Separa o nome curto em (família, idioma). "scru" -> ("scr", "u"). */
function splitLang(short: string): { family: string; lang: string } {
  const m = /^(.*?)([uc])$/.exec(short);
  if (!m) return { family: short, lang: '' };
  return { family: m[1], lang: m[2] };
}

/**
 * Ordem das raízes do part1: globais primeiro (inglês antes do japonês),
 * depois scripts, depois as zonas em ordem numérica — com o inglês antes do
 * japonês dentro de cada zona.
 */
export function part1FilelistOrder(name: string): string {
  const short = shortFilelistName(name);
  const { family, lang } = splitLang(short);
  const familyRank = family === '' ? '0' : family === 'scr' ? '1' : '2';
  const langRank = lang === 'u' ? '0' : lang === 'c' ? '1' : '2';
  return `${familyRank}|${family}|${langRank}`;
}

export const part1: PartUi = {
  id: 'part1',
  titleLogoSrc: '/part1-logo.avif',
  dataGroupLabel: 'Data',
  filelistGroupLabel: 'Filelist',
  filelistZoneLabel: 'Zone',
  zoneAsTopLevel: true,
  typeLabels: {
    folder: 'Pasta',
    file: 'Arquivo',
    system: 'Sistema',
    mission: 'Missões',
    event: 'Eventos',
    zone: 'Zonas',
    book: 'Livro',
    review: 'Revisão',
    ac: 'Habilidades',
    text: 'Texto',
  },
  kindLabels: {
    ac: 'Habilidades',
    resident: 'Sistema',
    zone: 'Zonas',
    event: 'Eventos',
  },
  kinds: ['ac', 'resident', 'zone', 'event'],
  buildDataTree: (entries) => buildUnitTree(entries),
  filelistRootOrder: (name) => part1FilelistOrder(name),
};
