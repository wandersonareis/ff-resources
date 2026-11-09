// Adaptador FF13 do navegador de container .vbf.
//
// O FF13 NÃO tem containers .vbf (a família White usa white bins e
// filelists). A UI copiada do FFX ainda chama estas funções: elas resolvem
// vazias para que a árvore simply não mostre nenhum container — em vez de
// quebrar a tela.
import type * as dto from './backend/dto/models';

/** Sem containers .vbf no FF13: sempre lista vazia. */
export function ListVbfRoots(): Promise<dto.VbfRoot[]> {
  return Promise.resolve([]);
}

/** Sem container .vbf: sempre lista vazia. */
export function ListVbfDir(_root: string, _dir: string): Promise<dto.VbfNode[]> {
  return Promise.resolve([]);
}

/** Sem container .vbf: sempre lista vazia. */
export function ListVbfMacroChunks(_root: string, _dir: string): Promise<dto.VbfNode[]> {
  return Promise.resolve([]);
}

/** Sem container .vbf: nada a ler. */
export function GetVbfTextEntry(_root: string, _path: string, _id: string): Promise<dto.FileEntry> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Sem container .vbf: nada a ler. */
export function GetVbfImageEntry(_root: string, _path: string): Promise<dto.ImageEntry> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Importar textura de .vbf não se aplica ao FF13. */
export function ImportVbfImage(_root: string, _path: string, _ddsPath: string): Promise<void> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Salvar textura de .vbf não se aplica ao FF13. */
export function SaveVbfImage(
  _root: string,
  _path: string,
  _format: string,
  _destPath: string,
): Promise<string> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Replicar textura de .vbf não se aplica ao FF13. */
export function ReplicateVbfImage(
  _root: string,
  _sourcePath: string,
  _targetPaths: string[],
): Promise<dto.BatchResult> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Preview de extração de .vbf não se aplica ao FF13. */
export function PreviewVbfExtraction(
  _root: string,
  _paths: string[],
  _destRoot: string,
): Promise<dto.VbfExtractPreview> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Extração de .vbf não se aplica ao FF13. */
export function ExtractVbfSelection(
  _root: string,
  _paths: string[],
  _destRoot: string,
): Promise<dto.BatchResult> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Extração de imagens de .vbf não se aplica ao FF13. */
export function ExtractVbfImagesSelection(
  _root: string,
  _paths: string[],
  _destRoot: string,
): Promise<dto.BatchResult> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Export de .vbf não se aplica ao FF13. */
export function ExportVbfSelection(
  _root: string,
  _format: string,
  _paths: string[],
  _langs: string[],
): Promise<string[]> {
  return Promise.reject(new Error('FF13 não usa containers .vbf'));
}

/** Seletor de pasta de extração: cancela (sem .vbf no FF13). */
export function SelectVbfExtractDir(): Promise<string> {
  return Promise.resolve('');
}

/** Seletor de pasta de extração de imagens: cancela. */
export function SelectVbfImageExtractDir(): Promise<string> {
  return Promise.resolve('');
}

/** Caminho de exportação de textura de .vbf: vazio. */
export function VbfImageExportPath(_root: string, _path: string, _format: string): Promise<string> {
  return Promise.resolve('');
}
