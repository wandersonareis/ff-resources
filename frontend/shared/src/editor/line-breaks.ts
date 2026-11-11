import { isFf13Version } from './game-text-tags';

/**
 * Tokens de quebra de linha do texto canônico:
 * - `{TEXT_NEWLINE}` é o nome histórico do editor/FFX;
 * - `{NL:B1:B2}` é o par que o codec FF13 emite (normalmente `{NL:40:72}`).
 *
 * O parser, o serializer e os helpers de busca/padding tratam os dois como a
 * MESMA quebra. O serializer escolhe a forma da versão.
 */
export const TEXT_NEWLINE_TAG = '{TEXT_NEWLINE}';

const NL_TAG = /\{NL:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}\}/i;
const NL_TAG_AT = /^\{NL:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}\}/i;
const TEXT_NEWLINE_AT = /^\{TEXT_NEWLINE\}/i;

/** Regex de split (grupo de captura) para as duas formas + CRLF/LF/CR. */
export const NEWLINE_SPLIT_RE =
  /(\r?\n|\{TEXT_NEWLINE\}|\{NL:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}\})/gi;

/** Tag de quebra que o serializer deve emitir para a versão. */
export function newlineTag(version: string): string {
  return isFf13Version(version) ? '{NL:40:72}' : TEXT_NEWLINE_TAG;
}

/** O token inteiro é uma quebra de linha (tag ou \n/\r/\r\n)? */
export function isNewlineToken(token: string): boolean {
  return (
    token === '\n' ||
    token === '\r' ||
    token === '\r\n' ||
    TEXT_NEWLINE_AT.test(token) ||
    NL_TAG.test(token)
  );
}

/** Token de quebra na posição, ou null. Devolve o texto exato do token. */
export function newlineTokenAt(text: string, index: number): string | null {
  if (text[index] === '\r') return text[index + 1] === '\n' ? '\r\n' : '\r';
  if (text[index] === '\n') return '\n';
  if (text[index] !== '{') return null;
  const slice = text.slice(index);
  const textNewline = TEXT_NEWLINE_AT.exec(slice);
  if (textNewline) return textNewline[0];
  const nl = NL_TAG_AT.exec(slice);
  return nl ? nl[0] : null;
}
