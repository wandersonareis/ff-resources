'use client';

import { CircleHelp } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { exportOutput, useExportOutput } from '@/lib/ffx/export-output';

/**
 * Saídas do Exportar.
 *
 * "Descompactar" = formatos úteis (texto em JSON/Strings/CSV/XLSX/PO; imagem
 * em DDS/PNG, conforme o seletor de formato).
 *
 * "Extrair" = binário REAL do filelist/white bin, descomprimido quando o
 * container está comprimido, gravado em data/partN. Só existe para itens do
 * filelist.
 *
 * Os dois são independentes; sem nenhum, Exportar fica desabilitado.
 */
export function ExportOutputOptions({
  disabled,
  extractDisabled,
  compact = false,
  idPrefix = 'export-out',
}: {
  disabled?: boolean;
  extractDisabled?: boolean;
  /** Versão curta para menus (header, imagem). */
  compact?: boolean;
  /** Prefixo dos ids dos checkboxes (menu e modal podem coexistir). */
  idPrefix?: string;
}) {
  const options = useExportOutput();
  const none = !options.decode && !options.extract;
  const decodeId = `${idPrefix}-decode`;
  const extractId = `${idPrefix}-extract`;

  return (
    <div className={compact ? 'space-y-1.5 px-2 py-1' : 'space-y-2 rounded-md border p-2'}>
      {!compact ? (
        <p className="text-xs font-medium text-muted-foreground">Saída do export</p>
      ) : null}

      <div className="flex items-start gap-2">
        <Checkbox
          id={decodeId}
          checked={options.decode}
          disabled={disabled}
          onCheckedChange={(checked) => exportOutput.set({ decode: checked === true })}
          aria-label="Descompactar: gerar os formatos úteis"
        />
        <div className="min-w-0">
          <Label htmlFor={decodeId} className="cursor-pointer text-sm font-medium">
            Descompactar
          </Label>
          <p className="text-xs text-muted-foreground">
            texto (JSON/Strings/CSV/XLSX/PO) ou imagem (DDS/PNG)
          </p>
        </div>
      </div>

      <div className="flex items-start gap-2">
        <Checkbox
          id={extractId}
          checked={options.extract}
          disabled={disabled || extractDisabled}
          onCheckedChange={(checked) => exportOutput.set({ extract: checked === true })}
          aria-label="Extrair: gravar o binário real do filelist em data"
        />
        <div className="min-w-0">
          <div className="flex items-center gap-1">
            <Label htmlFor={extractId} className="cursor-pointer text-sm font-medium">
              Extrair
            </Label>
            <Popover>
              <PopoverTrigger asChild>
                <button
                  type="button"
                  className="inline-flex size-4 items-center justify-center rounded-full text-muted-foreground hover:text-foreground"
                  aria-label="O que faz Extrair?"
                >
                  <CircleHelp size={13} />
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-72 text-xs" align="start">
                <p className="font-medium">Extrair o binário do filelist</p>
                <p className="mt-1 text-muted-foreground">
                  Copia o arquivo REAL do white bin para <code>data/partN</code>, no caminho
                  interno. Se estiver comprimido no white bin, descomprime para o binário
                  utilizável. Arquivos já existentes são pulados.
                </p>
                <p className="mt-1 text-muted-foreground">
                  Só vale para itens do filelist; itens de <code>data/</code> não têm o que
                  extrair.
                </p>
              </PopoverContent>
            </Popover>
          </div>
          <p className="text-xs text-muted-foreground">
            binário real do filelist → data/ (descomprime o white bin)
          </p>
        </div>
      </div>

      {none ? <p className="text-xs text-destructive">Marque Descompactar ou Extrair.</p> : null}
    </div>
  );
}
