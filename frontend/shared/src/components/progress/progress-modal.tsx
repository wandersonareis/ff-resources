'use client';

import { AlertTriangle, CheckCircle2, CircleDashed, XCircle } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Progress } from '../ui/progress';
import {
  clearProgressBatch,
  type ProgressBatch,
  type ProgressStep,
  progressEnded,
  progressStarted,
  progressUpdated,
  useProgressBatch,
} from '../../app/progress-store';
import { useWailsEvent } from '../../app/use-wails-event';

const fmt = (n: number): string => n.toLocaleString('pt-BR');

/** Contagem da ação: "301/301" quando o total é conhecido. */
function stepCount(step: ProgressStep): string {
  return step.total > 0 ? `${fmt(step.processed)}/${fmt(step.total)}` : fmt(step.processed);
}

/**
 * Maior texto que a contagem desta ação PODE assumir — é a largura
 * reservada ("301/301" = os "000/000" de 3 dígitos). Com o total já
 * declarado (ProgressStart leva o total), o espaço fica reservado desde o
 * primeiro frame; como nada passa do total, a coluna nunca cresce. Ação
 * sem total (indeterminada) reserva só o processado atual.
 */
function stepCountReserve(step: ProgressStep): string {
  const max = Math.max(step.total, step.processed);
  if (max <= 0) return '';
  const n = fmt(max);
  return step.total > 0 ? `${n}/${n}` : n;
}

/** Uma ação da lista: nome, barra e contagem usam as MESMAS colunas em todas
 *  as linhas (grid do pai + subgrid aqui), então a barra não muda de largura
 *  conforme o texto/contagem de cada ação. */
function StepRow({ step }: { step: ProgressStep }) {
  const dim = step.state === 'pending';
  const reserve = stepCountReserve(step);
  const showCount = step.state !== 'pending';
  const icon =
    step.state === 'done' ? (
      step.issueCount > 0 ? (
        <AlertTriangle className="size-4 text-warning" aria-label="Concluída com falhas" />
      ) : (
        <CheckCircle2 className="size-4 text-success" aria-label="Concluída" />
      )
    ) : step.state === 'error' ? (
      <XCircle className="size-4 text-destructive" aria-label="Falhou" />
    ) : step.state === 'pending' ? (
      <CircleDashed className="size-4" aria-label="Aguardando" />
    ) : null;
  return (
    <div
      className={`col-span-3 grid grid-cols-subgrid items-center gap-x-3 gap-y-0 ${dim ? 'opacity-50' : ''}`}
      role="status"
    >
      <span className="min-w-0 truncate text-sm" title={step.label}>
        {step.label}
      </span>
      {step.state === 'running' ? (
        <Progress
          value={step.total > 0 ? Math.min(100, (step.processed / step.total) * 100) : 0}
          className="h-1.5 min-w-0"
        />
      ) : (
        <span aria-hidden="true" />
      )}
      <span className="flex items-center justify-end gap-1.5">
        {reserve || showCount ? (
          // Fantasma invisível com o MAIOR valor da contagem: o inline-grid
          // dá a largura ("000/000") e a contagem real fica sobreposta na
          // mesma célula, alinhada à direita. A coluna auto do grid do pai
          // assume a largura do maior step e não muda durante o run.
          <span className="inline-grid text-xs tabular-nums text-muted-foreground">
            <span
              aria-hidden="true"
              className="invisible col-start-1 row-start-1 whitespace-nowrap text-right"
            >
              {reserve}
            </span>
            {showCount ? (
              <span className="col-start-1 row-start-1 whitespace-nowrap text-right">
                {stepCount(step)}
              </span>
            ) : null}
          </span>
        ) : null}
        {icon}
      </span>
      {step.state === 'error' && step.error ? (
        <p className="col-span-3 mt-0.5 truncate text-xs text-destructive" title={step.error}>
          {step.error}
        </p>
      ) : null}
    </div>
  );
}

