export type {
  Collection,
  EntrySummary,
  FileEntry,
  GameApi,
  ImportSummary,
} from './api/GameApi';
export { ThemeProvider, useTheme } from './theme';
export type { Theme, ThemeProviderProps } from './theme';
export { cn } from './lib/utils';

/* Primitivas de UI (shadcn) compartilhadas por FFX e FF13. */
export * from './components/ui/alert';
export * from './components/ui/badge';
export * from './components/ui/button';
export * from './components/ui/checkbox';
export * from './components/ui/collapsible';
export * from './components/ui/context-menu';
export * from './components/ui/dialog';
export * from './components/ui/dropdown-menu';
export * from './components/ui/input';
export * from './components/ui/label';
export * from './components/ui/popover';
export * from './components/ui/progress';
export * from './components/ui/resizable';
export * from './components/ui/scroll-area';
export * from './components/ui/separator';
export * from './components/ui/sheet';
export * from './components/ui/sonner';
export * from './components/ui/table';
export * from './components/ui/tabs';
export * from './components/ui/tooltip';
