import { LocalConfig } from '@ff13-bindings/ffresources/apps/ff13/ff13service';

/**
 * Config local do app (arquivo ff13.config.json): onde o jogo está instalado
 * e onde ficam os binários necessários. O arquivo é criado pelo backend com
 * placeholders quando não existe — a interface só LÊ.
 */
export interface LocalConfigSnapshot {
  /** Caminho do arquivo de config em uso. */
  path: string;
  /** Raiz de instalação por part ("part1", "part2", "part3"). */
  roots: Record<string, string>;
  /** Binários configurados (nome -> caminho). */
  binaries: Record<string, string>;
}

let cached: LocalConfigSnapshot | null = null;

/** Lê (com cache) a config local do backend. */
export async function loadLocalConfig(): Promise<LocalConfigSnapshot> {
  if (cached) return cached;
  const raw = await LocalConfig();
  cached = {
    path: raw.path ?? '',
    roots: { ...(raw.roots ?? {}) },
    binaries: { ...(raw.binaries ?? {}) },
  };
  return cached;
}

/** Raio configurada de uma part ("" quando não configurada). */
export function localRoot(config: LocalConfigSnapshot | null, part: string): string {
  if (!config) return '';
  return config.roots?.[part] ?? '';
}