/** Estado final do batch: resultado geral + resumo de arquivos/destino. */
function BatchResult({ batch }: { batch: ProgressBatch }) {
  const failed = batch.failed ?? false;
  const withIssues = batch.issueCount > 0;
  return (
    // min-w-0: o BatchResult é um ITEM do grid do DialogContent, e item de
    // grid nasce com min-width:auto (mínimo = min-content). Sem isso o
    // destino longo abaixo empurraria a coluna e o modal "cresceria" além
    // do sm:max-w-md do chamador.
    <div className="min-w-0 space-y-2 py-2" role="status">
      <div className="flex justify-center">
        {failed ? (
          <XCircle className="size-14 text-destructive" aria-label="Operação falhou" />
        ) : withIssues ? (
          <AlertTriangle className="size-14 text-warning" aria-label="Concluído com falhas" />
        ) : (
          <CheckCircle2 className="size-14 text-success" aria-label="Concluído" />
        )}
      </div>
      <p className="text-center text-sm text-muted-foreground">
        {failed
          ? 'Operação interrompida.'
          : withIssues
            ? `Concluído com ${batch.issueCount} falha(s).`
            : 'Processamento concluído.'}
      </p>
      {batch.written && batch.written > 0 ? (
        // wrap-anywhere (e não o wrap-break-word usado nos diálogos de
        // import): o destino pode ser UM caminho absoluto sem espaços —
        // export de um kind só, o caso da aba Eternal Calm, que exporta
        // apenas Eventos e por isso mostra o path completo do artefato em
        // vez de "N arquivos em <pasta>". Palavra sem espaço não quebra com
        // break-word sem estourar o grid: anywhere quebra em qualquer
        // carácter E reduz o min-content, então o modal mantém a largura e
        // o texto desce para a linha de baixo.
        <p className="text-center text-xs tabular-nums text-muted-foreground wrap-anywhere">
          {fmt(batch.written)} arquivo(s){batch.destination ? ':' : ''}
          {batch.destination ? (
            <span className="mt-0.5 block">{batch.destination}</span>
          ) : null}
        </p>
      ) : null}
    </div>
  );
}

/**
 * O modal de progresso: escuta os eventos do wails (ProgressStart abre,
 * ProgressUpdate ticka, ProgressEnd congela no resultado) e renderiza o
 * snapshot da store. Um batch agrupado = título + "Ação k/N" + um step por
 * ação com a própria barra (p.ex. "Eventos 0→301/301").
 */
export function ProgressModal() {
  const batch = useProgressBatch();
  // Fechamento é do usuário: a caixa fica aberta no estado de resultado
  // até o Fechar (ou o clique fora, liberado só quando concluído).
  const [dismissed, setDismissed] = useState(false);

  useWailsEvent('ProgressStart', (data) => {
    setDismissed(false);
    progressStarted(data as ProgressBatch);
  });
  useWailsEvent('ProgressUpdate', (data) => progressUpdated(data as ProgressBatch));
  useWailsEvent('ProgressEnd', (data) => progressEnded(data as ProgressBatch));

  // Batch novo (start) reseta o fechado; store limpa quando fecha.
  useEffect(() => {
    if (batch && !batch.done) setDismissed(false);
  }, [batch]);

  const open = batch !== null && !dismissed;
  const done = batch?.done ?? false;
  const action = batch
    ? `${Math.min(batch.current + 1, batch.steps.length)} de ${batch.steps.length}`
    : '';
  const anyIssue = (batch?.issueCount ?? 0) > 0 || (batch?.failed ?? false);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen && done) {
          setDismissed(true);
          clearProgressBatch();
        }
      }}
    >
      <DialogContent
        className="sm:max-w-md"
        onInteractOutside={(e) => e.preventDefault()}
        showCloseButton={done}
        onEscapeKeyDown={(e) => {
          if (!done) e.preventDefault();
        }}
      >
        <DialogHeader>
          <DialogTitle>{batch?.title ?? 'Processando…'}</DialogTitle>
          <DialogDescription>
            {done
              ? anyIssue
                ? 'Concluído com falhas — confira as ações marcadas.'
                : 'Todas as ações concluídas.'
              : `Ação ${action} em andamento.`}
          </DialogDescription>
        </DialogHeader>
        {batch ? (
          done ? (
            <BatchResult batch={batch} />
          ) : (
            <div className="grid grid-cols-[minmax(0,auto)_minmax(0,1fr)_auto] gap-x-3 gap-y-2.5 py-1">
              {batch.steps.map((step) => (
                <StepRow key={step.label} step={step} />
              ))}
            </div>
          )
        ) : null}
        {done ? (
          <div className="flex justify-end">
            <Button
              onClick={() => {
                setDismissed(true);
                clearProgressBatch();
              }}
            >
              Fechar
            </Button>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
