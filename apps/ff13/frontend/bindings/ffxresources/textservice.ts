// Adaptador FF13 para os imports da UI copiada do FFX.
//
// A UI do FFX chama TextService(kind, version, ...); o app FF13 expõe
// FF13Service(part, ...). Este módulo traduz os dois mundos e mantém a UI
// independente dos bindings gerados pelo Wails.
import * as ff13 from '@ff13-bindings/ffresources/apps/ff13/backend/service/ff13service';

function partOf(version: unknown): string {
  const v = String(version ?? 'part1');
  return v === 'part2' || v === 'part3' ? v : 'part1';
}

export async function ListTextEntries(
  _kind: string,
  version: unknown,
  ..._args: any[]
): Promise<any> {
  return ff13.ListTextUnits(partOf(version));
}

/**
 * Inventário da coleção de data/partN (unidades de texto + arquivos fora do
 * escopo). É a carga da aba: o backend cacheia por part.
 */
export async function ListDataEntries(version: unknown): Promise<any> {
  return ff13.ListDataEntries(partOf(version));
}

/**
 * Limpa o cache da part (mods/ mudou por fora, save ou cold-reload): a próxima
 * leitura refaz a varredura.
 */
export async function InvalidateDataCollection(version: unknown): Promise<void> {
  await ff13.InvalidateDataCollection(partOf(version));
}

export async function GetTextEntry(
  _kind: string,
  id: string,
  version: unknown,
  lang?: string,
  origin?: { root?: string; path?: string } | null,
  ..._args: any[]
): Promise<any> {
  return ff13.GetTextEntry(
    partOf(version),
    id,
    String(lang ?? ''),
    String(origin?.root ?? ''),
    String(origin?.path ?? ''),
  );
}

export async function ApplyTextCollection(
  _kind: string,
  version: unknown,
  collection: any,
  ..._args: any[]
): Promise<void> {
  await ff13.ApplyTextCollection(partOf(version), collection);
}

export async function ListLanguages(..._args: any[]): Promise<any> {
  return ff13.ListLanguages();
}

/**
 * Catálogo de tags da part (CLR/ICON/BTN/VAR/CTR/KEY/PAGE/NL/SPACE) para o
 * autocomplete do editor — binding real do FF13Service.
 */
export async function GetTagCatalog(
  version: unknown,
  lang?: string,
  ..._args: any[]
): Promise<any> {
  return ff13.GetTagCatalog(partOf(version), String(lang ?? 'us'));
}

export async function SearchTextEntries(
  _version: unknown,
  query: string,
  ..._args: any[]
): Promise<any> {
  return { results: [], totalRows: 0, totalFiles: 0, truncated: false, query };
}

export async function WarmTextSearch(..._args: any[]): Promise<void> {}

export async function ExportEntry(
  _kind: string,
  id: string,
  version: unknown,
  langs?: string[],
  ..._args: any[]
): Promise<string[]> {
  return (await ff13.ExportEntry(partOf(version), id, langs ?? null)) ?? [];
}

export async function ExportArtifactBatch(
  items: any,
  version: unknown,
  langs?: string[],
  format?: string,
  ..._args: any[]
): Promise<string[]> {
  return (await ff13.ExportArtifactBatch(items ?? null, partOf(version), langs ?? null, String(format ?? ''))) ?? [];
}

/** Destino padrão do import de texto: converte para mods/. */
const defaultTextImportOptions = { write_mods: true, insert_filelist: false };

/** Seleciona UM artefato de texto (json/strings/csv/xlsx/po). */
export async function SelectImportFile(version: unknown, ..._args: any[]): Promise<string> {
  return ff13.SelectImportFile(partOf(version));
}

export async function PreviewImport(
  path: string,
  version: unknown,
  options?: any,
  ..._args: any[]
): Promise<any> {
  return ff13.PreviewImport(partOf(version), path, options ?? defaultTextImportOptions);
}

export async function ImportFile(
  path: string,
  version: unknown,
  options?: any,
  ..._args: any[]
): Promise<number> {
  return ff13.ImportFile(partOf(version), path, options ?? defaultTextImportOptions);
}

export async function ImportEntry(..._args: any[]): Promise<string[]> {
  return [];
}

export async function PreviewImportTranslated(
  version: unknown,
  kinds: string[],
  ..._args: any[]
): Promise<any> {
  return ff13.PreviewImportTranslated(partOf(version), kinds ?? []);
}

export async function ImportTranslated(
  version: unknown,
  kinds: string[],
  options?: any,
  ..._args: any[]
): Promise<any> {
  return ff13.ImportTranslated(partOf(version), kinds ?? [], options ?? defaultTextImportOptions);
}

export async function PreviewSavePrune(..._args: any[]): Promise<any[]> {
  return [];
}

export async function CountChangedTexts(..._args: any[]): Promise<number> {
  return 0;
}

export async function ApplyVbfTextCollection(..._args: any[]): Promise<void> {}

export async function GetMetadata(query: string, ..._args: any[]): Promise<any> {
  return { key: query };
}

export async function GetTextCollection(..._args: any[]): Promise<any> {
  return {};
}

