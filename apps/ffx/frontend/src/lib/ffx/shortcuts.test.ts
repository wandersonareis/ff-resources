import { describe, expect, it } from 'vitest';
import { GAME_VERSIONS } from './game-version';
import { SHORTCUTS, versionTabKeys } from '@/lib/ffx/shortcuts';

// Uma tecla de aba por versão: o registro (app-shell) e o tooltip por aba
// derivam do MESMO array, então este é o único acoplamento que precisa bater.
describe('shortcuts', () => {
  it('tem uma tecla de aba por versão de GAME_VERSIONS', () => {
    expect(versionTabKeys(GAME_VERSIONS.length)).toHaveLength(GAME_VERSIONS.length);
  });

  it('o catálogo canônico cobre todas as versões', () => {
    expect(SHORTCUTS.versionTabs.keys.length).toBeGreaterThanOrEqual(GAME_VERSIONS.length);
  });
});
