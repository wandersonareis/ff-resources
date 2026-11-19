'use client';

import { cn } from 'cn';
import { AlertTriangle, Check, ChevronRight, Copy, Info, X } from 'lucide-react';
import { useState } from 'react';
import type * as dto from '@ff-resources/ui/dto';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import { ImportDestinationOptions } from '@/components/dialogs/import-destination-options';
import { allSelectedState, plannedWrites } from './import-selection';

interface ImportImagesDetailedDialogProps {
  open: boolean;
  preview: dto.ImageImportPreview | null;
  busy: boolean;
  options: dto.ImageImportOptions;
  onOptionsChange: (next: dto.ImageImportOptions) => void;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  onToggleAll: () => void;
  onToggleEntry: (entryId: string) => void;
  onToggleCopy: (entryId: string, copyId: string) => void;
}

/**
 * Estado visual do checkbox de uma linha: o checkbox controla SÓ a imagem,
 * mas o desenho é o agregado imagem + cópias — por isso os 3 estados
 * (indeterminate quando um lado tem marcações e o outro não).
 */
function rowCheckboxState(entry: dto.ImageImportPreviewEntry): boolean | 'indeterminate' {
  const copies = entry.copies ?? [];
  const marked = copies.filter((copy) => copy.selected).length;
  if (copies.length === 0) return entry.selected;
  if (entry.selected && marked === copies.length) return true;
  if (!entry.selected && marked === 0) return false;
  return 'indeterminate';
}

/**
 * Linha do modal: caminho relativo à raiz da part em translated/images
 * (<container-rel>/<registro>.dds), checkbox da imagem e o collapse com as
 * cópias ausentes (vazio no FF13 até existir índice de duplicatas).
 */
function EntryRow({
  entry,
  busy,
  onToggleEntry,
  onToggleCopy,
}: {
  entry: dto.ImageImportPreviewEntry;
  busy: boolean;
  onToggleEntry: (entryId: string) => void;
  onToggleCopy: (entryId: string, copyId: string) => void;
}) {
  const [copyOpen, setCopyOpen] = useState(false);
  const copies = entry.copies ?? [];
  const hasError = Boolean(entry.validation_error);
  const identical = !entry.differs;

  return (
    <div className="space-y-1">
      <div
        data-identical={identical ? 'true' : 'false'}
        className={cn(
          'flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors',
          hasError && 'bg-destructive/5 text-destructive',
          identical && !hasError && 'bg-muted/40 text-muted-foreground',
          entry.selected && !identical && !hasError && 'bg-primary/5',
        )}
      >
        <Checkbox
          id={`img-${entry.id}`}
          checked={rowCheckboxState(entry)}
          disabled={hasError || busy}
          onCheckedChange={() => onToggleEntry(entry.id)}
          aria-label={`Importar ${entry.display_path}`}
        />
        <Label
          htmlFor={`img-${entry.id}`}
          className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-sm font-normal"
        >
          <span className="truncate font-mono">{entry.display_path}</span>
          {entry.copy_count > 0 ? (
            <Badge variant="outline" className="shrink-0 gap-1">
              <Copy size={10} />
              {entry.copy_count} cópia{entry.copy_count > 1 ? 's' : ''}
            </Badge>
          ) : null}
          {entry.copy_missing_count > 0 ? (
            <Badge variant="outline" className="shrink-0 border-warning/50 text-warning">
              {entry.copy_missing_count} ausente
              {entry.copy_missing_count > 1 ? 's' : ''}
            </Badge>
          ) : null}
          {hasError ? (
            <Badge variant="destructive" className="shrink-0 gap-1">
              <AlertTriangle size={10} />
              Erro
            </Badge>
          ) : null}
        </Label>
      </div>

      {hasError ? (
        <p className="ml-7 break-all pl-1 text-xs text-destructive">{entry.validation_error}</p>
      ) : null}

      {copies.length > 0 ? (
        <Collapsible open={copyOpen} onOpenChange={setCopyOpen}>
          <CollapsibleTrigger
            className="ml-7 flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground data-[state=open]:text-foreground"
            aria-label={`Cópias ausentes de ${entry.display_path}`}
          >
            <ChevronRight
              size={12}
              className={cn('transition-transform', copyOpen && 'rotate-90')}
            />
            Cópias ausentes na pasta traduzida ({copies.length})
          </CollapsibleTrigger>
          <CollapsibleContent className="ml-7 mt-1 space-y-1 border-l pl-3">
            {copies.map((copy) => (
              <div key={copy.id} className="flex items-center gap-2 text-sm">
                <Checkbox
                  id={`copy-${entry.id}-${copy.id}`}
                  checked={copy.selected}
                  disabled={busy}
                  onCheckedChange={() => onToggleCopy(entry.id, copy.id)}
                  aria-label={`Importar cópia ${copy.display_path}`}
                />
                <Label
                  htmlFor={`copy-${entry.id}-${copy.id}`}
                  className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 font-normal text-muted-foreground"
                >
                  <span className="truncate font-mono">{copy.display_path}</span>
                  {copy.in_mods ? (
                    <Badge variant="outline" className="shrink-0">
                      em mods/
                    </Badge>
                  ) : null}
                  {copy.identical ? (
                    <Badge variant="outline" className="shrink-0 border-success/50 text-success">
                      sincronizada
                    </Badge>
                  ) : null}
                </Label>
              </div>
            ))}
          </CollapsibleContent>
        </Collapsible>
      ) : null}
    </div>
  );
}

