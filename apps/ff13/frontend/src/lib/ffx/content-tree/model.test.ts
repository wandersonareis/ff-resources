import { describe, expect, it } from 'vitest';
import {
  ancestorPathOf,
  buildContentRoots,
  buildNodeIndex,
  entryCheckState,
  entryCheckStateByKind,
  entryIdsInNode,
  mergeLoadedChildren,
  nextCheckState,
  nodeEntries,
  type SideNode,
  unloadedImageContainers,
} from '@/lib/ffx/content-tree/model';
import type { EntryKind } from '@/lib/ffx/display-names';

const imageTree: SideNode = {
  id: 'category',
  label: 'category',
  kind: 'images',
  children: [
    {
      id: 'directory',
      label: 'directory',
      kind: 'images',
      children: [
        {
          id: 'leaf:a',
          label: 'a',
          kind: 'images',
          entry: { kind: 'images', id: 'a', key: 'a', label: 'a' },
        },
        {
          id: 'leaf:b',
          label: 'b',
          kind: 'images',
          entry: { kind: 'images', id: 'b', key: 'b', label: 'b' },
        },
      ],
    },
  ],
};

describe('data tree checkbox selection', () => {
  it('collects every image leaf under nested groups', () => {
    expect(entryIdsInNode(imageTree)).toEqual(['a', 'b']);
  });

  it('calculates tri-state from every descendant leaf', () => {
    expect(entryCheckState(imageTree, new Set())).toBe(false);
    expect(entryCheckState(imageTree, new Set(['a']))).toBe('indeterminate');
    expect(entryCheckState(imageTree, new Set(['a', 'b']))).toBe(true);
  });
});

describe('filelist group selection (kinds mistos)', () => {
  const mixed: SideNode = {
    id: 'filelist:root|dir',
    label: 'dir',
    filelist: true,
    children: [
      {
        id: 'filelist:root|a.ztr',
        label: 'a.ztr',
        kind: 'resident',
        entry: { kind: 'resident', id: 'txt/a', key: 'k', label: 'a' },
      },
      {
        id: 'filelist:root|b.dds',
        label: 'b.dds',
        kind: 'images',
        entry: { kind: 'images', id: 'ref-b', key: 'ref-b', label: 'b' },
      },
      // Arquivo fora do escopo: sem entry, não entra na seleção.
      { id: 'filelist:root|system.imgb', label: 'system.imgb', unsupported: true },
    ],
  };

  it('lists only the compatible leaves, with their own kind', () => {
    expect(nodeEntries(mixed)).toEqual([
      { kind: 'resident', id: 'txt/a' },
      { kind: 'images', id: 'ref-b' },
    ]);
  });

  it('reads each leaf from its own kind set', () => {
    const byKind = new Map<EntryKind, ReadonlySet<string>>([
      ['resident', new Set(['txt/a'])],
      ['images', new Set(['ref-b'])],
    ]);
    expect(entryCheckStateByKind(mixed, new Map())).toBe(false);
    expect(
      entryCheckStateByKind(
        mixed,
        new Map<EntryKind, ReadonlySet<string>>([['resident', new Set(['txt/a'])]]),
      ),
    ).toBe('indeterminate');
    expect(entryCheckStateByKind(mixed, byKind)).toBe(true);
  });

  it('a group with unlisted subtrees never reports a full check', () => {
    const collapsed: SideNode = {
      id: 'filelist:root|dir',
      label: 'dir',
      expandable: true,
    };
    expect(entryCheckStateByKind(collapsed, new Map())).toBe(false);

    const partiallyListed: SideNode = {
      ...collapsed,
      children: [
        {
          id: 'filelist:root|a.ztr',
          label: 'a.ztr',
          kind: 'resident',
          entry: { kind: 'resident', id: 'txt/a', key: 'k', label: 'a' },
        },
        { id: 'filelist:root|sub', label: 'sub', expandable: true },
      ],
    };
    const byKind = new Map<EntryKind, ReadonlySet<string>>([['resident', new Set(['txt/a'])]]);
    expect(entryCheckStateByKind(partiallyListed, byKind)).toBe('indeterminate');
  });
});

