import type { EntryRow } from '@/lib/ffx/tree-data';
import { entryKindOfUnit } from '@/lib/ffx/text-kind';
import type { FilelistNode } from './filelist-common';
import { lowerName } from './filelist-common';

/**
 * A FOLHA de texto do filelist: o que ela é, que entry ela abre e como ela
 * se chama.
 *
 * A lógica mora aqui (e não na árvore nem na store) porque a folha do
 * filelist tem uma identidade própria: ela representa UM ARQUIVO — e o
 * arquivo tem IDIOMA ("txtres_us.ztr" é o inglês, "txtres_jp.ztr" o
 * japonês). A unidade de texto é a mesma nos dois casos (o id é o mesmo da
 * árvore /data), mas o idioma muda o que a tabela mostra.
 *
 * O `filelist-tree` usa este módulo para montar os nós; a store usa a entry
 * que veio no nó.
 */

/**
 * A folha é de TEXTO? (arquivo .ztr solto ou .bin container WPD — o backend
 * já resolveu isso e mandou `text` + `unitId`).
 */
export function isFilelistTextLeaf(node: FilelistNode): boolean {
  return Boolean(node.text && node.unitId);
}

/**
 * Entry da folha de texto: a UNIDADE de texto (o mesmo id da árvore /data:
 * "txtres/resident/book/txtres"), com o IDIOMA do arquivo clicado.
 *
 * O idioma entra em `lang`: é ele que faz a tabela mostrar o idioma pedido e
 * a edição ficar bloqueada fora do default.
 */
export function filelistEntryOf(node: FilelistNode, version: string): EntryRow | undefined {
  const unitId = node.unitId;
  if (!isFilelistTextLeaf(node) || !unitId) return undefined;
  return {
    // O kind sai do DIRETÓRIO da unidade (a família do texto): é o mesmo
    // mapeamento do backend, e faz o id do nó casar com a árvore /data
    // (`leaf:<kind>:<id>`). Sem isso, clicar numa folha do filelist não
    // realçaria a unidade correspondente em /data.
    kind: entryKindOfUnit(unitId),
    id: unitId,
    key: `${version}/${unitId}`,
    label: unitId.slice(unitId.lastIndexOf('/') + 1),
    container: node.container,
    // O idioma do ARQUIVO ("us"/"jp"/…): vazio quando o nome não diz.
    lang: node.unitLang || undefined,
  };
}

/**
 * Rótulo da folha de TEXTO: o nome do arquivo em minúsculas, COM a
 * extensão ("txtres_us.ztr") — é o nome que o índice usa e o que o usuário
 * reconhece. Os arquivos fora do escopo seguem a mesma regra (o nome é o que
 * identifica o arquivo do jogo).
 */
export function filelistLeafLabel(node: FilelistNode): string {
  return lowerName(node.name);
}

/**
 * Rótulo de um arquivo que o app NÃO abre: o nome do arquivo em minúsculas,
 * COM a extensão ("system.win32.imgb") — no filelist o nome é o que identifica
 * o arquivo do jogo (é o mesmo nome que o índice usa).
 */
export function filelistOpaqueLabel(node: FilelistNode): string {
  return lowerName(node.name);
}
