import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import wails from '@wailsio/runtime/plugins/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { sharedAliases } from '../../../frontend/shared/vite-aliases.mjs';

const here = path.dirname(fileURLToPath(import.meta.url));
const sharedRoot = path.resolve(here, '../../../frontend/shared');

/**
 * Os bindings gerados pelo Wails importam com extensão .js apontando para
 * arquivos .ts (ESM). O Next resolvia via extensionAlias; no Vite este plugin
 * resolve o .js para o .ts real.
 */
function bindingsJsToTs() {
  return {
    name: 'bindings-js-to-ts',
    enforce: 'pre' as const,
    resolveId(source: string, importer: string | undefined) {
      if (!source.endsWith('.js') || !source.startsWith('.')) return null;
      if (!importer) return null;
      const target = path.resolve(path.dirname(importer), source);
      const ts = target.replace(/\.js$/, '.ts');
      return fs.existsSync(ts) ? ts : null;
    },
  };
}

/** Dependências do pacote compartilhado: pré-bundle no dev para o HMR não
 *  recompilar o Tiptap/Radix a cada edição. */
const sharedDeps = [
  'react',
  'react-dom',
  '@tanstack/react-hotkeys',
  '@tanstack/react-store',
  '@tanstack/store',
  '@tanstack/react-table',
  '@tiptap/core',
  '@tiptap/pm',
  '@tiptap/starter-kit',
  '@tiptap/extension-color',
  '@tiptap/extension-text-style',
  '@tiptap/suggestion',
  'radix-ui',
  'lucide-react',
  'sonner',
  'react-resizable-panels',
  '@wailsio/runtime',
];

export default defineConfig({
  plugins: [
    // React Compiler: memoização automática dos componentes (o Next fazia
    // isso com reactCompiler: true; aqui é o plugin de Babel).
    react({ babel: { plugins: ['babel-plugin-react-compiler'] } }),
    tailwindcss(),
    wails('./bindings'),
    bindingsJsToTs(),
  ],
  resolve: {
    // Vite 8 resolve os paths do tsconfig nativamente.
    tsconfigPaths: true,
    dedupe: [...sharedDeps, 'cn', 'class-variance-authority'],
    alias: {
      ...sharedAliases(here),
      '@/bindings': path.resolve(here, 'bindings'),
    },
  },
  optimizeDeps: {
    // @tiptap/pm só exporta SUBPATHS (não tem export "."): incluí-lo no
    // include faz o dev server tentar resolver a raiz e falhar. Continua no
    // dedupe (que não pré-empacota).
    include: sharedDeps.filter((dep) => dep !== '@tiptap/pm'),
  },
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    // O pacote compartilhado vive FORA da raiz do app: sem isto o dev server
    // recusa os arquivos dele.
    fs: { allow: [here, sharedRoot] },
  },
  build: {
    outDir: 'out',
    emptyOutDir: true,
    target: 'es2022',
    cssCodeSplit: true,
    reportCompressedSize: false,
    chunkSizeWarningLimit: 900,
    rollupOptions: {
      input: {
        main: path.resolve(here, 'index.html'),
        w: path.resolve(here, 'w/index.html'),
      },
      output: {
        // Separa os vendors pesados: o app shell recarrega menos em update.
        manualChunks(id: string) {
          if (!id.includes('node_modules')) return undefined;
          if (id.includes('@tiptap') || id.includes('prosemirror')) return 'tiptap';
          if (
            id.includes('react-dom') ||
            id.includes('node_modules/react/') ||
            id.includes('node_modules/scheduler')
          ) {
            return 'react';
          }
          return undefined;
        },
      },
    },
  },
});
