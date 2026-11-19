'use client';

import type * as dto from '@ff-resources/ui/dto';
import { ChevronDown, FolderInput, Images, Undo2, Upload } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ImportDestinationOptions } from '@/components/dialogs/import-destination-options';
import type { ImageBusy } from './use-image-actions';

export function ImageImportMenu({
  disabled,
  onImportFile,
  onImportTranslated,
  options,
  onOptionsChange,
  onRestoreInserts,
  busy,
}: {
  disabled: boolean;
  onImportFile: () => void;
  onImportTranslated: () => void;
  options: dto.ImageImportOptions;
  onOptionsChange: (next: dto.ImageImportOptions) => void;
  onRestoreInserts: () => void;
  busy: ImageBusy;
}) {
  const isBusy = busy === 'import' || busy === 'importBatch';
  const noDestination = !options.write_mods && !options.insert_filelist;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          size="sm"
          variant="outline"
          disabled={disabled || isBusy}
          title="Importar imagens — de mods/translated/images/<part> (lote) ou de um arquivo .dds para a imagem aberta"
        >
          <Upload size={16} />
          {isBusy ? 'Importando…' : 'Importar'}
          <ChevronDown size={12} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-70">
        <ImportDestinationOptions
          options={options}
          onChange={onOptionsChange}
          disabled={isBusy}
          compact
          idPrefix="menu-import-dest"
        />
        <DropdownMenuSeparator />

        {/* 1. Padrão = mods/translated/images (primeiro item, como no texto) */}
        <DropdownMenuItem
          onSelect={() => onImportTranslated()}
          title={
            noDestination
              ? 'Marque Comprimir ou Inserir antes de importar'
              : 'Importa os .dds de mods/translated/images/<part> — pasta = caminho do container, arquivo = nome do registro'
          }
          disabled={isBusy || noDestination}
        >
          <Images size={14} />
          Importar
          <span className="ml-auto pl-3 text-xs opacity-60">mods/translated/images</span>
        </DropdownMenuItem>
        {/* 2. Importar de arquivo .dds */}
        <DropdownMenuItem
          onSelect={() => onImportFile()}
          title={
            noDestination
              ? 'Marque Comprimir ou Inserir antes de importar'
              : 'Aplica um .dds escolhido na imagem ABERTA — o caminho e o nome do arquivo não importam'
          }
          disabled={isBusy || noDestination}
        >
          <FolderInput size={14} />
          Importar de…
          <span className="ml-auto pl-3 text-xs opacity-60">arquivo</span>
        </DropdownMenuItem>

        <DropdownMenuSeparator />
        <DropdownMenuItem
          onSelect={() => onRestoreInserts()}
          title="Restaura os filelists do backup e trunca os white bins ao tamanho original"
          disabled={isBusy}
        >
          <Undo2 size={14} />
          Reverter inserções
          <span className="ml-auto pl-3 text-xs opacity-60">filelist</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

