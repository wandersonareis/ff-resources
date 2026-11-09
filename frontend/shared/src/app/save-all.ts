import { getCurrentApi } from '../api/registry';
import type { PrunedRowView } from '../api/appTypes';
import { loggedToast as toast } from './toast-logged';
import type { EntryKind } from './display-names';
import { editDraft } from './edit-draft';
import { sendErrorNotification } from './error-handler';
import { askPruneConfirm } from './prune-confirm-store';

// A chave de versão é o id do app (string): o pacote é agnóstico de jogo.
type GameVersionId = string;

export const SOURCE_LANG = 'us';

let afterSaveHook: (() => void) | null = null;

/** Hook pós-save do app (ex.: invalidar o cache de .vbf do FFX). */
export function setAfterSaveHook(fn: (() => void) | null): void {
  afterSaveHook = fn;
}

/**
 * Linhas que o save vai EXCLUIR por não existirem no original (ponteiro
 * errado no binário traduzido etc.). O dicionário reconstrói o arquivo
 * inteiro, então a prévia vem do backend; nos demais kinds o save só toca as
 * entradas sujas e o próprio rascunho sabe quais linhas são órfãs.
 */
async function collectPruneRows(): Promise<PrunedRowView[]> {
  const api = getCurrentApi();
  const out: PrunedRowView[] = [];
  const seen = new Set<string>();
  const add = (rows: PrunedRowView[]) => {
    for (const row of rows) {
      const key = `${row.file}|${row.block ?? ''}|${row.index}|${row.name}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push(row);
    }
  };

  for (const [version, byKind] of editDraft.dirtyBatches()) {
    for (const kind of byKind.keys()) {
      if (kind === 'macro') {
        add(await api.previewSavePrune(kind, version));
        continue;
      }
      add(editDraft.orphanRows(version, kind));
    }
  }
  for (const [version, byRoot] of editDraft.dirtyVbfBatches()) {
    for (const byKind of byRoot.values()) {
      for (const kind of byKind.keys()) add(editDraft.orphanRows(version, kind));
    }
  }
  return out;
}

export async function saveAllDrafts(
  isSaving: () => boolean,
  setSaving: (value: boolean) => void,
): Promise<boolean> {
  if (!editDraft.getSnapshot().hasDirty || isSaving()) return false;
  // Gate de tags protegidas: restaura o que sumiu e aborta se ainda houver
  // valor perdido ou `{…` aberto — o texto não vai ao binário quebrado.
  const report = editDraft.restoreProtectedTags();
  if (report.restored > 0) {
    toast(`Tags protegidas restauradas em ${report.restored} célula(s).`, { duration: 4000 });
  }
  if (report.unresolved > 0 || report.unclosed > 0) {
    const parts: string[] = [];
    if (report.unresolved > 0) {
      parts.push(`${report.unresolved} célula(s) sem tag protegida obrigatória`);
    }
    if (report.unclosed > 0) {
      parts.push(`${report.unclosed} célula(s) com tag aberta ({…)`);
    }
    toast.error(
      `Salvamento bloqueado: ${parts.join(' e ')}. Abra a linha e use Restaurar/Corrigir.`,
      { duration: 8000 },
    );
    return false;
  }

  // Poda do save: avisa (lista completa) antes de excluir qualquer linha.
  // Cancelar aborta o save inteiro — nada é gravado.
  const pruned = await collectPruneRows();
  if (pruned.length > 0) {
    const proceed = await askPruneConfirm(pruned);
    if (!proceed) return false;
  }

  const api = getCurrentApi();
  setSaving(true);
  try {
    for (const [version, byKind] of editDraft.dirtyBatches()) {
      for (const kind of byKind.keys()) {
        const collection = editDraft.buildCollection(version as GameVersionId, kind as EntryKind);
        if (Object.keys(collection).length === 0) continue;
        await api.applyCollection(kind, collection as never, version);
      }
    }
    // Tabelas abertas pelo .vbf: mesmo motor, escopo da sessão do container.
    for (const [version, byRoot] of editDraft.dirtyVbfBatches()) {
      for (const [root, byKind] of byRoot) {
        for (const kind of byKind.keys()) {
          const collection = editDraft.buildSourceCollection(
            version as GameVersionId,
            kind as EntryKind,
            root,
          );
          if (Object.keys(collection).length === 0) continue;
          await api.applyVbfCollection(root, kind, version, collection as never);
        }
      }
    }
    editDraft.markSaved();
    afterSaveHook?.();
    toast('Alterações salvas no jogo.', { duration: 3000 });
    return true;
  } catch (error) {
    sendErrorNotification(error);
    return false;
  } finally {
    setSaving(false);
  }
}
