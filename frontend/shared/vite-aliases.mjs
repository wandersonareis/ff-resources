import path from 'node:path';

/**
 * Aliases do pacote local @ff-resources/ui, compartilhados pelo Vite e pelo
 * Vitest dos apps ffx e ff13.
 *
 * `appDir` = diretório do frontend do app (apps/<app>/frontend).
 * Os aliases do pacote vêm ANTES do alias genérico '@', porque o alias é
 * prefixo: '@' sozinho capturaria '@/components/ui' e quebraria o import.
 */
export function sharedAliases(appDir) {
  const shared = path.resolve(appDir, '../../../frontend/shared/src');
  return {
    'next-themes': path.resolve(shared, 'theme.tsx'),
    '@ff-resources/ui': shared,
    '@/components/ui': path.resolve(shared, 'components/ui'),
    '@/components/editor': path.resolve(shared, 'editor'),
    '@/lib/utils': path.resolve(shared, 'lib/utils.ts'),
    '@/lib/ffx/game-text-parser': path.resolve(shared, 'editor/game-text-parser.ts'),
    '@/lib/ffx/game-text-format': path.resolve(shared, 'editor/game-text-format.ts'),
    '@/lib/ffx/game-text-tags': path.resolve(shared, 'editor/game-text-tags.ts'),
    '@/lib/ffx/plain-text': path.resolve(shared, 'editor/plain-text.ts'),
    '@/lib/ffx/find-in-text': path.resolve(shared, 'editor/find-in-text.ts'),
    '@/lib/ffx/protected-tags': path.resolve(shared, 'editor/protected-tags.ts'),
    '@/lib/ffx/leading-padding': path.resolve(shared, 'editor/leading-padding.ts'),
    '@/lib/ffx/blank-us': path.resolve(shared, 'editor/blank-us.ts'),
    '@/lib/ffx/bytes': path.resolve(shared, 'editor/bytes.ts'),
    '@/lib/ffx/clipboard': path.resolve(shared, 'editor/clipboard.ts'),
    '@/lib/ffx/tag-catalog': path.resolve(shared, 'editor/tag-catalog.ts'),
    '@/lib/ffx/statusbar-store': path.resolve(shared, 'app/statusbar-store.ts'),
    '@/lib/ffx/progress-store': path.resolve(shared, 'app/progress-store.ts'),
    '@/lib/ffx/use-wails-event': path.resolve(shared, 'app/use-wails-event.ts'),
    '@/lib/ffx/shortcuts': path.resolve(shared, 'app/shortcuts.ts'),
    '@/lib/ffx/layout-storage': path.resolve(shared, 'app/layout-storage.ts'),
    '@/lib/ffx/tab-storage': path.resolve(shared, 'app/tab-storage.ts'),
    '@/lib/ffx/use-persisted-layout': path.resolve(shared, 'app/use-persisted-layout.ts'),
    '@/lib/ffx/error-handler': path.resolve(shared, 'app/error-handler.ts'),
    '@/lib/ffx/hash-ref': path.resolve(shared, 'app/hash-ref.ts'),
    '@/lib/ffx/find-store': path.resolve(shared, 'app/find-store.ts'),
    '@/lib/ffx/display-names': path.resolve(shared, 'app/display-names.ts'),
    '@/lib/ffx/content-tree/model': path.resolve(shared, 'app/content-tree/model.ts'),
    '@/lib/ffx/tree-data': path.resolve(shared, 'app/tree-data.ts'),
    '@/lib/ffx/export-selection': path.resolve(shared, 'app/export-selection.ts'),
    '@/lib/ffx/toast-logged': path.resolve(shared, 'app/toast-logged.ts'),
    '@/lib/ffx/prune-confirm-store': path.resolve(shared, 'app/prune-confirm-store.ts'),
    '@/lib/ffx/edit-draft': path.resolve(shared, 'app/edit-draft.ts'),
    '@/lib/ffx/save-all': path.resolve(shared, 'app/save-all.ts'),
    '@/components/game-text-view': path.resolve(shared, 'components/game-text-view.tsx'),
    '@/components/progress/progress-modal': path.resolve(
      shared,
      'components/progress/progress-modal.tsx',
    ),
    '@/components/entry-view/entry-table/entry-table-empty': path.resolve(
      shared,
      'components/entry-view/entry-table/entry-table-empty.tsx',
    ),
    '@/components/entry-view/content-tree/search-control': path.resolve(
      shared,
      'components/entry-view/content-tree/search-control.tsx',
    ),
    '@/components/entry-view/content-tree/tree-context': path.resolve(
      shared,
      'components/entry-view/content-tree/tree-context.tsx',
    ),
    '@/components/entry-view/content-tree/tree-item': path.resolve(
      shared,
      'components/entry-view/content-tree/tree-item.tsx',
    ),
    '@/components/entry-view/content-tree/tree-list': path.resolve(
      shared,
      'components/entry-view/content-tree/tree-list.tsx',
    ),
    '@/lib/ffx/vbf-selection': path.resolve(shared, 'app/vbf-selection.ts'),
    '@/lib/ffx/entry-progress': path.resolve(shared, 'app/entry-progress.ts'),
    '@/components/entry-view/entry-table/table-types': path.resolve(
      shared,
      'components/entry-view/entry-table/table-types.ts',
    ),
    '@/components/entry-view/entry-table/entry-table-header': path.resolve(
      shared,
      'components/entry-view/entry-table/entry-table-header.tsx',
    ),
    '@/components/entry-view/entry-table/entry-table-body': path.resolve(
      shared,
      'components/entry-view/entry-table/entry-table-body.tsx',
    ),
    '@/components/entry-view/entry-table/table-bar': path.resolve(
      shared,
      'components/entry-view/entry-table/table-bar.tsx',
    ),
    '@/components/entry-view/entry-table/columns/editable-cell': path.resolve(
      shared,
      'components/entry-view/entry-table/columns/editable-cell.tsx',
    ),
    '@/components/entry-view/entry-table/columns/index-column': path.resolve(
      shared,
      'components/entry-view/entry-table/columns/index-column.tsx',
    ),
    '@/components/entry-view/entry-table/columns/linked-cell': path.resolve(
      shared,
      'components/entry-view/entry-table/columns/linked-cell.tsx',
    ),
    '@/components/entry-view/entry-table/columns/name-column': path.resolve(
      shared,
      'components/entry-view/entry-table/columns/name-column.tsx',
    ),
    '@/components/entry-view/entry-table/columns/original-column': path.resolve(
      shared,
      'components/entry-view/entry-table/columns/original-column.tsx',
    ),
    '@/components/entry-view/image-panel/use-image-preview': path.resolve(
      shared,
      'components/entry-view/image-panel/use-image-preview.ts',
    ),
    '@/components/entry-view/image-panel/image-preview': path.resolve(
      shared,
      'components/entry-view/image-panel/image-preview.tsx',
    ),
  };
}
