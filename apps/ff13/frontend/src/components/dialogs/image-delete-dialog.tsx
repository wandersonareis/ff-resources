'use client';

import { AlertTriangle, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { ImageDeletePlans } from '@/bindings/ffxresources/imageservice';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { parseError } from '@/lib/ffx/error-handler';
import { loggedToast as toast } from '@/lib/ffx/toast-logged';
import { deleteImages } from '@/lib/ffx/tree-data';
import {
  type ActionProps,
  DeleteIrreversibleAlert,
  type DeleteScope,
  DeleteScopeField,
  reportBatch,
  useActionDuplicates,
} from './image-action-shared';

/**
 * Deletar: escopo explícito (nunca cascata implícita) + cópias só se
 * marcadas, com o aviso de irreversível antes de qualquer gravação. Sempre
 * leva junto os .dds/.png extraídos — senão Resolve serviria imagem órfã.
 */
export function DeleteImageDialog({ view, action }: ActionProps) {
  const { version, actions } = view;
  const list = useActionDuplicates(view, action);
  const [scope, setScope] = useState<DeleteScope>('both');
  const [withCopies, setWithCopies] = useState(false);
  const [busy, setBusy] = useState(false);
  const copies = list ?? [];
  const targets = withCopies ? copies.map((d) => d.id) : [];
  const divergent = copies.filter((d) => !d.identical).length;
  const close = () => actions.closeImageAction();
  const [plans, setPlans] = useState<
    Array<{ warning?: string; files?: Array<{ path: string; scope: string }> | null }>
  >([]);

  useEffect(() => {
    let dead = false;
    void ImageDeletePlans(version, [action.id])
      .then((res) => {
        if (!dead) setPlans((res ?? []) as typeof plans);
      })
      .catch(() => {
        if (!dead) setPlans([]);
      });
    return () => {
      dead = true;
    };
  }, [action.id, version]);

  const deleteFiles = plans
    .flatMap((p) => p.files ?? [])
    .filter((file) => scope === 'both' || file.scope === scope);
  const deleteWarning = plans.find((p) => p.warning)?.warning;

  const run = async () => {
    setBusy(true);
    try {
      const res = await deleteImages(action.id, targets, scope, version);
      reportBatch(res, 'Apagada', 'Apagadas');
      await actions.reload();
      close();
    } catch (error) {
      toast.error(parseError(error));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) close();
      }}
    >
      <DialogContent
        className="max-h-[calc(100vh-2rem)] min-w-0 overflow-y-auto sm:max-w-lg"
        onKeyDown={(event) => {
          if (event.key === 'Enter' && !busy && list !== null) {
            event.preventDefault();
            void run();
          }
        }}
      >
        <DialogHeader>
          <DialogTitle>Deletar {action.label}?</DialogTitle>
          <DialogDescription>
            Escolha o escopo — o aviso abaixo lista exatamente o que sai do disco.
          </DialogDescription>
        </DialogHeader>

        <DeleteIrreversibleAlert scope={scope} />

        <DeleteScopeField
          scope={scope}
          onChange={setScope}
          legend="Escopo"
          name="image-delete-scope"
        />

        {deleteWarning ? (
          <Alert variant="warning">
            <AlertTriangle />
            <AlertTitle>Container inteiro</AlertTitle>
            <AlertDescription>{deleteWarning}</AlertDescription>
          </Alert>
        ) : null}

        {deleteFiles.length > 0 ? (
          <div className="max-h-32 overflow-y-auto rounded border p-2 text-xs">
            <p className="mb-1 font-medium">Arquivos que serão apagados:</p>
            <ul className="space-y-0.5">
              {deleteFiles.map((file) => (
                <li key={`${file.scope}:${file.path}`} className="break-all opacity-80">
                  <span className="mr-1 uppercase opacity-60">{file.scope}</span>
                  {file.path}
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {copies.length > 0 ? (
          <>
            <div className="flex items-center gap-2">
              <Checkbox
                id="delete-copies"
                checked={withCopies}
                onCheckedChange={(value) => setWithCopies(value === true)}
              />
              <Label htmlFor="delete-copies" className="cursor-pointer text-sm">
                Apagar também as {copies.length} cópia
                {copies.length > 1 ? 's' : ''}
              </Label>
            </div>
            {withCopies && divergent > 0 ? (
              <Alert variant="warning">
                <AlertTriangle />
                <AlertTitle>
                  {divergent} cópia{divergent > 1 ? 's' : ''} diverge
                  {divergent > 1 ? 'm' : ''} do original
                </AlertTitle>
                <AlertDescription>
                  Estão divergentes porque foram editadas em separado — a edição delas também será
                  apagada.
                </AlertDescription>
              </Alert>
            ) : null}
          </>
        ) : (
          <p className="text-sm opacity-70">Imagem única: sem cópias.</p>
        )}

        <DialogFooter>
          <Button variant="outline" disabled={busy} onClick={close}>
            Cancelar
          </Button>
          <Button variant="destructive" disabled={busy || list === null} onClick={() => void run()}>
            <Trash2 size={16} />
            {busy
              ? 'Apagando…'
              : `Deletar ${targets.length + 1} textura${targets.length > 0 ? 's' : ''}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

