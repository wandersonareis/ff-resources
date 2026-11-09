import { beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * Store de config do FF13: ele espelha o snapshot do backend (preferência de
 * mods + part ativa). O ponto sensível é o ECO (evento AppConfigChanged):
 * só ele avisa as abas para recarregar o conteúdo — o set otimista do
 * checkbox e a carga inicial não podem disparar o reload.
 */

const { GetConfigSnapshot, SetEnableMods, toastError } = vi.hoisted(() => ({
  GetConfigSnapshot: vi.fn(async () => ({ enable_mods: true, game_version: 'part1' })),
  SetEnableMods: vi.fn(async (_enabled: boolean) => undefined),
  toastError: vi.fn(),
}));

vi.mock('@/bindings/ffxresources/configservice', () => ({
  GetConfigSnapshot,
  SetEnableMods,
}));

vi.mock('@/lib/ffx/toast-logged', () => ({
  loggedToast: Object.assign(vi.fn(), {
    error: toastError,
    info: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
    loading: vi.fn(),
    message: vi.fn(),
  }),
}));

/** Importa o store com o estado de módulo limpo (lastConfirmedMods). */
async function freshStore() {
  vi.resetModules();
  return import('./config-store');
}

beforeEach(() => {
  GetConfigSnapshot.mockClear();
  SetEnableMods.mockClear();
  toastError.mockClear();
});

describe('config-store — snapshot', () => {
  it('aplica o payload cru (snake_case do DTO Go)', async () => {
    const { applyConfigSnapshot, configStore } = await freshStore();

    applyConfigSnapshot({ enable_mods: false, game_version: 'part2' });

    expect(configStore.state.enableMods).toBe(false);
    expect(configStore.state.gameVersion).toBe('part2');
  });

  it('payload inválido não deixa o store pior do que a última verdade', async () => {
    const { applyConfigSnapshot, configStore } = await freshStore();

    applyConfigSnapshot({ enable_mods: false, game_version: 'part3' });
    applyConfigSnapshot(null);
    applyConfigSnapshot('nada');

    expect(configStore.state.enableMods).toBe(false);
    expect(configStore.state.gameVersion).toBe('part3');
  });

  it('a carga inicial busca o snapshot do backend', async () => {
    const { loadConfigSnapshot, configStore } = await freshStore();
    GetConfigSnapshot.mockResolvedValueOnce({ enable_mods: false, game_version: 'part2' });

    await loadConfigSnapshot();

    expect(GetConfigSnapshot).toHaveBeenCalled();
    expect(configStore.state.enableMods).toBe(false);
    expect(configStore.state.gameVersion).toBe('part2');
  });
});

describe('config-store — eco do backend', () => {
  it('o ECO com mudança real avisa os assinantes (a aba recarrega)', async () => {
    const { applyConfigSnapshot, onEnableModsChanged } = await freshStore();
    const listener = vi.fn();
    const off = onEnableModsChanged(listener);

    applyConfigSnapshot({ enable_mods: false, game_version: 'part1' });
    expect(listener).not.toHaveBeenCalled();

    applyConfigSnapshot({ enable_mods: true, game_version: 'part1' }, true);
    expect(listener).toHaveBeenCalledTimes(1);

    off();
  });

  it('o ECO SEM mudança não avisa (o valor não mudou de verdade)', async () => {
    const { applyConfigSnapshot, onEnableModsChanged } = await freshStore();
    const listener = vi.fn();
    const off = onEnableModsChanged(listener);

    applyConfigSnapshot({ enable_mods: true, game_version: 'part1' }, true);
    applyConfigSnapshot({ enable_mods: true, game_version: 'part1' }, true);

    expect(listener).not.toHaveBeenCalled();
    off();
  });
});

describe('config-store — commit da preferência', () => {
  it('o commit é otimista: o estado muda na hora e o backend é chamado', async () => {
    const { commitEnableMods, configStore } = await freshStore();

    await commitEnableMods(false);

    expect(SetEnableMods).toHaveBeenCalledWith(false);
    expect(configStore.state.enableMods).toBe(false);
  });

  it('falha no commit: mostra o erro e volta ao que ficou persistido', async () => {
    const { commitEnableMods, configStore } = await freshStore();
    SetEnableMods.mockRejectedValueOnce(new Error('sem permissão'));
    GetConfigSnapshot.mockResolvedValueOnce({ enable_mods: true, game_version: 'part1' });

    await commitEnableMods(false);

    expect(toastError).toHaveBeenCalledWith('sem permissão', { duration: 4000 });
    // O rollback consulta o backend: a preferência real volta.
    expect(configStore.state.enableMods).toBe(true);
  });
});
