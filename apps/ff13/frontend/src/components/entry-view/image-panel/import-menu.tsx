'use client';

import { ChevronDown, FolderInput, Images, Upload } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import type { ImageBusy } from './use-image-actions';

export function ImageImportMenu({
  disabled,
  onImportFile,
  onImportTranslated,
  busy,
}: {
  disabled: boolean;
  onImportFile: () => void;
  onImportTranslated: () => void;
  busy: ImageBusy;
}) {
  const isBusy = busy === 'import' || busy === 'importBatch';

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
        {/* 1. Padrão = mods/translated/images (primeiro item, como no texto) */}
        <DropdownMenuItem
          onSelect={() => onImportTranslated()}
          title="Importa os .dds de mods/translated/images/<part> — pasta = caminho do container, arquivo = nome do registro"
          disabled={isBusy}
        >
          <Images size={14} />
          Importar
          <span className="ml-auto pl-3 text-xs opacity-60">mods/translated/images</span>
        </DropdownMenuItem>
        {/* 2. Importar de arquivo .dds */}
        <DropdownMenuItem
          onSelect={() => onImportFile()}
          title="Aplica um .dds escolhido na imagem ABERTA — o caminho e o nome do arquivo não importam"
          disabled={isBusy}
        >
          <FolderInput size={14} />
          Importar de…
          <span className="ml-auto pl-3 text-xs opacity-60">arquivo</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
