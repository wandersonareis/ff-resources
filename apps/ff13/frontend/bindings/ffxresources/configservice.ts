// Adaptador FF13 do serviço de CONFIGURAÇÃO.
//
// A configuração do FF13 é o arquivo local `ff13.config.json` (raiz de
// instalação por part) — não o config do FFX. O backend expõe a leitura
// (ConfigSnapshot/LocalConfig); os SETTERS do FFX não têm equivalente no
// FF13 ainda, então ficam sem efeito: o painel de configuração do FF13 é
// somente leitura e o store do frontend é otimista.
import * as ff13 from '@ff13-bindings/ffresources/apps/ff13/ff13service';

/** Snapshot do config (mesmo formato snake_case do DTO Go). */
export function GetConfigSnapshot(): Promise<unknown> {
  return ff13.ConfigSnapshot() as Promise<unknown>;
}

/** Define a pasta dos arquivos originais do jogo (sem efeito no FF13). */
export function SetGameLocation(_path: string): Promise<void> {
  return Promise.resolve();
}

/** Define a pasta das traduções (sem efeito no FF13). */
export function SetTranslateLocation(_path: string): Promise<void> {
  return Promise.resolve();
}

/** Define o executável do jogo (sem efeito no FF13). */
export function SetGameExeLocation(_path: string): Promise<void> {
  return Promise.resolve();
}

/** Liga/desliga o modo mods (o FF13 é mods-first por padrão). */
export function SetEnableMods(_enabled: boolean): Promise<void> {
  return Promise.resolve();
}

/** Troca a versão/jogo ativo (o FF13 troca a PART em memória). */
export function SetGameVersion(_version: string): Promise<void> {
  return Promise.resolve();
}

/** Persiste o tema ("dark" | "light"); o FF13 guarda no localStorage. */
export function SetTheme(_theme: string): Promise<void> {
  return Promise.resolve();
}

/**
 * Seletor nativo de diretório. O FF13 não expõe o diálogo do Wails ainda:
 * devolve string vazia (o chamador trata como cancelamento).
 */
export function SelectDirectory(_title: string): Promise<string> {
  return Promise.resolve('');
}

/** Seletor nativo de executável (mesma regra do seletor de diretório). */
export function SelectGameExeFile(): Promise<string> {
  return Promise.resolve('');
}

/**
 * Confirma a troca do modo mods. Sem backend no FF13, a decisão é sempre
 * "aplicar" (o app é mods-first).
 */
export function ResolveEnableModsChange(
  _enabled: boolean,
  _decision: string,
): Promise<void> {
  return Promise.resolve();
}
