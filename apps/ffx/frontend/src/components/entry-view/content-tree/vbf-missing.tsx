'use client';

import { useSelector } from '@tanstack/react-store';
import { Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { openConfigDialog } from '@/lib/ffx/config-dialog-store';
import { configStore } from '@/lib/ffx/config-store';

/**
 * Aviso exibido no lugar do grupo de containers .vbf quando nenhuma raiz
 * chegou. É específico do FFX (o FF13 não usa .vbf) e fica no app; o TreeList
 * compartilhado recebe este conteúdo pelo `emptyContent` do grupo.
 */
export function VbfMissing() {
  const gameExe = useSelector(configStore, (s) => s.gameExeLocation);
  if (gameExe !== '') {
    return (
      <p
        data-vbf-missing=""
        className="mx-2 mt-1 rounded-md border border-border px-2 py-1.5 text-xs text-muted-foreground"
      >
        Nenhum <code>.vbf</code> encontrado ao lado do executável configurado.
      </p>
    );
  }
  return (
    <div
      data-vbf-missing=""
      role="alert"
      className="mx-2 mt-1 flex flex-col gap-1.5 rounded-md border border-destructive/40 bg-destructive/10 px-2 py-1.5 text-xs text-destructive"
    >
      <span className="font-medium">Nenhum container .vbf</span>
      <span>
        Aponte o executável do jogo (FFX.exe ou FFX-2.exe) nas Configurações — dele saem os
        containers exibidos na árvore.
      </span>
      <Button size="sm" variant="outline" className="h-7 w-fit" onClick={openConfigDialog}>
        <Settings size={14} />
        Abrir Configurações
      </Button>
    </div>
  );
}
