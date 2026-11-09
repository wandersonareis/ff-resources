import { describe, expect, it } from 'vitest';
import type { SideNode } from '@/lib/ffx/content-tree/model';
import type { EntryRow } from '@/lib/ffx/tree-data';
import { buildUnitTree, displaySegments } from './unit-tree';

/** Entrada de unidade de texto (o que o backend manda para o kind). */
function unit(id: string, label: string, extra: Partial<EntryRow> = {}): EntryRow {
  return {
    kind: 'resident',
    id,
    key: `part1/${id}`,
    label,
    type: 'text',
    text: true,
    ...extra,
  };
}

/** Procura um nó pelo id em qualquer nível da árvore. */
function findNode(nodes: SideNode[], id: string): SideNode | undefined {
  for (const node of nodes) {
    if (node.id === id) return node;
    const hit = node.children ? findNode(node.children, id) : undefined;
    if (hit) return hit;
  }
  return undefined;
}

describe('displaySegments — o prefixo estrutural do kind é só visual', () => {
  it('txtres/<kind> sai do caminho; o id da unidade não muda', () => {
    expect(displaySegments(unit('txtres/resident/book/txtres', 'txtres'))).toEqual([
      'book',
      'txtres',
    ]);
    expect(
      displaySegments(unit('txtres/event/ev_comn_000/txtres', 'txtres', { kind: 'event' })),
    ).toEqual(['ev_comn_000', 'txtres']);
    expect(displaySegments(unit('txtres/ac/ac_aaea_007/txtres', 'txtres', { kind: 'ac' }))).toEqual(
      ['ac_aaea_007', 'txtres'],
    );
    expect(displaySegments(unit('txtres/zone/z000/txtres', 'txtres', { kind: 'zone' }))).toEqual([
      'z000',
      'txtres',
    ]);
  });

  it('o txtres de zona do part1 também perde o invólucro zone/<z>', () => {
    expect(
      displaySegments(unit('zone/z002/txtres/ac/ac_hgcr_051/txtres', 'txtres', { kind: 'ac' })),
    ).toEqual(['ac_hgcr_051', 'txtres']);
    expect(
      displaySegments(
        unit('zone/z002/txtres/event/ev_hgcr_000/txtres', 'txtres', { kind: 'event' }),
      ),
    ).toEqual(['ev_hgcr_000', 'txtres']);
    expect(
      displaySegments(unit('zone/z002/txtres/zone/z002/txtres', 'txtres', { kind: 'zone' })),
    ).toEqual(['z002', 'txtres']);
  });

  it('scenario perde db/ai/npc/pack', () => {
    expect(
      displaySegments(
        unit('db/ai/npc/pack/auto_yus', 'auto_yus', { kind: 'scenario', container: 'wpd' }),
      ),
    ).toEqual(['auto_yus']);
  });

  it('caminho fora das pastas conhecidas fica inteiro (kind legado ztr)', () => {
    expect(displaySegments(unit('gui/x/y', 'y', { kind: 'ztr' }))).toEqual(['gui', 'x', 'y']);
  });
});