/**
 * Sumário do import de imagens: varredura de mods/translated/images/<part>
 * com uma linha por .dds encontrado (pasta = caminho do container, arquivo =
 * nome do registro). Só a SELEÇÃO é importada; as iguais ao que já está em
 * mods/ nascem desmarcadas e em cor distinta — marcá-las regrava o payload.
 */
export function ImportImagesDetailedDialog({
  open,
  preview,
  busy,
  options,
  onOptionsChange,
  onOpenChange,
  onConfirm,
  onToggleAll,
  onToggleEntry,
  onToggleCopy,
}: ImportImagesDetailedDialogProps) {
  const planned = preview ? plannedWrites(preview) : 0;
  const hasDestination = options.write_mods || options.insert_filelist;

  if (!preview) {
    return (
      <Dialog
        open={open}
        onOpenChange={(o) => {
          if (!o || !busy) onOpenChange(o);
        }}
      >
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>Importar imagens traduzidas</DialogTitle>
            <DialogDescription>Varrendo mods/translated/images…</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" disabled={busy} onClick={() => onOpenChange(false)}>
              Cancelar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  const entries = preview.entries;
  const errors = preview.errors ?? [];
  const nothingNew = preview.new_count === 0 && preview.max > 0;
  const allState = allSelectedState(preview);

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o || !busy) onOpenChange(o);
      }}
    >
      <DialogContent className="flex max-h-[85vh] flex-col overflow-hidden sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Importar imagens traduzidas</DialogTitle>
          <DialogDescription>
            Confira o que será importado: só as imagens marcadas são convertidas. Escolha abaixo se
            o binário vai para mods/ (Comprimir) e/ou para o filelist (Inserir). Os caminhos são
            relativos a mods/translated/images/&lt;part&gt; (pasta = caminho do container, arquivo =
            nome do registro).
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
          <ImportDestinationOptions
            options={options}
            onChange={onOptionsChange}
            disabled={busy}
          />

          {/* Origem — exibida uma vez, completa; as linhas ficam relativas. */}
          <div className="rounded-md border border-muted bg-muted/40 p-2 text-sm">
            <p className="font-medium text-muted-foreground">Origem:</p>
            <p className="break-all font-mono text-xs">{preview.root}</p>
          </div>

          {/* Todas (3 estados) + contagens. */}
          <div className="flex flex-wrap items-center gap-2 rounded-md border bg-popover px-2 py-1.5">
            <Checkbox
              id="img-import-all"
              checked={allState}
              disabled={busy}
              onCheckedChange={() => onToggleAll()}
              aria-label="Marcar todas as imagens"
            />
            <Label htmlFor="img-import-all" className="cursor-pointer text-sm font-medium">
              Todas
            </Label>
            <div className="ml-auto flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
              <span>{preview.max} candidato(s)</span>
              <span className="font-medium text-success">{preview.new_count} nova(s)</span>
              <span>{preview.identical_count} igual(is) ao payload atual</span>
              {errors.length > 0 ? (
                <span className="font-medium text-destructive">{errors.length} com erro</span>
              ) : null}
            </div>
          </div>

          {/* Nada novo: aviso pulsante (mesmo efeito do sumário de texto). */}
          {nothingNew ? (
            <div className="motion-safe:animate-pulse rounded-md border border-warning/50 bg-warning/10 p-2 text-sm text-warning">
              <p className="flex items-center gap-1 font-semibold">
                <AlertTriangle size={14} />
                Nada novo para importar.
              </p>
              <p className="mt-0.5 text-xs">
                Todos os .dds da pasta são iguais ao payload atual (mods/ ou data/). As linhas em
                cinza podem ser marcadas para forçar a regravação do payload.
              </p>
            </div>
          ) : null}

          {/* Lista PLANA: uma linha por imagem encontrada, sem agrupamento. */}
          {entries.length === 0 ? (
            <div className="rounded-md border border-muted bg-muted/40 p-2 text-sm">
              Nenhum .dds encontrado nessa raiz.
            </div>
          ) : (
            <ScrollArea className="max-h-[46vh] rounded-md border p-2">
              <div className="space-y-1">
                {entries.map((entry) => (
                  <EntryRow
                    key={entry.id}
                    entry={entry}
                    busy={busy}
                    onToggleEntry={onToggleEntry}
                    onToggleCopy={onToggleCopy}
                  />
                ))}
              </div>
            </ScrollArea>
          )}

          {/* Erros que impedem o import (formato DX, dimensões, mipmaps, IO). */}
          {errors.length > 0 ? (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-2 text-sm text-destructive">
              <p className="flex items-center gap-1 font-semibold">
                <X size={14} />
                Texturas que não serão importadas ({errors.length}):
              </p>
              <ScrollArea className="mt-1 max-h-40">
                <ul className="list-disc space-y-1 pl-5">
                  {errors.map((e) => (
                    <li key={e} className="break-all">
                      {e}
                    </li>
                  ))}
                </ul>
              </ScrollArea>
            </div>
          ) : null}

          {/* Legenda: o que cada cor significa. */}
          <div className="rounded-md border border-info/50 bg-info/10 p-3 text-sm text-info">
            <p className="flex items-center gap-1 font-semibold">
              <Info size={14} />O que as cores dizem:
            </p>
            <ul className="mt-1.5 list-disc space-y-1 text-xs">
              <li>
                <span className="rounded bg-primary/10 px-1 font-mono">branco</span> = imagem
                diferente do payload atual (mods/ ou data/) — nasce marcada.
              </li>
              <li>
                <span className="rounded bg-muted px-1 font-mono text-muted-foreground">cinza</span>{' '}
                = imagem <strong>igual ao payload atual</strong> (sem alteração desde o último
                import) — nasce desmarcada. Marcar força uma nova importação, regravando o payload.
              </li>
              <li>
                <span className="rounded bg-destructive/10 px-1 font-mono text-destructive">
                  vermelho
                </span>{' '}
                = erro de validação (container fora do filelist, imagem inexistente no container,
                dimensões/formato DX/mipmaps divergentes ou IO) — não pode ser importada; corrija o
                caminho ou o arquivo na pasta traduzida.
              </li>
            </ul>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" disabled={busy} onClick={() => onOpenChange(false)}>
            Cancelar
          </Button>
          <Button disabled={busy || planned === 0 || !hasDestination} onClick={onConfirm}>
            {busy ? (
              'Importando…'
            ) : (
              <>
                <Check size={16} className="mr-1" />
                Importar {planned}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
