import type { PartUi } from './part-ui';
import { buildUnitTree } from './unit-tree';

/**
 * part2 = FINAL FANTASY XIII-2.
 *
 * Mesma árvore por diretório do `.ztr`, mas o conteúdo tem categorias
 * próprias (udp/patch, coliseum). Os rótulos vivem aqui para poderem divergir
 * do part1 sem mexer na árvore compartilhada.
 */
export const part2: PartUi = {
  id: 'part2',
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
    text: 'Texto',
    udp: 'Patches',
  },
  kindLabels: {
    ac: 'Habilidades',
    resident: 'Sistema',
    zone: 'Zonas',
    event: 'Eventos',
  },
  kinds: ['ac', 'resident', 'zone', 'event'],
  buildDataTree: (entries) => buildUnitTree(entries),
};
