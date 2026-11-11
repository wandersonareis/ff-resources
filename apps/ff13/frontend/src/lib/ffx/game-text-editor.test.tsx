import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { GameApi, TagCatalog } from '@ff-resources/ui/api/GameApi';
import { ApiProvider } from '@ff-resources/ui/api/ApiProvider';
import { TooltipProvider } from '@/components/ui/tooltip';
import { GameTextEditor } from '@/components/editor/game-text-editor';

const CATALOG: TagCatalog = {
  version: 'part1',
  lang: 'us',
  tags: [
    {
      tag: 'CLR',
      prefix: '{CLR:',
      hint: 'Cor do texto',
      values: [
        {
          tag: '{CLR:F9:40:White}',
          label: 'White',
          key: 'F9:40',
          hex: '#ffffff',
          reset: true,
        },
        { tag: '{CLR:F9:42:Gold}', label: 'Gold', key: 'F9:42', hex: '#e8c15a' },
        {
          tag: '{CLR:F9:32:Ex00}',
          label: 'Ex00',
          key: 'F9:32',
          hex: '#111111',
          group: 'ex',
          semiTransparent: true,
        },
      ],
    },
  ],
};

function mount(value: string) {
  const api = { getTagCatalog: async () => CATALOG } as unknown as GameApi;
  return render(
    <TooltipProvider>
      <ApiProvider api={api}>
        <GameTextEditor value={value} onValueChange={() => {}} version="part1" lang="us" />
      </ApiProvider>
    </TooltipProvider>,
  );
}

describe('GameTextEditor — cor com bytes do FF13', () => {
  it('renderiza a cor como mark com data-game-color', async () => {
    mount('{CLR:F9:42:Gold}Oi');
    await waitFor(() => expect(document.querySelector('.tiptap')).toBeTruthy());
    expect(document.querySelector('.tiptap')?.innerHTML).toContain(
      'data-game-color="{CLR:F9:42:Gold}"',
    );
  });

  it('o botão de cor abre o popover com a paleta do idioma', async () => {
    mount('texto');
    await waitFor(() => expect(document.querySelector('.tiptap')).toBeTruthy());

    fireEvent.click(screen.getByRole('button', { name: 'Cor do texto' }));
    expect(await screen.findByRole('button', { name: 'Cor Gold' })).toBeTruthy();
    // White é reset: fica no botão Resetar, não no grid de cores.
    expect(screen.queryByRole('button', { name: 'Cor White' })).toBeNull();
    const reset = screen.getByRole('button', { name: 'Resetar cor' });
    expect(reset.getAttribute('title')).toBe('{CLR:F9:40:White}');
    // Ex fica num grupo recolhido, com o nome visível.
    const exSummary = screen.getByText('Ex — semitransparentes (1)');
    expect(exSummary).toBeTruthy();
    fireEvent.click(exSummary);
    expect(await screen.findByRole('button', { name: 'Cor Ex00' })).toBeTruthy();
  });

});