describe('buildUnitTree — devolve os FILHOS do kind (a raiz é do store)', () => {
  it('não cria raiz kind: devolve um nó por diretório com conteúdo', () => {
    const nodes = buildUnitTree([
      unit('txtres/resident/book/txtres', 'txtres'),
      unit('txtres/resident/game/txtres', 'txtres'),
      unit('txtres/resident/mission/txtres', 'txtres'),
    ]);

    expect(nodes.map((n) => n.label)).toEqual(['book', 'game', 'mission']);
    expect(nodes.every((n) => !n.id.startsWith('kind:'))).toBe(true);
    expect(findNode(nodes, 'kind:resident')).toBeUndefined();
  });

  it('pasta com UMA unidade vira o nó clicável (rótulo = nome da pasta)', () => {
    const nodes = buildUnitTree([unit('txtres/resident/book/txtres', 'txtres')]);

    expect(nodes).toHaveLength(1);
    expect(nodes[0].label).toBe('book');
    expect(nodes[0].children).toBeUndefined();
    // O id/caminho real continua completo — só o rótulo é o da pasta.
    expect(nodes[0].entry?.id).toBe('txtres/resident/book/txtres');
    expect(nodes[0].entry?.label).toBe('book');
    expect(nodes[0].id).toBe('leaf:resident:txtres/resident/book/txtres');
  });

  it('pasta com VÁRIAS unidades mantém uma folha por unidade', () => {
    const nodes = buildUnitTree([
      unit('txtres/resident/misc/foo', 'foo'),
      unit('txtres/resident/misc/bar', 'bar'),
    ]);

    expect(nodes).toHaveLength(1);
    const misc = nodes[0];
    expect(misc.type).toBe('folder');
    expect(misc.label).toBe('misc');
    expect(misc.children?.map((n) => n.label)).toEqual(['bar', 'foo']);
    for (const child of misc.children ?? []) {
      expect(child.entry?.id).toContain('/misc/');
      expect(child.entry?.label).toBe(child.label);
    }
  });

  it('colapsa diretórios sem unidade direta e com filho único', () => {
    const nodes = buildUnitTree([unit('txtres/resident/a/b/one/txtres', 'txtres')]);

    expect(nodes).toHaveLength(1);
    expect(nodes[0].label).toBe('one');
    expect(nodes[0].entry?.id).toBe('txtres/resident/a/b/one/txtres');
  });

  it('para de colapsar no diretório que ramifica', () => {
    const nodes = buildUnitTree([
      unit('txtres/resident/a/b/one/txtres', 'txtres'),
      unit('txtres/resident/a/b/two/txtres', 'txtres'),
    ]);

    expect(nodes).toHaveLength(1);
    expect(nodes[0].label).toBe('b');
    expect(nodes[0].type).toBe('folder');
    expect(nodes[0].children?.map((n) => n.label)).toEqual(['one', 'two']);
  });

  it('o kind ZONAS mostra o id da zona direto (Zonas > zXXX)', () => {
    const nodes = buildUnitTree([
      unit('zone/z002/txtres/zone/z002/txtres', 'txtres', { kind: 'zone' }),
      unit('zone/z003/txtres/zone/z003/txtres', 'txtres', { kind: 'zone' }),
    ]);

    expect(nodes.map((n) => n.label)).toEqual(['z002', 'z003']);
    expect(nodes.map((n) => n.entry?.id)).toEqual([
      'zone/z002/txtres/zone/z002/txtres',
      'zone/z003/txtres/zone/z003/txtres',
    ]);
  });

  it('deduplica escolhendo a cópia com tradução em mods/ (modded)', () => {
    const global = unit('txtres/event/ev_comn_000/txtres', 'txtres', { kind: 'event' });
    const zone = unit('zone/z000/txtres/event/ev_comn_000/txtres', 'txtres', {
      kind: 'event',
      modded: true,
    });
    const nodes = buildUnitTree([zone, global]);

    // Uma folha só (a pasta vira o nó clicável) e o id é o da cópia que
    // realmente tem a tradução em mods/.
    expect(nodes).toHaveLength(1);
    expect(nodes[0].label).toBe('ev_comn_000');
    expect(nodes[0].entry?.id).toBe(zone.id);
  });

  it('sem tradução em nenhuma cópia, prefere a global/mais curta', () => {
    const global = unit('txtres/event/ev_comn_000/txtres', 'txtres', { kind: 'event' });
    const zone = unit('zone/z000/txtres/event/ev_comn_000/txtres', 'txtres', { kind: 'event' });
    const nodes = buildUnitTree([zone, global]);

    expect(nodes).toHaveLength(1);
    expect(nodes[0].label).toBe('ev_comn_000');
    expect(nodes[0].entry?.id).toBe(global.id);
  });

  it('cenário (WPD) isolado é folha com o rótulo sem o _us', () => {
    const nodes = buildUnitTree([
      unit('db/ai/npc/pack/auto_yus', 'auto_yus', {
        kind: 'scenario',
        container: 'wpd',
        langs: ['us'],
      }),
    ]);

    expect(nodes).toHaveLength(1);
    expect(nodes[0].label).toBe('auto_yus');
    expect(nodes[0].entry?.container).toBe('wpd');
    expect(nodes[0].entry?.langs).toEqual(['us']);
  });
});
