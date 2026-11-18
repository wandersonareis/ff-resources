import { type PartUi, shortFilelistName } from './part-ui';
import { buildUnitTree } from './unit-tree';

/**
 * part3 = LIGHTNING RETURNS: FINAL FANTASY XIII.
 *
 * A árvore de /data ganha o grupo de DLC (cada DLC é um índice próprio em
 * weiss_data/dlc) e o esquema de tipos do backend já manda `dlc`/`zone`.
 */

/** Separa o nome curto em (família, idioma). "scra" -> ("scr", "u"). */
function splitLang(short: string): { family: string; lang: string } {
  const m = /^(.*?)([av])$/.exec(short);
  if (!m) return { family: short, lang: '' };
  return { family: m[1], lang: m[2] === 'a' ? 'u' : 'c' };
}

/**
 * Ordem das raízes do part3: globais primeiro (inglês antes do japonês),
 * depois scripts, depois os demais — com o inglês antes do japonês dentro de
 * cada família (a mesma régua do part1).
 *
 * O part3 não usa o par "u"/"c" do part1/2: o par é "a" (inglês) e "v"
 * (japonês), e o marcador é o último caractere do nome curto.
 */
export function part3FilelistOrder(name: string): string {
  const short = shortFilelistName(name);
  const { family, lang } = splitLang(short);
  const familyRank = family === '' ? '0' : family === 'scr' ? '1' : '2';
  const langRank = lang === 'u' ? '0' : lang === 'c' ? '1' : '2';
  return `${familyRank}|${family}|${langRank}`;
}

export const part3: PartUi = {
  id: 'part3',
  titleEffect: 'lightning',
  titleLogoSrc: '/part3-logo.avif',
  dataGroupLabel: 'Data',
  filelistGroupLabel: 'Filelist',
  filelistZoneLabel: 'zone',
  typeLabels: {
    folder: 'Pasta',
    file: 'Arquivo',
    system: 'Sistema',
    mission: 'Missões',
    event: 'Eventos',
    zone: 'Zonas',
    dlc: 'DLC',
    text: 'Texto',
  },
  // O part3 (LR) não tem `ac`; acrescenta o `scenario` (db/ai/npc/pack).
  kindLabels: {
    resident: 'Sistema',
    zone: 'Zonas',
    event: 'Eventos',
    scenario: 'Cenários',
  },
  kinds: ['resident', 'zone', 'event', 'scenario'],
  buildDataTree: (entries) => buildUnitTree(entries),
  filelistRootOrder: (name) => part3FilelistOrder(name),
};