describe('nextCheckState (clique no checkbox)', () => {
  const folder = (children: SideNode[]): SideNode => ({
    id: 'datadir:folder',
    label: 'folder',
    children,
  });
  const imageLeaf = (id: string): SideNode => ({
    id: `leaf:images:${id}`,
    label: id,
    kind: 'images',
    entry: { kind: 'images', id, key: id, label: id },
  });
  // Container de imagem ainda FECHADO: existe como nó, sem filhos listados.
  const closedContainer: SideNode = {
    id: 'imagecontainer:ref',
    label: 'container.bin',
    kind: 'images',
    imageContainer: true,
    imageRef: 'ref',
    expandable: true,
  };
  const byKind = (ids: string[]) =>
    new Map<EntryKind, ReadonlySet<string>>([['images', new Set(ids)]]);

  it('marks a group that has nothing selected', () => {
    const node = folder([imageLeaf('a'), closedContainer]);
    expect(nextCheckState(node, byKind([]))).toBe(true);
  });

  it('completes a partially marked group', () => {
    const node = folder([imageLeaf('a'), imageLeaf('b')]);
    expect(nextCheckState(node, byKind(['a']))).toBe(true);
  });

  it('unmarks a fully marked group', () => {
    const node = folder([imageLeaf('a'), imageLeaf('b')]);
    expect(nextCheckState(node, byKind(['a', 'b']))).toBe(false);
  });

  it('unmarks a saturated group even with closed containers below (the stuck case)', () => {
    const node = folder([imageLeaf('a'), closedContainer]);
    // Todas as folhas LISTADAS marcadas + container fechado: o estado visual
    // fica `indeterminate` para sempre, e o Radix pediria `true`. A intenção
    // do clique é remover.
    expect(entryCheckStateByKind(node, byKind(['a']))).toBe('indeterminate');
    expect(nextCheckState(node, byKind(['a']))).toBe(false);
  });
});

describe('unloadedImageContainers', () => {
  const leaf = (id: string): SideNode => ({
    id: `leaf:images:${id}`,
    label: id,
    kind: 'images',
    entry: { kind: 'images', id, key: id, label: id },
  });
  const container = (ref: string, children?: SideNode[]): SideNode => ({
    id: `imagecontainer:${ref}`,
    label: `${ref}.bin`,
    kind: 'images',
    imageContainer: true,
    imageRef: ref,
    expandable: true,
    ...(children ? { children } : {}),
  });

  it('collects closed containers at any depth', () => {
    const node: SideNode = {
      id: 'datadir:images',
      label: 'images',
      children: [
        leaf('a'),
        container('wpd1'),
        {
          id: 'datadir:images/sub',
          label: 'sub',
          children: [container('trb1'), container('wpdOpen', [leaf('b')])],
        },
      ],
    };
    expect(unloadedImageContainers(node).map((n) => n.imageRef)).toEqual(['wpd1', 'trb1']);
  });

  it('ignores a text unit that is also a container', () => {
    const textContainer: SideNode = {
      ...container('binText'),
      entry: { kind: 'resident', id: 'txt/unit', key: 'k', label: 'unit' },
    };
    expect(unloadedImageContainers(textContainer)).toEqual([]);
  });
});

