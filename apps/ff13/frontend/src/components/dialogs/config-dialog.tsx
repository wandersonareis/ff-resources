'use client';

import { useEffect, useState } from 'react';
import { HasUnsavedEdits } from '@/bindings/ffxresources/app';
import { ResolveEnableModsChange } from '@/bindings/ffxresources/configservice';
import {
  type ModsGuardDecision,
  ModsToggleGuardDialog,
} from '@/components/dialogs/mods-toggle-guard-dialog';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { commitEnableMods, useAppConfig } from '@/lib/ffx/config-store';
import { editDraft } from '@/lib/ffx/edit-draft';
import { loggedToast as toast } from '@/lib/ffx/toast-logged';
import { type LocalConfigSnapshot, loadLocalConfig, localRoot } from '@/lib/parts/local-config';

export function ConfigDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-160">
        {/* Header fica AQUI (radix DialogTitle exige o root Dialog); na
            janela filha o mesmo papel é do titlebar do WindowChrome. */}
        <DialogHeader>
          <DialogTitle>Configurações</DialogTitle>
          <DialogDescription className="sr-only">Preferências do app</DialogDescription>
        </DialogHeader>
        <ConfigDialogContent />
      </DialogContent>
    </Dialog>
  );
}

/**
 * Corpo do diálogo — compartilhado pelo modal (browser/server mode) e
 * pela janela filha /w/#config: mesma leitura e mesmo toggle.
 *
 * O FF13 não tem os campos do FFX (executável do jogo, pasta de
 * arquivos originais, pasta de translated): a instalação vem do
 * ff13.config.json em DISCO (a interface só lê) e a única preferência do
 * app é o toggle de mods.
 *
 * O toggle de mods com rascunho sujo passa pela guarda
 * (ModsToggleGuardDialog) antes de aplicar: o toggle recarrega todo o
 * conteúdo, então salvar/descartar é decisão explícita do usuário.
 */
export function ConfigDialogContent() {
  const config = useAppConfig();

  // Config local (ff13.config.json): caminho do arquivo e pasta de instalação
  // do jogo da aba ativa. É LEITURA — quem escreve é o arquivo em disco.
  const [local, setLocal] = useState<LocalConfigSnapshot | null>(null);
  useEffect(() => {
    let alive = true;
    void loadLocalConfig()
      .then((cfg) => {
        if (alive) setLocal(cfg);
      })
      .catch(() => {
        /* config ausente: o campo fica vazio */
      });
    return () => {
      alive = false;
    };
  }, []);
  const installedRoot = localRoot(local, config.gameVersion);

  // Toggle de mods pendente na guarda de edições não salvas: o valor que o
  // usuário quer aplicar, null = guarda fechada. Enquanto a guarda decide,
  // NADA é commitado — o checkbox segue exibindo o estado do store (o
  // antigo), e o cancelamento o devolve intacto, sem tocar rascunhos.
  const [pendingMods, setPendingMods] = useState<boolean | null>(null);

  /**
   * Alternar o checkbox de mods (#EnableMods). Sem edições não salvas
   * aplica direto (commit otimista + eco; o eco dispara o coldReload das
   * abas). Com rascunho sujo, abre a guarda: aplicar o toggle recarrega
   * todo o conteúdo e a decisão de salvar/descartar é do usuário.
   *
   * O critério de "sujo" é o MESMO boolean que habilita o botão Salvar
   * da tabela (editDraft) — leitura local e exata na janela principal.
   * A janela FILHA de config (desktop) tem outro editDraft (vazio): lá
   * vale a flag do backend (HasUnsavedEdits), espelho da mesma fonte
   * mantido pelo GameVersionTab via SetUnsavedEdits.
   */
  const onToggleMods = async (next: boolean): Promise<void> => {
    if (next === config.enableMods) return;
    let dirty = editDraft.getSnapshot().hasDirty;
    if (!dirty) {
      dirty = await HasUnsavedEdits().catch(() => false);
    }
    if (!dirty) {
      void commitEnableMods(next);
      return;
    }
    setPendingMods(next);
  };

  /**
   * Decisão da guarda (Salvar/Descartar): viaja pelo evento
   * EnableModsChange e a janela PRINCIPAL — dona dos rascunhos — executa
   * (save/discard + apply). Se o salvar falhar lá, o toggle não é
   * aplicado e o checkbox permanece no estado antigo.
   */
  const resolveModsGuard = (decision: ModsGuardDecision): void => {
    const next = pendingMods;
    setPendingMods(null);
    if (next === null) return;
    void ResolveEnableModsChange(next, decision).catch(notify);
  };

  return (
    <>
      <div className="flex flex-col gap-3 pt-2 min-w-0">
        <div className="grid gap-1.5">
          <Label htmlFor="InstalledGameRoot">Instalação do jogo (config local)</Label>
          <Input
            id="InstalledGameRoot"
            readOnly
            value={installedRoot}
            placeholder="Não configurada"
          />
          <p className="text-xs text-muted-foreground">
            Vem do arquivo de config local{' '}
            <code>{local?.path ? local.path : 'ff13.config.json'}</code> — edite o arquivo para
            apontar a instalação. É de lá que saem os filelists e white bins.
          </p>
        </div>
      </div>

      <div className="flex items-start gap-2 pt-2">
        <Checkbox
          id="EnableMods"
          checked={config.enableMods}
          onCheckedChange={(v) => void onToggleMods(Boolean(v))}
        />
        <div className="grid gap-1 leading-none">
          <Label htmlFor="EnableMods" className="cursor-pointer">
            Preferir binários traduzidos (mods)
          </Label>
          <p className="text-xs text-muted-foreground">
            Ligado, a tabela mostra a tradução de <code className="mx-1">mods/</code>; desligado,
            mostra o original de <code className="mx-1">data/</code>. O export sempre usa o original
            como fonte e a tradução como destino.
          </p>
        </div>
      </div>

      <ModsToggleGuardDialog
        next={pendingMods}
        onDecision={resolveModsGuard}
        onCancel={() => setPendingMods(null)}
      />
    </>
  );
}

/**
 * Erros do seletor nativo: toast com cópia para o log (loggedToast.error
 * já escreve no log do backend via WriteLog — não emitir "Notify", que
 * circularia de volta ao ProgressNotifier).
 */
function notify(error: unknown): void {
  const message = error instanceof Error ? error.message : String(error ?? 'Erro');
  toast.error(message, { duration: 4000 });
}
