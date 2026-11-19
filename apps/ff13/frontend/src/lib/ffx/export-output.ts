import { useSelector } from '@tanstack/react-store';
import { createStore } from '@tanstack/store';

/**
 * Saídas do Exportar, compartilhadas por TODAS as ações de exportação
 * (header, painel de imagem e menu de contexto):
 *
 *   decode  = Descompactar (artefato de texto / DDS+PNG)
 *   extract = Extrair (binário real do filelist para data/)
 *
 * Descompactar nasce ligado; Extrair desligado. Sem nenhum, Exportar fica
 * desabilitado. O estado é de sessão (não persiste entre execuções).
 */
export interface ExportOutputSnapshot {
  decode: boolean;
  extract: boolean;
}

const store = createStore<ExportOutputSnapshot>({ decode: true, extract: false });

class ExportOutputStore {
  getSnapshot(): ExportOutputSnapshot {
    return store.state;
  }

  set(next: Partial<ExportOutputSnapshot>): void {
    store.setState((prev) => ({ ...prev, ...next }));
  }

  decodeOf(): boolean {
    return store.state.decode;
  }

  extractOf(): boolean {
    return store.state.extract;
  }

  hasAny(): boolean {
    return store.state.decode || store.state.extract;
  }
}

export const exportOutput = new ExportOutputStore();

/** Snapshot reativo (rerender ao marcar/desmarcar). */
export function useExportOutput(): ExportOutputSnapshot {
  return useSelector(store);
}
