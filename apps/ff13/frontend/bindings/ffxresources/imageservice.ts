// Adaptador FF13 para os métodos de imagem do FF13Service.
//
// O GameApi fala (kind, version, id); o backend fala (part, ref). Este módulo
// converte a version para a part e chama os bindings gerados pelo Wails.
import * as ff13 from '@ff13-bindings/ffresources/apps/ff13/backend/service/ff13service';

const svc = ff13 as any;

function partOf(version: unknown): string {
  const v = String(version ?? 'part1');
  return v === 'part2' || v === 'part3' ? v : 'part1';
}

export function GetImageEntry(version: unknown, ref: string): Promise<any> {
  return svc.GetImageEntry(partOf(version), ref);
}

export function ImageExists(version: unknown, ref: string): Promise<boolean> {
  return svc.ImageExists(partOf(version), ref);
}

export function ImageDiagnostics(version: unknown, ref: string): Promise<any> {
  return svc.ImageDiagnostics(partOf(version), ref);
}

export function ListImageContainer(version: unknown, ref: string): Promise<any[]> {
  return svc.ListImageContainer(partOf(version), ref);
}

export function SaveImage(
  version: unknown,
  ref: string,
  format: string,
  destPath: string,
): Promise<void> {
  return svc.SaveImage(partOf(version), ref, format, destPath);
}

export function ImageExportPath(version: unknown, ref: string, format: string): Promise<string> {
  return svc.ImageExportPath(partOf(version), ref, format);
}

export function ExtractImageSelection(
  version: unknown,
  refs: string[],
  destRoot: string,
): Promise<any> {
  return svc.ExtractImageSelection(partOf(version), refs, destRoot);
}

export function ImageDeletePlans(version: unknown, refs: string[]): Promise<any[]> {
  return svc.ImageDeletePlans(partOf(version), refs);
}

export function DeleteImages(version: unknown, refs: string[], scope: string): Promise<any> {
  return svc.DeleteImages(partOf(version), refs, scope);
}

export function DeleteImageSelection(
  version: unknown,
  refs: string[],
  scope: string,
): Promise<any> {
  return svc.DeleteImageSelection(partOf(version), refs, scope);
}

export function SelectImageExportDir(version: unknown): Promise<string> {
  return Promise.resolve(svc.SelectImageExportDir(partOf(version)) ?? '');
}

export function SelectImageFile(version: unknown, id: string): Promise<string> {
  return Promise.resolve(svc.SelectImageFile(partOf(version), id, '') ?? '');
}

export function SelectImageSavePath(
  version: unknown,
  format: string,
  suggestedName: string,
  id: string,
): Promise<string> {
  return Promise.resolve(
    svc.SelectImageSavePath(partOf(version), format, suggestedName, id, '') ?? '',
  );
}

export function ImportImage(version: unknown, ref: string, ddsPath: string): Promise<void> {
  return svc.ImportImage(partOf(version), ref, ddsPath);
}

export function ImportImageGroup(
  version: unknown,
  ref: string,
  ddsPath: string,
  targets: string[],
): Promise<any> {
  return svc.ImportImageGroup(partOf(version), ref, ddsPath, targets ?? null);
}

export function PreviewImportImages(version: unknown): Promise<any> {
  return svc.PreviewImportImages(partOf(version));
}

export function ImportImages(version: unknown, selection: any): Promise<any> {
  return svc.ImportImages(partOf(version), selection);
}

export function RefreshImageDuplicates(): Promise<void> {
  return Promise.resolve();
}
