import { Window } from '@wailsio/runtime';
import { useEffect, useState } from 'react';
import { LogPanelControls } from '@/components/log-panel-controls';
import { ConfigWindowContent } from '@/components/window/config-window';
import { LogWindowContent } from '@/components/window/log-window';
import { WindowChrome } from '@/components/window/window-chrome';

/**
 * Host das janelas filhas: o backend cria a janela com URL "/w/#<nome>" e o
 * hash escolhe o conteúdo montado aqui dentro. Substitui a rota /w do Next.
 */
const WINDOWS: Record<
  string,
  {
    title: string;
    Content: () => React.JSX.Element;
    TitleExtra?: () => React.JSX.Element;
  }
> = {
  config: { title: 'Configurações', Content: ConfigWindowContent },
  logs: {
    title: 'Logs',
    Content: LogWindowContent,
    TitleExtra: () => <LogPanelControls />,
  },
};

export function WindowHost() {
  const [name, setName] = useState<string | null>(null);

  useEffect(() => {
    setName(window.location.hash.slice(1) || null);
  }, []);

  const entry = name ? WINDOWS[name] : undefined;

  if (!entry) {
    // Hash desconhecido ou ausente: fecha para não deixar janela órfã.
    if (name !== null) {
      void Window.Close().catch(() => undefined);
    }
    return null;
  }

  const { title, Content, TitleExtra } = entry;
  return (
    <WindowChrome title={title} titleExtra={TitleExtra ? <TitleExtra /> : undefined}>
      <Content />
    </WindowChrome>
  );
}
