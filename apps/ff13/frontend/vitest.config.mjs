/// <reference types="vitest" />

import path from 'node:path';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';
import { sharedAliases } from '../../../frontend/shared/vite-aliases.mjs';

const here = path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1'));

export default defineConfig({
  plugins: [react({ babel: { plugins: ['babel-plugin-react-compiler'] } })],
  resolve: {
    tsconfigPaths: true,
    dedupe: [
      'react',
      'react-dom',
      'cn',
      'radix-ui',
      'lucide-react',
      'class-variance-authority',
      'sonner',
      'react-resizable-panels',
      '@tanstack/react-hotkeys',
      '@tanstack/react-store',
      '@tanstack/react-table',
      '@tanstack/store',
      '@wailsio/runtime',
      '@tiptap/core',
      '@tiptap/pm',
      '@tiptap/starter-kit',
      '@tiptap/extension-color',
      '@tiptap/extension-text-style',
      '@tiptap/suggestion',
    ],    // A ordem importa: o alias é prefixo e o PRIMEIRO que casa vence. Os
    // aliases do pacote compartilhado vêm ANTES do '@' genérico.
    alias: {
      ...sharedAliases(here),
      '@/bindings': path.resolve(here, 'bindings'),
      '@ff13-bindings': path.resolve(here, 'bindings-ff13'),
      '@': path.resolve(here, 'src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./vitest.setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    alias: {
      ...sharedAliases(here),
      '@/bindings': path.resolve(here, 'bindings'),
      '@ff13-bindings': path.resolve(here, 'bindings-ff13'),
      '@': path.resolve(here, 'src'),
    },
    deps: {
      inline: ['@/lib/ffx/**'],
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        '**/node_modules/**',
        '**/*.d.ts',
        '**/*.config.*',
        'vitest.setup.ts',
        '**/bindings/**',
        '**/*.generated.ts',
      ],
    },
  },
});
