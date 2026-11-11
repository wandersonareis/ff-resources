import type { SideNode } from '@/lib/ffx/content-tree/model';
import type { GameVersionId } from '@/lib/ffx/game-version';
import type { EntryRow } from '@/lib/ffx/tree-data';

/**
 * Particularidades de UI de UM jogo da trilogia. O backend manda dados + `type`
 * (sem traduzir); cada part decide rótulos, ícones e como montar a árvore de
 * /data. Part2/part3 podem divergir aqui sem tocar no part1.
 */
export interface PartUi {
  id: GameVersionId;
  /** Rótulo do grupo de arquivos extraídos (/data/partN). */
  dataGroupLabel: string;
  /** Rótulo do grupo de índices filelist. */
  filelistGroupLabel: string;
  /** Rótulos i18n por `node.type` (folder/file/system/mission/event/...). */
  typeLabels: Readonly<Record<string, string>>;
  /**
   * Rótulos i18n por KIND de texto (ac/resident/zone/event/scenario).
   * Fica separado do `typeLabels`: `type` é a categoria de CONTEÚDO (ícone
   * da linha), `kind` é a FAMÍLIA do texto (escopo de export/import e
   * raiz da árvore). São ortogonais — uma pasta `resident` pode ter
   * unidades de tipo system, mission, book...
   */
  kindLabels: Readonly<Partial<Record<string, string>>>;
  /** Kinds de TEXTO que a part servi (ordem canônica). */
  kinds: readonly string[];
  /** Monta os nós da árvore de /data. Cada part tem o seu agrupamento. */
  buildDataTree(entries: EntryRow[]): SideNode[];
  /**
   * Chave de ORDENAÇÃO das raízes de filelist (menor = primeiro). Cada part
   * decide: o part1 põe o inglês antes do japonês. Ausente = ordem
   * alfabética pelo nome do arquivo.
   *
   * O RÓTULO das raízes não vem da part: ele é o nome do arquivo sem
   * extensão, em minúsculas (com o idioma entre parênteses), montado pelo
   * `filelist-common`.
   */
  filelistRootOrder?: (name: string) => string;
  /** Rótulo do grupo que agrupa os índices de zona (pai "zone"). */
  filelistZoneLabel: string;
  /**
   * Efeito aplicado ao texto do título quando a part está ativa.
   * Ausente = título padrão (degradê). Cada efeito é um componente em
   * `components/` mapeado no `AppHeader`.
   */
  titleEffect?: 'lightning';
  /**
   * Fonte do logo do título (caminho em `public/`) quando a part troca o
   * texto por imagem — o efeito de `titleEffect` passa a recortar no logo.
   */
  titleLogoSrc?: string;
  /**
   * true quando os índices de ZONA viram o nó "Zone" no topo da árvore de
   * filelist (part1): o conteúdo da zona vive só ali.
   */
  zoneAsTopLevel?: boolean;
}

/**
 * Nome curto de um índice de filelist: tira o prefixo "filelist", o "_" e a
 * extensão. É a mesma regra para as três parts — o que muda entre elas é
 * apenas o nome do arquivo.
 *
 *   filelistu.win32.bin        -> u
 *   filelist_scru.win32.bin    -> scru
 *   filelist_patch_c.win32.bin -> patch_c
 *   filelist_z000u.win32.bin   -> z000u
 */
export function shortFilelistName(name: string): string {
  let n = name.replace(/\\/g, '/');
  n = n.slice(n.lastIndexOf('/') + 1);
  n = n.replace(/\.(win32\.)?bin2?$/i, '');
  n = n.replace(/^filelist[_-]?/i, '');
  return n;
}
