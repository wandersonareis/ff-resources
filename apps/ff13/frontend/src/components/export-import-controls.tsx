'use client';

import { Events } from '@wailsio/runtime';
import { ChevronDown, Download, FileUp, FolderInput, ListChecks, Upload } from 'lucide-react';
import { useCallback, useState } from 'react';
import type * as dto from '@ff-resources/ui/dto';
import { ImportDestinationOptions } from '@/components/dialogs/import-destination-options';
import { ImportSummaryDialog } from '@/components/dialogs/import-summary-dialog';
import { ImportTranslatedDialog } from '@/components/dialogs/import-translated-dialog';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { type EntryKind, KIND_LABELS } from '@/lib/ffx/display-names';
import { parseError } from '@/lib/ffx/error-handler';
import { exportLangs } from '@/lib/ffx/export-langs';
import { EXPORT_FORMAT_LABELS, useExportSelection } from '@/lib/ffx/export-selection';
import { invalidateFilelistCache } from '@/lib/ffx/filelist';
import { GAME_VERSIONS, type GameVersionId } from '@/lib/ffx/game-version';
import { compareKindLabels, entryKindsAlpha } from '@/lib/ffx/kinds';
import { progressWasShown } from '@/lib/ffx/progress-store';
import { loggedToast as toast } from '@/lib/ffx/toast-logged';
import {
  exportArtifactBatch,
  importFile,
  importTranslated,
  previewImport,
  previewImportTranslated,
  selectImportFile,
} from '@/lib/ffx/tree-data';

/**
 * Controles Exportar/Importar da barra de abas + diálogo de resumo de import.
 * Única prop: a versão ativa. Tudo o mais é estado interno (escopo por versão,
 * resumo/importando) ou store módulo (`exportSelection`) — zero prop drilling.
 */
