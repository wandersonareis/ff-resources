'use client';

import { ChevronDown, FolderOpen, Save } from 'lucide-react';
import { ExportOutputOptions } from '@/components/dialogs/export-output-options';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useExportOutput } from '@/lib/ffx/export-output';
import {
  IMAGE_EXPORT_FORMAT_LABELS,
  type ImageExportFormat,
  useImageExportFormat,
} from '@/lib/ffx/image-export-selection';

/**
 * "Exportar como" do painel: usa o formato selecionado no seletor de formato.
 *
 *  - `Exportar` (item do menu) → salva no formato selecionado (DDS, PNG ou DDS+PNG)
 *    direto em mods/images (sem diálogo).
 *  - `Exportar para...`        → abre seletor de pasta nativo e extrai no formato
 *    selecionado para destino customizado.
 *
 * O .dds é o padrão sem perda (volta no repack); o .png é só para
 * visualizar/distribuir. DDS+PNG gera os dois formatos.
 */
export function ImageExportMenu({
  disabled,
  extractDisabled,
  onExport,
  onExtractTo,
}: {
  disabled: boolean;
  /** Extrair binário do filelist não se aplica a data/VBF. */
  extractDisabled?: boolean;
  /** Exporta no formato selecionado para mods/images (sem diálogo). */
  onExport: (format: ImageExportFormat) => void;
  /** Exporta no formato selecionado para pasta customizada (com diálogo). */
  onExtractTo: (format: ImageExportFormat) => void;
}) {
  const { format } = useImageExportFormat();
  const output = useExportOutput();
  const hasOutput = output.decode || output.extract;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          size="sm"
          variant="outline"
          disabled={disabled}
          title="Exportar — salva no formato selecionado em mods/images, ou escolha pasta customizada"
        >
          <Save size={16} />
          Exportar
          <ChevronDown size={12} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-60">
        <ExportOutputOptions
          compact
          extractDisabled={extractDisabled}
          idPrefix="image-export-out"
        />
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={!hasOutput}
          onClick={() => onExport(format)}
          title={`Exporta como ${IMAGE_EXPORT_FORMAT_LABELS[format]} para mods/ e/ou extrai o binário do filelist para data/`}
        >
          <Save size={14} />
          Exportar
          <span className="ml-auto pl-3 text-xs opacity-60">mods/ · data/</span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={!hasOutput}
          onClick={() => onExtractTo(format)}
          title="Escolhe a pasta de destino e exporta/extrai preservando o caminho interno"
        >
          <FolderOpen size={14} />
          Exportar para…
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
