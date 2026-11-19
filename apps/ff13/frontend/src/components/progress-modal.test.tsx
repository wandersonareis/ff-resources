import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { ProgressModal } from '@/components/progress/progress-modal';
import { clearProgressBatch, progressStarted } from '@/lib/ffx/progress-store';

describe('ProgressModal — largura reservada da contagem', () => {
  afterEach(() => clearProgressBatch());

  it('reserva o maior "000/000" desde o ProgressStart, antes de a ação rodar', () => {
    // Total declarado upfront (StepsWithTotals do backend): o modal abre
    // com a reserva mesmo com todos os steps pendentes (nenhuma contagem
    // real visível ainda).
    progressStarted({
      title: 'Exportando textos (JSON)…',
      steps: [
        { label: 'Eventos', total: 301, processed: 0, issueCount: 0, state: 'pending' },
        { label: 'Habilidades', total: 42, processed: 0, issueCount: 0, state: 'pending' },
      ],
      current: 0,
      done: false,
      issueCount: 0,
    });
    render(<ProgressModal />);

    expect(screen.getByText('301/301')).toHaveClass('invisible');
    expect(screen.getByText('42/42')).toHaveClass('invisible');
    // Pendente = sem contagem real, só a reserva.
    expect(screen.queryByText('0/301')).toBeNull();
  });

  it('sobrepõe a contagem real na célula reservada (não cresce)', () => {
    progressStarted({
      title: 'Exportando textos (JSON)…',
      steps: [{ label: 'Eventos', total: 301, processed: 5, issueCount: 0, state: 'running' }],
      current: 0,
      done: false,
      issueCount: 0,
    });
    render(<ProgressModal />);

    expect(screen.getByText('301/301')).toHaveClass('invisible');
    const count = screen.getByText('5/301');
    expect(count).toHaveClass('col-start-1', 'row-start-1');
  });
});
