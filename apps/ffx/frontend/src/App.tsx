import { AppShell } from '@/components/app-shell';
import { WindowHost } from '@/components/window/window-host';

/**
 * Roteamento mínimo do app desktop: /w/* são as janelas filhas; qualquer outra
 * rota é a janela principal. Substitui as páginas do App Router do Next.
 */
export function App() {
  const isChildWindow = window.location.pathname.startsWith('/w');
  return isChildWindow ? <WindowHost /> : <AppShell />;
}