export function ExportImportControls({ version }: { version: GameVersionId }) {
  // Escopo de exportação por versão (kinds marcados no dropdown Exportar).
  // Lista e escopo inicial em ordem alfabética de exibição (Eventos primeiro).
  const [scopes, setScopes] = useState<Record<string, EntryKind[]>>(() =>
    Object.fromEntries(GAME_VERSIONS.map((tab) => [tab.id, entryKindsAlpha(tab.id)])),
  );
  const [importSummary, setImportSummary] = useState<dto.ImportSummary | null>(null);
  const [importing, setImporting] = useState(false);

  const servedKinds = entryKindsAlpha(version);
  const activeScope = scopes[version] ?? servedKinds;

  // Escopo do import em lote (mods/translated), separado do de exportação:
  // escolher o que exportar não decide o que importar.
  const [importScopes, setImportScopes] = useState<Record<string, EntryKind[]>>(() =>
    Object.fromEntries(GAME_VERSIONS.map((tab) => [tab.id, entryKindsAlpha(tab.id)])),
  );
  const [importBatch, setImportBatch] = useState<dto.ImportBatch | null>(null);
  const activeImportScope = importScopes[version] ?? servedKinds;

  /**
   * Destinos do import de TEXTO. Comprimir (mods/) nasce ligado; Inserir
   * (filelist) desligado. Os dois são independentes; sem nenhum o import
   * fica desabilitado.
   */
  const [importOptions, setImportOptions] = useState<dto.ImportDestination>({
    write_mods: true,
    insert_filelist: false,
  });
  const hasDestination = importOptions.write_mods || importOptions.insert_filelist;
  const destinationLabel = (opts: dto.ImportDestination): string => {
    const dests: string[] = [];
    if (opts.write_mods) dests.push('mods/');
    if (opts.insert_filelist) dests.push('filelist');
    return dests.length === 0 ? 'nenhum destino' : dests.join(' e ');
  };
  const afterTextImport = (opts: dto.ImportDestination) => {
    if (opts.insert_filelist) invalidateFilelistCache();
    try {
      Events.Emit('ImportDone');
    } catch {
      // sem runtime Wails (dev no browser) — segue sem recarga automática
    }
  };

  // Formato de exportação vem do BOTÃO DE FORMATO do header (direita do
  // botão de configurações) — a store `exportSelection` é a única fonte;
  // o popover do export não tem seletor próprio.
  const activeFormat = useExportSelection().format;

  const toggleImportScope = (kind: EntryKind) => {
    setImportScopes((prev) => {
      const current = prev[version] ?? entryKindsAlpha(version);
      const next = current.includes(kind) ? current.filter((k) => k !== kind) : [...current, kind];
      return { ...prev, [version]: next };
    });
  };

  const toggleScope = (kind: EntryKind) => {
    setScopes((prev) => {
      const current = prev[version] ?? entryKindsAlpha(version);
      const next = current.includes(kind) ? current.filter((k) => k !== kind) : [...current, kind];
      return { ...prev, [version]: next };
    });
  };

  // Exporta os kinds no FORMATO SELECIONADO como UMA operação agrupada:
  // o modal de progresso cuida de tudo (um step por kind com barra própria
  // + resumo de arquivos/destino no final) — nada de toast paralelo.
  //
  // O lote sai em ordem ALFABÉTICA de rótulo (Eventos, Habilidades...): a
  // ordem dos items é a ordem dos steps no modal, e o escopo guardado pode
  // estar fora de ordem depois de marcar/desmarcar checkboxes.
  //
  // O botão Exportar é SÓ DE ESCOPO DE TIPO: cada kind sai por inteiro
  // (ids = [], o contrato de "tudo do kind" do backend). A seleção de
  // itens da árvore não entra aqui — ela é do menu de contexto da árvore,
  // que é onde o usuário espera exportar "o que marquei".
  const runExport = useCallback(
    async (kinds: EntryKind[]) => {
      const format = activeFormat;
      if (kinds.length === 0) {
        toast.info('Nada para exportar — marque um tipo no escopo.');
        return;
      }
      const items = [...kinds]
        .sort(compareKindLabels)
        .map((kind) => ({ kind, ids: [] as string[] }));
      try {
        // Idiomas: array do seletor do header (default + extras marcados);
        // `[]` (pré-hidratação) = todos, o contrato do backend.
        const paths = await exportArtifactBatch(items, version, exportLangs.langsOf(), format);
        if ((paths ?? []).length === 0) {
          toast.info('Nada para exportar.');
        }
        // Saiu arquivo: o modal mostrou o resumo — sem toast de sucesso.
      } catch (error) {
        // Erro ANTES do batch abrir (formato/kind inválido, lote vazio):
        // toast. Erro no meio da operação: o modal já exibiu (step falho).
        if (!progressWasShown()) toast.error(parseError(error));
      }
    },
    [version, activeFormat],
  );

  const onImportClick = useCallback(async () => {
    if (!hasDestination) {
      toast.info('Marque Comprimir ou Inserir para escolher o destino do import.');
      return;
    }
    try {
      const path = await selectImportFile(version);
      if (!path) return;
      setImportSummary(await previewImport(path, version, importOptions));
    } catch (error) {
      toast.error(parseError(error));
    }
  }, [version, hasDestination, importOptions]);

  // Import em lote de mods/translated/text: primeiro o resumo (sem aplicar)
  // para o diálogo de confirmação, depois a aplicação arquivo a arquivo.
  const onImportTranslatedClick = useCallback(async () => {
    if (!hasDestination) {
      toast.info('Marque Comprimir ou Inserir para escolher o destino do import.');
      return;
    }
    if (activeImportScope.length === 0) {
      toast.info('Marque pelo menos um tipo para importar.');
      return;
    }
    try {
      setImportBatch(await previewImportTranslated(version, activeImportScope));
    } catch (error) {
      toast.error(parseError(error));
    }
  }, [version, activeImportScope, hasDestination]);

  const onConfirmImportTranslated = useCallback(async () => {
    if (!importBatch) return;
    if (!hasDestination) {
      toast.info('Marque Comprimir ou Inserir para escolher o destino do import.');
      return;
    }
    const options = importOptions;
    setImporting(true);
    const toastId = 'import';
    toast.loading('Importando...', { id: toastId });
    try {
      const result = await importTranslated(version, activeImportScope, options);
      setImportBatch(null);
      if (result.changed_texts === 0) {
        const motivo =
          result.errors.length > 0
            ? ` (${result.errors.length} erro(s))`
            : result.warnings.length > 0
              ? ` (${result.warnings.length} aviso(s))`
              : '';
        toast.info(`Nada a importar.${motivo}`, { id: toastId });
      } else {
        const arquivos =
          result.errors.length > 0 ? ` (${result.errors.length} arquivo(s) pulado(s))` : '';
        const avisos =
          result.warnings.length > 0 ? ` (${result.warnings.length} aviso(s))` : '';
        toast.success(
          `Importado: ${result.changed_texts} texto(s) em ${result.files.length} arquivo(s) → ` +
            `${destinationLabel(options)}.${arquivos}${avisos}`,
          { id: toastId },
        );
        afterTextImport(options);
      }
    } catch (error) {
      toast.error(parseError(error), { id: toastId });
    } finally {
      setImporting(false);
    }
  }, [importBatch, version, activeImportScope, importOptions, hasDestination]);

  const onConfirmImport = useCallback(async () => {
    const current = importSummary;
    if (!current) return;
    if (!hasDestination) {
      toast.info('Marque Comprimir ou Inserir para escolher o destino do import.');
      return;
    }
    const options = importOptions;
    setImporting(true);
    const toastId = 'import';
    toast.loading('Importando…', { id: toastId });
    try {
      const changed = await importFile(current.path, version, options);
      setImportSummary(null);
      const ignored = current.warnings?.length ?? 0;
      const skippedRows = current.skipped?.length ?? 0;
      // Sucesso com asterisco: entrada ignorada (aviso) e/ou row pulada
      // continuam existindo depois que o modal fecha — o toast carrega o
      // mesmo número que o botão pulsante.
      const parts: string[] = [];
      if (ignored > 0) parts.push(`${ignored} entrada(s) ignorada(s)`);
      if (skippedRows > 0) parts.push(`${skippedRows} row(s) ignorada(s)`);
      const suffix = parts.length > 0 ? ` (${parts.join(', ')})` : '';
      toast.success(
        `Importação concluída — ${changed} texto(s) atualizado(s) → ` +
          `${destinationLabel(options)}.${suffix}`,
        { id: toastId },
      );
      afterTextImport(options);
    } catch (error) {
      toast.error(parseError(error), { id: toastId });
    } finally {
      setImporting(false);
    }
  }, [importSummary, version, importOptions, hasDestination]);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            size="sm"
            variant="outline"
            title={`Exportar textos — ${EXPORT_FORMAT_LABELS[activeFormat]} (formato do seletor ao lado das configurações): todos os tipos, ou só os tipos marcados no escopo`}
          >
            <Download size={16} />
            <span className="hidden sm:inline">Exportar</span>
            <ChevronDown size={12} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-70">
          <DropdownMenuItem onSelect={() => void runExport(servedKinds)}>
            <Download size={14} />
            Exportar
            <span className="ml-auto pl-3 text-xs opacity-60">todos os tipos</span>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuLabel className="text-muted-foreground">Escopo</DropdownMenuLabel>
          {servedKinds.map((kind) => (
            <DropdownMenuCheckboxItem
              key={kind}
              checked={activeScope.includes(kind)}
              onCheckedChange={() => toggleScope(kind)}
              onSelect={(event) => event.preventDefault()}
            >
              {KIND_LABELS[kind]}
            </DropdownMenuCheckboxItem>
          ))}
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => void runExport(activeScope)}>
            <ListChecks size={14} />
            Exportar seleção
            <span className="ml-auto pl-3 text-xs opacity-60">tipos marcados</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            size="sm"
            variant="outline"
            title="Importar textos — em lote de mods/translated/text ou de um arquivo; Comprimir (mods/) e Inserir (filelist)"
          >
            <Upload size={16} />
            <span className="hidden sm:inline">Importar</span>
            <ChevronDown size={12} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-70">
          <DropdownMenuLabel className="text-muted-foreground">Tipos</DropdownMenuLabel>
          {servedKinds.map((kind) => (
            <DropdownMenuCheckboxItem
              key={kind}
              checked={activeImportScope.includes(kind)}
              onCheckedChange={() => toggleImportScope(kind)}
              onSelect={(event) => event.preventDefault()}
            >
              {KIND_LABELS[kind]}
            </DropdownMenuCheckboxItem>
          ))}
          <DropdownMenuSeparator />
          <ImportDestinationOptions
            options={importOptions}
            onChange={setImportOptions}
            disabled={importing}
            compact
            idPrefix="text-import-dest"
          />
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onSelect={() => void onImportTranslatedClick()}
            disabled={!hasDestination}
          >
            <FolderInput size={14} />
            Importar
            <span className="ml-auto pl-3 text-xs opacity-60">translated/text</span>
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => void onImportClick()} disabled={!hasDestination}>
            <FileUp size={14} />
            Importar de…
            <span className="ml-auto pl-3 text-xs opacity-60">arquivo</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <ImportSummaryDialog
        summary={importSummary}
        open={importSummary !== null}
        importing={importing}
        options={importOptions}
        onOptionsChange={setImportOptions}
        onOpenChange={(open) => {
          if (!open && !importing) setImportSummary(null);
        }}
        onConfirm={() => void onConfirmImport()}
      />
      <ImportTranslatedDialog
        batch={importBatch}
        open={importBatch !== null}
        importing={importing}
        options={importOptions}
        onOptionsChange={setImportOptions}
        onOpenChange={(open) => {
          if (!open && !importing) setImportBatch(null);
        }}
        onConfirm={() => void onConfirmImportTranslated()}
      />
    </>
  );
}