describe('mergeLoadedChildren', () => {
  it('keeps loaded subtrees by id when the roots are rebuilt', () => {
    const prev: SideNode[] = [
      {
        id: 'filelist:root|',
        label: 'filelistu',
        filelist: true,
        expandable: true,
        children: [
          {
            id: 'filelist:root|txtres',
            label: 'txtres',
            filelist: true,
            children: [{ id: 'filelist:root|txtres/a.ztr', label: 'a.ztr' }],
          },
        ],
      },
    ];
    const next: SideNode[] = [{ id: 'filelist:root|', label: 'filelistu', expandable: true }];
    const merged = mergeLoadedChildren(next, prev);
    expect(merged[0].children?.[0].children?.[0].label).toBe('a.ztr');
    // A raiz antiga não é reaproveitada: o nó novo manda no label/flags.
    expect(merged[0]).not.toBe(prev[0]);
  });

  it('drops nodes that disappeared and keeps static children of the new tree', () => {
    const prev: SideNode[] = [
      {
        id: 'filelist:zone|',
        label: 'Zone',
        children: [
          {
            id: 'filelist:zone:z000',
            label: 'z000',
            filelistZone: 'z000',
            children: [{ id: 'zone-leaf', label: 'loaded' }],
          },
          { id: 'gone', label: 'gone', children: [{ id: 'x', label: 'x' }] },
        ],
      },
    ];
    const next: SideNode[] = [
      {
        id: 'filelist:zone|',
        label: 'Zone',
        children: [{ id: 'filelist:zone:z000', label: 'z000', filelistZone: 'z000' }],
      },
    ];
    const merged = mergeLoadedChildren(next, prev);
    expect(merged[0].children?.map((c) => c.id)).toEqual(['filelist:zone:z000']);
    expect(merged[0].children?.[0].children?.[0].label).toBe('loaded');
  });

  it('returns the new roots untouched when there is nothing loaded', () => {
    const next: SideNode[] = [{ id: 'filelist:root|', label: 'filelistu' }];
    expect(mergeLoadedChildren(next, [])).toEqual(next);
  });
});

describe('buildNodeIndex', () => {
  it('indexes nested nodes so the keyboard can resolve [data-node-id]', () => {
    const index = buildNodeIndex([imageTree], []);
    expect(index.get('category')).toBe(imageTree);
    expect(index.get('directory')?.id).toBe('directory');
    expect(index.get('leaf:b')?.entry?.id).toBe('b');
    expect(index.get('missing')).toBeUndefined();
  });

  it('merges data/ roots with .vbf roots', () => {
    const vbfRoot: SideNode = { id: 'vbf-root', label: 'mod.vbf', vbf: true, vbfRoot: '/mod.vbf' };
    const index = buildNodeIndex([imageTree], [vbfRoot]);
    expect(index.size).toBe(5);
    expect(index.get('vbf-root')?.vbfRoot).toBe('/mod.vbf');
  });
});

describe('buildContentRoots', () => {
  it('groups text kinds under Texto and leaves Imagens as its own root', () => {
    const events: SideNode = { id: 'kind:events', label: 'Eventos', kind: 'events' };
    const objects: SideNode = { id: 'kind:objects', label: 'Sistema', kind: 'objects' };
    const images: SideNode = { id: 'kind:images', label: 'Imagens', kind: 'images' };

    const roots = buildContentRoots([events, objects, images]);

    expect(roots).toEqual([
      { id: 'kind:text', label: 'Texto', children: [events, objects] },
      images,
    ]);
    expect(roots[0].kind).toBeUndefined();
    expect(roots[1].kind).toBe('images');

    const index = buildNodeIndex(roots, []);
    expect(index.get('kind:text')?.children).toEqual([events, objects]);
    expect(index.get('kind:images')).toBe(images);
  });

  it('does not create an empty Texto root if there are no text kinds', () => {
    const images: SideNode = { id: 'kind:images', label: 'Imagens', kind: 'images' };
    expect(buildContentRoots([images])).toEqual([images]);
  });
});

describe('ancestorPathOf', () => {
  const events: SideNode = {
    id: 'kind:events',
    label: 'Eventos (1)',
    kind: 'events',
    children: [
      {
        id: 'group:events:azit',
        label: 'Home (az)',
        kind: 'events',
        children: [
          {
            id: 'leaf:events:azit0000',
            label: 'azit0000',
            kind: 'events',
            entry: { kind: 'events', id: 'azit0000', key: 'k', label: 'azit0000' },
          },
        ],
      },
    ],
  };

  it('walks from the kind root down to the leaf, inclusive — for the reveal', () => {
    expect(ancestorPathOf([events], 'leaf:events:azit0000')).toEqual([
      'kind:events',
      'group:events:azit',
      'leaf:events:azit0000',
    ]);
  });

  it('handles depth-1 leaves and reports missing nodes as an empty path', () => {
    expect(ancestorPathOf([imageTree], 'leaf:b')).toEqual(['category', 'directory', 'leaf:b']);
    expect(ancestorPathOf([events], 'leaf:events:missing')).toEqual([]);
    expect(ancestorPathOf([], 'leaf:events:azit0000')).toEqual([]);
  });
});
