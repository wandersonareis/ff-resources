// Catálogo de tags nomeadas (PC/MCR/BUTTON/ICON) para o autocomplete do
// editor e validação das tags digitadas à mão.
//
// A validação tem 2 camadas:
//   1. gramática: espelho de converter.ParseCommand (string_to_bytes.go) —
//      o que o backend NÃO reconhece como comando vira texto literal com
//      chaves no binário (round-trip não garantido);
//   2. valores: os índices de PC/MCR/BUTTON/ICON precisam existir no catálogo
//      da versão — caso contrário o re-import reescreve a tag (o decoder
//      normaliza o índice/nome) e o texto do tradutor muda sozinho.

import type { GameApi, TagCatalog } from '../api/GameApi';
import {
  colorsForVersion,
  type ColorDefinition,
  extractChipLabel,
  isFf13Version,
  isLockedTagInner,
} from './game-text-tags';

export type { TagCatalog };

/** Tags nomeadas sugeridas pelo autocomplete (estágio 1). */
export const TAG_NAMES = ['PC', 'MCR', 'BUTTON', 'ICON'] as const;
export type TagName = (typeof TAG_NAMES)[number];

/**
 * Classes do FF13 (codec `ztr/encoding`): pares de bytes com hex
 * autoritativo. Entram no estágio 1 quando o catálogo da versão as serve —
 * o FFX nunca as emite, então a lista efetiva vem do catálogo.
 */
export const FF13_TAG_NAMES = [
  'CLR',
  'ICON',
  'BTN',
  'VAR',
  'CTR',
  'KEY',
  'PAGE',
  'NL',
  'SPACE',
] as const;

/** Máximo de itens do estágio 2 por query (filtre para ver mais). */
const MAX_VALUES = 200;

/** Itens exibidos no popup do autocomplete. */
export interface TagSuggestionItem {
  kind: 'tag' | 'value';
  /** Texto inserido ao selecionar: prefixo "{PC:" (estágio 1) ou tag completa. */
  insert: string;
  /** Linha principal do item. */
  label: string;
  /** Linha secundária (detalhe/preview). */
  detail: string;
  /** Atributos do chip atômico a inserir (só no estágio 2). */
  chip?: { value: string; label: string; locked: boolean; invalid: boolean };
  /** A seleção insere uma quebra de linha (hardBreak), não um chip. */
  hardBreak?: boolean;
}

/** true quando a tag interna sobrevive ao round-trip texto→bytes→texto. */
export type ChipValidator = (inner: string) => boolean;

// ---------------------------------------------------------------------------
// carga do catálogo (cacheada por versão)
// ---------------------------------------------------------------------------

const catalogCache = new Map<string, Promise<TagCatalog | null>>();

/**
 * Carrega o catálogo da versão/idioma uma única vez (promessa cacheada). O
 * erro de paleta NÃO é engolido: o editor mostra a mensagem e desabilita as
 * cores. A promessa rejeitada sai do cache para permitir retry.
 */
export function loadTagCatalog(
  version: string,
  api: GameApi,
  lang = 'us',
): Promise<TagCatalog | null> {
  const key = `${version}\u0000${lang}`;
  let pending = catalogCache.get(key);
  if (!pending) {
    pending = api.getTagCatalog(version, lang).then((catalog) => catalog ?? null);
    pending.catch(() => catalogCache.delete(key));
    catalogCache.set(key, pending);
  }
  return pending;
}

/**
 * Paleta de cores do editor. No FF13 vem do catálogo (com os bytes do idioma);
 * null = paleta indisponível, nunca cai numa paleta de outro idioma.
 */
export function paletteFromCatalog(
  catalog: TagCatalog | null,
  version: string,
): ColorDefinition[] | null {
  const entry = catalog?.tags?.find((t) => t.tag.toUpperCase() === 'CLR');
  if (entry?.values?.length) {
    // A paleta é a lista do catálogo; dedupe por tag para que um payload com
    // entradas repetidas não infle o popover.
    const seen = new Set<string>();
    const palette: ColorDefinition[] = [];
    for (const value of entry.values) {
      if (seen.has(value.tag)) continue;
      seen.add(value.tag);
      const color: ColorDefinition = {
        label: value.label,
        name: value.label,
        hex: value.hex ?? '#9ca3af',
        tag: value.tag,
      };
      if (value.group) color.group = value.group;
      if (value.semiTransparent) color.semiTransparent = true;
      if (value.reset) color.reset = true;
      palette.push(color);
    }
    return palette;
  }
  if (!isFf13Version(version)) return colorsForVersion(version);
  return null;
}

// ---------------------------------------------------------------------------
// gramática: espelho de converter.ParseCommand
// ---------------------------------------------------------------------------

const EXACT_TAGS = new Set([
  'PAUSE',
  'BREAK',
  '\\n',
  'TEXT_NEWLINE',
  'TEXT_ITALIC',
  'TEXT_NORMAL',
  'BLANK05',
  'BLANK0C',
  'BLANK0F',
  'BLANK11',
  'CHOICE-END',
]);

