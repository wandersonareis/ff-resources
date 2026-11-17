import { describe, expect, it } from 'vitest';
import { exportSelection } from '@/lib/ffx/export-selection';

describe('export selection snapshot', () => {
  it('trocar o formato preserva a identidade de byKind', () => {
    const before = exportSelection.getSnapshot();
    const other = before.format === 'json' ? 'strings' : 'json';

    exportSelection.setFormat(other);
    const after = exportSelection.getSnapshot();

    expect(after.format).toBe(other);
    expect(after.byKind).toBe(before.byKind);
    expect(after.revision).toBeGreaterThan(before.revision);

    // Volta ao estado anterior para não vazar para os demais testes.
    exportSelection.setFormat(before.format);
    expect(exportSelection.getSnapshot().format).toBe(before.format);
  });

  it('marcar entradas reconstrói byKind (identidade nova)', () => {
    const before = exportSelection.getSnapshot();
    exportSelection.setMany('us', 'events', ['k'], true);
    const after = exportSelection.getSnapshot();
    expect(after.byKind).not.toBe(before.byKind);
    expect(after.byKind.get('us|events')).toEqual(['k']);

    exportSelection.setMany('us', 'events', ['k'], false);
    expect(exportSelection.getSnapshot().byKind.has('us|events')).toBe(false);
  });

  it('marca kinds diferentes numa ÚNICA notificação (grupo do filelist)', () => {
    const before = exportSelection.getSnapshot();
    const entries = [
      { kind: 'events' as const, id: 'a' },
      { kind: 'images' as const, id: 'b' },
    ];

    exportSelection.setManyByKind('us', entries, true);
    const after = exportSelection.getSnapshot();
    expect(after.revision).toBe(before.revision + 1);
    expect(after.byKind.get('us|events')).toEqual(['a']);
    expect(after.byKind.get('us|images')).toEqual(['b']);

    exportSelection.setManyByKind('us', entries, false);
    expect(exportSelection.getSnapshot().byKind.has('us|events')).toBe(false);
    expect(exportSelection.getSnapshot().byKind.has('us|images')).toBe(false);
  });
});
