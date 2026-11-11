/**
 * Separa o "padding de tela" do início de um texto canônico: a sequência
 * contínua de espaços, tabs, \n/\r e tags de quebra ({TEXT_NEWLINE} ou
 * {NL:B1:B2}) com que o jogo alinha o texto na tela.
 *
 * Somente a UI usa isto (badge no editor / indicador no preview). O valor
 * canônico serializado continua incluindo o padding — nada é removido de
 * fato do dado.
 */

import { newlineTokenAt } from './line-breaks';

export interface LeadingPadding {
  /** Prefixo bruto (espaços, \n/\r, tags de quebra) preservado. */
  padding: string;
  /** Texto sem o prefixo, para edição/exibição. */
  body: string;
}

const NEWLINE_TAG_PATTERN =
  String.raw`(?:\{TEXT_NEWLINE\}|\{NL:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}\})`;
const LEADING_PADDING_RE = new RegExp(`^(?:\\s|${NEWLINE_TAG_PATTERN})+`, 'i');

export function splitLeadingPadding(raw: string): LeadingPadding {
  if (!raw) return { padding: '', body: '' };
  const match = raw.match(LEADING_PADDING_RE);
  if (!match) return { padding: '', body: raw };
  return { padding: match[0], body: raw.slice(match[0].length) };
}

export function summarizePadding(padding: string): string {
  if (!padding) return '';
  let textNewlines = 0;
  let nlNewlines = 0;
  let realNewlines = 0;
  let spaces = 0;
  let i = 0;
  while (i < padding.length) {
    const token = newlineTokenAt(padding, i);
    if (token) {
      const upper = token.toUpperCase();
      if (upper === '{TEXT_NEWLINE}') textNewlines++;
      else if (upper.startsWith('{NL:')) nlNewlines++;
      else realNewlines++;
      i += token.length;
      continue;
    }
    spaces++;
    i++;
  }
  const parts: string[] = [];
  const newlines = textNewlines + nlNewlines + realNewlines;
  if (newlines > 0) {
    const label =
      nlNewlines > 0 && textNewlines === 0 && realNewlines === 0
        ? '{NL}'
        : nlNewlines === 0
          ? '{TEXT_NEWLINE}'
          : 'quebra';
    parts.push(`${label} ×${newlines}`);
  }
  if (spaces > 0) parts.push(`␣ ×${spaces}`);
  return parts.join(' · ');
}