// Um padrão por caso do switch do ParseCommand, na mesma ordem de exigência:
// hex de 1–2 dígitos onde o backend faz ParseUint(...,16,8) e o resto
// aceito/ignorado apenas quando o decode re-emite idêntico.
const GRAMMAR_RULES: RegExp[] = [
  /^(?:SPACE|TIME):[0-9A-Fa-f]{1,2}$/,
  /^CLR:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}(?::.*)?$/, // FF13 (par + nome)
  /^CLR:.+$/, // FFX (nome; o validador do FF13 exige os bytes)
  /^COLOR:.+$/, // legado de display; o codec FF13 não aceita
  /^(?:ICON|BUTTON):[0-9A-Fa-f]{1,2}:.*$/, // FFX (1 byte + nome)
  /^(?:ICON|BTN):[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}(?::.*)?$/, // FF13 (par + nome)
  /^CHOICE:[0-9A-Fa-f]{1,2}$/,
  /^VAR:[0-9A-Fa-f]{1,2}$/, // FFX
  /^(?:VAR|CTR|KEY|PAGE|NL|SPACE):[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}$/, // FF13
  /^PC:[0-9A-Fa-f]{1,2}:.*$/,
  /^MCR:s[0-9A-Fa-f]{1,2}:l[0-9A-Fa-f]{1,2}:.*$/,
  /^KEY:[0-9A-Fa-f]{2}$/,
  /^CMD:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}$/,
  /^RAW:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}$/, // FF13: bytes crus
  /^B:[0-9A-Fa-f]{2}$/, // FF13: byte isolado
  /^UNKCHR:[0-9A-Fa-f]{2}$/,
  /^UNKDBLCHR:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}$/,
  /^HEX:[0-9A-Fa-f]{2}(?::[0-9A-Fa-f]{2})*$/,
  /^PUA:[0-9A-Fa-f]+(?::.*)?$/,
];

const HEX2 = /^[0-9A-Fa-f]{2}$/;

/** O backend importa `inner` como comando (e não como texto literal)? */
function matchesGrammar(inner: string): boolean {
  if (EXACT_TAGS.has(inner)) return true;
  if (GRAMMAR_RULES.some((rule) => rule.test(inner))) return true;
  // MACRO: exige len >= 12 e hex em cmd[7:9] e cmd[10:12] (mesmo corte do
  // backend; o prefixo legado não é emitido pelo decoder).
  if (inner.startsWith('MACRO:')) {
    return inner.length >= 12 && HEX2.test(inner.slice(7, 9)) && HEX2.test(inner.slice(10, 12));
  }
  return false;
}

// ---------------------------------------------------------------------------
// valores: pertence ao catálogo da versão?
// ---------------------------------------------------------------------------

/** Identidade do valor dentro do catálogo: ["PC","01"] / ["MCR","s01:l05"]. */
function semanticKey(inner: string): [tag: string, key: string] | null {
  // FF13 primeiro: o par de bytes (`ICON:F0:40:Clock`) também casa o padrão
  // do FFX (`ICON:01:Nome`) — e o par é a identidade que manda lá.
  const pair = /^(CLR|ICON|BTN|VAR|CTR|KEY|PAGE|NL|SPACE):([0-9A-Fa-f]{2}:[0-9A-Fa-f]{2})(?::|$)/.exec(inner);
  if (pair) return [pair[1], pair[2].toUpperCase()];
  const color = /^COLOR:(.+)$/.exec(inner);
  if (color) return ['COLOR', color[1].trim().toUpperCase()];
  const named = /^(PC|BUTTON|ICON):([0-9A-Fa-f]{1,2}):/.exec(inner);
  if (named) return [named[1], named[2].toUpperCase()];
  const mcr = /^MCR:s([0-9A-Fa-f]{1,2}):l([0-9A-Fa-f]{1,2}):/.exec(inner);
  if (mcr) return ['MCR', `s${mcr[1].toUpperCase()}:l${mcr[2].toUpperCase()}`];
  return null;
}

function buildKeySets(catalog: TagCatalog): Map<string, Set<string>> {
  const sets = new Map<string, Set<string>>();
  for (const entry of catalog.tags ?? []) {
    const keys = new Set<string>();
    for (const value of entry.values ?? []) keys.add(value.key.toLowerCase());
    sets.set(entry.tag.toUpperCase(), keys);
  }
  return sets;
}

/**
 * Cria o validador de chips para o catálogo. `null` → só gramática (usado
 * quando o catálogo não carregou). A mesma instância deve valer para o parse
 * e para a conversão de tags digitadas, senão o HTML divergiria do valor.
 */
export function createChipValidator(catalog: TagCatalog | null): ChipValidator {
  const sets = catalog ? buildKeySets(catalog) : null;
  // O FF13 serve CLR com bytes: a forma só-nome é do FFX e não codifica lá.
  const hasColorPair = !!catalog?.tags?.some((t) => t.tag.toUpperCase() === 'CLR');
  const colorPairRe = /^CLR:[0-9A-Fa-f]{1,2}:[0-9A-Fa-f]{1,2}(?::|$)/i;
  return (inner: string) => {
    const trimmed = inner.trim();
    if (!matchesGrammar(trimmed)) return false;
    if (hasColorPair && /^CLR:/i.test(trimmed) && !colorPairRe.test(trimmed)) {
      return false;
    }
    if (!sets) return true;
    const semantic = semanticKey(trimmed);
    if (!semantic) return true;
    const keys = sets.get(semantic[0]);
    return keys ? keys.has(semantic[1].toLowerCase()) : false;
  };
}

