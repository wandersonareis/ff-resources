'use client';

import type * as dto from '@ff-resources/ui/dto';
import { CircleHelp } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';

/**
 * Destinos do import de imagem.
 *
 * "Comprimir" converte o DDS/payload para o binário do jogo e grava em
 * mods/. "Inserir" grava esse binário no filelist (append no white bin +
 * reaponta os ponteiros) — a instalação do jogo é alterada; /data não.
 *
 * Os dois são independentes; sem nenhum marcado não há destino.
 */
export function ImportDestinationOptions({
  options,
  onChange,
  disabled,
  compact = false,
  idPrefix = 'import-dest',
}: {
  options: dto.ImportDestination;
  onChange: (next: dto.ImportDestination) => void;
  disabled?: boolean;
  /** Versão curta para o dropdown do menu. */
  compact?: boolean;
  /** Prefixo dos ids dos checkboxes (menu e modal podem coexistir). */
  idPrefix?: string;
}) {
  const none = !options.write_mods && !options.insert_filelist;
  const modsId = `${idPrefix}-mods`;
  const filelistId = `${idPrefix}-filelist`;
  return (
    <div className={compact ? 'space-y-1.5 px-2 py-1' : 'space-y-2 rounded-md border p-2'}>
      {!compact ? (
        <p className="text-xs font-medium text-muted-foreground">Destino do import</p>
      ) : null}

      <div className="flex items-start gap-2">
        <Checkbox
          id={modsId}
          checked={options.write_mods}
          disabled={disabled}
          onCheckedChange={(checked) =>
            onChange({ ...options, write_mods: checked === true })
          }
          aria-label="Comprimir: gravar os binários em mods"
        />
        <div className="min-w-0">
          <Label htmlFor={modsId} className="cursor-pointer text-sm font-medium">
            Comprimir
          </Label>
          <p className="text-xs text-muted-foreground">
            converte e grava os binários em mods/
          </p>
        </div>
      </div>

      <div className="flex items-start gap-2">
        <Checkbox
          id={filelistId}
          checked={options.insert_filelist}
          disabled={disabled}
          onCheckedChange={(checked) =>
            onChange({ ...options, insert_filelist: checked === true })
          }
          aria-label="Inserir: gravar os binários no filelist"
        />
        <div className="min-w-0">
          <div className="flex items-center gap-1">
            <Label
              htmlFor={filelistId}
              className="cursor-pointer text-sm font-medium"
            >
              Inserir
            </Label>
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="inline-flex size-4 items-center justify-center rounded-full text-muted-foreground hover:text-foreground"
                  aria-label="O que faz Inserir?"
                >
                  <CircleHelp size={13} />
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-72 text-xs" align="start">
                <p className="font-medium">Inserir no filelist</p>
                <p className="mt-1 text-muted-foreground">
                  Insere os arquivos no filelist (white bin): os dados vão para o fim
                  do bin e os ponteiros do filelist passam a apontar para eles. O
                  /data não é alterado.
                </p>
                <p className="mt-1 text-muted-foreground">
                  A instalação do jogo é modificada. Use &quot;Reverter
                  inserções&quot; no menu Importar para desfazer.
                </p>
              </PopoverContent>
            </Popover>
          </div>
          <p className="text-xs text-muted-foreground">
            insere no filelist (white bin); /data não muda
          </p>
        </div>
      </div>

      {none ? (
        <p className="text-xs text-destructive">Marque Comprimir ou Inserir.</p>
      ) : null}
    </div>
  );
}