const validatorCache = new WeakMap<TagCatalog, ChipValidator>();
let structuralValidator: ChipValidator | null = null;

/** Validador cacheado por catálogo (uma instância = um documento coerente). */
export function getChipValidator(catalog: TagCatalog | null): ChipValidator {
  if (!catalog) {
    structuralValidator ??= createChipValidator(null);
    return structuralValidator;
  }
  let validator = validatorCache.get(catalog);
  if (!validator) {
    validator = createChipValidator(catalog);
    validatorCache.set(catalog, validator);
  }
  return validator;
}

// ---------------------------------------------------------------------------
// queries do suggestion
// ---------------------------------------------------------------------------

/**
 * A query parece uma tentativa de tag nomeada? Controla o `shouldShow` do
 * suggestion: texto corrido após `{` não abre o popup nem marca a decoração.
 */
export function looksLikeTagAttempt(query: string, catalog: TagCatalog | null = null): boolean {
  if (query === '') return true;
  const names = stage1Names(catalog);
  const colon = query.indexOf(':');
  if (colon >= 0) {
    return names.includes(query.slice(0, colon).toUpperCase());
  }
  const upper = query.toUpperCase();
  return names.some((name) => name.startsWith(upper));
}

/**
 * A query é um parcial de tag que o autocomplete pode ter gerado e que deve
 * ser removido ao desistir (Escape/clique fora)? `""` (só `{`), prefixo de
 * tag nomeada ("P", "MCR") ou `TAG:` + filtro. Query já fechada com `}` NÃO
 * é parcial: ou o usuário digitou a tag inteira ou ela já virou chip.
 */
export function isPartialTagQuery(query: string, catalog: TagCatalog | null = null): boolean {
  if (query.endsWith('}')) return false;
  return looksLikeTagAttempt(query, catalog);
}

/**
 * Nomes do estágio 1: os do catálogo da versão quando ele carregou (o FFX
 * serve PC/MCR/BUTTON/ICON; o FF13 serve CLR/ICON/BTN/VAR/CTR/KEY/PAGE/NL/
 * SPACE), com fallback para a lista estática do FFX.
 */
function stage1Names(catalog: TagCatalog | null): string[] {
  const fromCatalog = (catalog?.tags ?? [])
    .map((t) => t.tag.toUpperCase())
    .filter((name) => name.length > 0);
  return fromCatalog.length > 0 ? fromCatalog : [...TAG_NAMES];
}

/**
 * Itens do popup para a query atual. Query sem `:` → estágio 1 (nomes);
 * `TAG:filtro` → valores do catálogo filtrados por índice ou texto.
 */
export function computeSuggestions(query: string, catalog: TagCatalog | null): TagSuggestionItem[] {
  const colon = query.indexOf(':');

  if (colon < 0) {
    const upper = query.toUpperCase();
    return stage1Names(catalog)
      .filter((name) => name.startsWith(upper))
      .map((name) => {
        const entry = catalog?.tags?.find((t) => t.tag.toUpperCase() === name);
        return {
          kind: 'tag' as const,
          insert: entry?.prefix ?? `{${name}:`,
          label: name,
          detail: entry?.hint ?? '',
        };
      });
  }

  const name = query.slice(0, colon).toUpperCase();
  if (!stage1Names(catalog).includes(name)) return [];
  const entry = catalog?.tags?.find((t) => t.tag.toUpperCase() === name);
  if (!entry) return [];

  const filter = query
    .slice(colon + 1)
    .trim()
    .toLowerCase();
  const validator = getChipValidator(catalog);
  const newline = name === 'NL';
  return (entry.values ?? [])
    // White é reset (botão do toolbar), não uma cor de paleta; Ex fica no
    // grupo secundário do popover.
    .filter(
      (value) =>
        name !== 'CLR' || (value.group !== 'ex' && value.reset !== true),
    )
    .filter(
      (value) =>
        !filter ||
        value.key.toLowerCase().includes(filter) ||
        value.label.toLowerCase().includes(filter),
    )
    .slice(0, MAX_VALUES)
    .map((value) => {
      const inner = value.tag.slice(1, -1);
      return {
        kind: 'value' as const,
        insert: value.tag,
        label: newline ? 'Nova linha' : value.label,
        // MCR: a tag completa repete o texto; mostra só sXX:lYY como detalhe.
        detail: name === 'MCR' ? value.key : value.tag,
        hardBreak: newline,
        chip: newline
          ? undefined
          : {
              value: value.tag,
              label: extractChipLabel(inner),
              locked: isLockedTagInner(inner),
              invalid: !validator(inner),
            },
      };
    });
}
