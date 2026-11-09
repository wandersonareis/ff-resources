// Tags canônicas do backend (backend/core/converter/string_bytes.go +
// string_to_bytes.go). O diretório backend/core/encoding/tags é legado e
// NÃO é usado aqui. Terceiro campo das tags ({X:YY:Nome}) é só humano.

export interface ColorDefinition {
  label: string;
  name: string;
  hex: string;
}

// Nomes canônicos em MAIÚSCULAS, como emitidos por GetColorString.
// Hexes reaproveitados do protótipo tiptap-color-tag-editor.
export const KNOWN_COLORS: Record<string, ColorDefinition> = {
  BLUE: { label: 'Azul', name: 'BLUE', hex: '#2563eb' },
  RED: { label: 'Vermelho', name: 'RED', hex: '#dc2626' },
  YELLOW: { label: 'Amarelo', name: 'YELLOW', hex: '#eab308' },
  GREY: { label: 'Cinza', name: 'GREY', hex: '#6b7280' },
  PINK: { label: 'Rosa', name: 'PINK', hex: '#ec4899' },
  OL_PURPLE: { label: 'Roxo', name: 'OL_PURPLE', hex: '#9333ea' },
  OL_CYAN: { label: 'Ciano', name: 'OL_CYAN', hex: '#06b6d4' },
  WHITE: { label: 'Branco (reset)', name: 'WHITE', hex: '#ffffff' },
};

/**
 * Paleta do FF13 (trilogia XIII) para o `{COLOR:Nome}` do codec novo. Os
 * hexes são só para o CHIP da UI — a cor real sai do jogo. `EXTRA00..28` são
 * slots que o ZTRtool deixou sem nome no ff131: cinza neutro até nomearem.
 */
const FF13_COLOR_HEX: Record<string, string> = {
  White: '#ffffff',
  IceBlue: '#a5d8e6',
  Gold: '#e8c15a',
  LightRed: '#e8736b',
  Yellow: '#f2d24a',
  Green: '#6fbf73',
  Gray: '#9aa0a6',
  LightGold: '#f0dfa0',
  Rose: '#e58fb0',
  Purple: '#9b6bc7',
  DarkYellow: '#b89a2e',
  Gray2: '#7d838a',
  Voilet: '#8a6bb5',
  LightGreen: '#a8dda8',
  Sapphire: '#3a6fd8',
  Voilet2: '#7a5fb0',
  OliveGreen: '#8a9a4a',
  DarkCyan: '#3f7a80',
  Lavender: '#c9b8e0',
  Brown: '#8a5a30',
  Gold2: '#d4af37',
  Gold3: '#c9a227',
  DarkGray: '#5f666d',
  DarkRed: '#9c2b2b',
  Jade: '#4fa88a',
  SmokeGray: '#b5babd',
  DarkGold: '#a8842c',
  Magenta: '#c74bb0',
  PureWhite: '#ffffff',
  Orange: '#e8913a',
  NavyBlue: '#2c3e70',
};

/** Paleta completa do FF13 (EXTRA00..28 entram como cinza neutro). */
export const FF13_KNOWN_COLORS: Record<string, ColorDefinition> = (() => {
  const out: Record<string, ColorDefinition> = {};
  for (const [name, hex] of Object.entries(FF13_COLOR_HEX)) {
    out[name.toUpperCase()] = { label: name, name, hex };
  }
  for (let i = 0; i <= 28; i++) {
    const name = `EXTRA${String(i).padStart(2, '0')}`;
    out[name] = { label: name, name, hex: '#9ca3af' };
  }
  return out;
})();

/** true para as versões da trilogia FF13 (part1/2/3). */
export function isFf13Version(version: string): boolean {
  return version === 'part1' || version === 'part2' || version === 'part3';
}

/** Paleta de cores da versão (FFX vs. FF13), sem o "reset". */
export function colorsForVersion(version: string): ColorDefinition[] {
  const table = isFf13Version(version) ? FF13_KNOWN_COLORS : KNOWN_COLORS;
  return Object.values(table).filter((c) => !c.name.toUpperCase().endsWith('WHITE'));
}

export const COLOR_RESET_TAG = 'WHITE';

/**
 * Nome da paleta a partir do byte canônico de cor (o decode emite `%02X` para
 * bytes fora do mapa: `{CLR:88}` = BLUE). Serve para reconhecer esses bytes
 * sem transformar em chip genérico.
 */
const COLOR_NAME_BY_BYTE: Record<string, string> = {
  '41': 'WHITE',
  '43': 'YELLOW',
  '52': 'GREY',
  '88': 'BLUE',
  '94': 'RED',
  '97': 'PINK',
  A1: 'OL_PURPLE',
  B1: 'OL_CYAN',
};

interface RgbColor {
  r: number;
  g: number;
  b: number;
}

/** Nomes CSS básicos -> hex (o CSSOM pode devolver nome em vez de rgb()). */
const CSS_NAMED_COLORS: Record<string, string> = {
  black: '#000000',
  silver: '#c0c0c0',
  gray: '#808080',
  grey: '#808080',
  white: '#ffffff',
  maroon: '#800000',
  red: '#ff0000',
  purple: '#800080',
  fuchsia: '#ff00ff',
  magenta: '#ff00ff',
  green: '#008000',
  lime: '#00ff00',
  olive: '#808000',
  yellow: '#ffff00',
  navy: '#000080',
  blue: '#0000ff',
  teal: '#008080',
  aqua: '#00ffff',
  cyan: '#00ffff',
  orange: '#ffa500',
  pink: '#ffc0cb',
  brown: '#a52a2a',
  gold: '#ffd700',
  indigo: '#4b0082',
  violet: '#ee82ee',
  beige: '#f5f5dc',
  coral: '#ff7f50',
  crimson: '#dc143c',
  darkblue: '#00008b',
  darkgray: '#a9a9a9',
  darkgreen: '#006400',
  darkred: '#8b0000',
  lightblue: '#add8e6',
  lightgray: '#d3d3d3',
  lightgreen: '#90ee90',
  limegreen: '#32cd32',
  royalblue: '#4169e1',
  skyblue: '#87ceeb',
  slategray: '#708090',
  tomato: '#ff6347',
  turquoise: '#40e0d0',
};

function clampByte(value: number): number {
  return Math.max(0, Math.min(255, Math.round(value)));
}

function hexToRgb(hex: string): RgbColor | null {
  let digits = hex.trim().replace(/^#/, '');
  if (digits.length === 3 || digits.length === 4) {
    digits = digits
      .slice(0, 3)
      .split('')
      .map((digit) => digit + digit)
      .join('');
  } else if (digits.length === 6 || digits.length === 8) {
    digits = digits.slice(0, 6);
  } else {
    return null;
  }
  if (!/^[0-9a-f]{6}$/i.test(digits)) return null;
  return {
    r: parseInt(digits.slice(0, 2), 16),
    g: parseInt(digits.slice(2, 4), 16),
    b: parseInt(digits.slice(4, 6), 16),
  };
}

function rgbToHex(color: RgbColor): string {
  return `#${[color.r, color.g, color.b]
    .map((channel) => clampByte(channel).toString(16).padStart(2, '0'))
    .join('')}`;
}

/** Componente rgb(): número puro ou percentual. */
function parseRgbComponent(part: string): number | null {
  const value = parseFloat(part);
  if (!Number.isFinite(value)) return null;
  return part.trim().endsWith('%') ? (value / 100) * 255 : value;
}

function hslToRgb(hue: number, saturation: number, lightness: number): RgbColor {
  const h = ((hue % 360) + 360) % 360;
  const s = Math.max(0, Math.min(1, saturation));
  const l = Math.max(0, Math.min(1, lightness));
  const chroma = (1 - Math.abs(2 * l - 1)) * s;
  const hp = h / 60;
  const x = chroma * (1 - Math.abs((hp % 2) - 1));
  let rgb: [number, number, number];
  if (hp < 1) rgb = [chroma, x, 0];
  else if (hp < 2) rgb = [x, chroma, 0];
  else if (hp < 3) rgb = [0, chroma, x];
  else if (hp < 4) rgb = [0, x, chroma];
  else if (hp < 5) rgb = [x, 0, chroma];
  else rgb = [chroma, 0, x];
  const m = l - chroma / 2;
  return {
    r: (rgb[0] + m) * 255,
    g: (rgb[1] + m) * 255,
    b: (rgb[2] + m) * 255,
  };
}

/**
 * Converte um valor CSS de cor (`#rgb`, `#rrggbb`, `rgb()/rgba()`,
 * `hsl()/hsla()` ou nome) em RGB. Devolve null quando não é cor.
 *
 * O Tiptap guarda em `textStyle.color` exatamente o que veio no `style` do
 * HTML — inclusive `rgb(...)` de conteúdo externo (colar/soltar HTML). Sem
 * normalizar, a serialização emitiria `{CLR:RGB(...)}` e a releitura viraria
 * chip (ver getColorTagName).
 */
export function parseCssColor(colorValue?: string | null): RgbColor | null {
  if (!colorValue) return null;
  const value = colorValue.trim().toLowerCase();
  if (!value) return null;

  if (value.startsWith('#')) return hexToRgb(value);

  const named = CSS_NAMED_COLORS[value];
  if (named) return hexToRgb(named);

  const rgbMatch = /^rgba?\(([^)]*)\)$/.exec(value);
  if (rgbMatch) {
    const parts = rgbMatch[1].split(/[,/\s]+/).filter(Boolean);
    if (parts.length < 3) return null;
    const r = parseRgbComponent(parts[0]);
    const g = parseRgbComponent(parts[1]);
    const b = parseRgbComponent(parts[2]);
    if (r === null || g === null || b === null) return null;
    return { r, g, b };
  }

  const hslMatch = /^hsla?\(([^)]*)\)$/.exec(value);
  if (hslMatch) {
    const parts = hslMatch[1].split(/[,/\s]+/).filter(Boolean);
    if (parts.length < 3) return null;
    const h = parseFloat(parts[0]);
    const s = parseFloat(parts[1]);
    const l = parseFloat(parts[2]);
    if (![h, s, l].every(Number.isFinite)) return null;
    return hslToRgb(h, s / 100, l / 100);
  }

  return null;
}

/** Cor da paleta mais próxima (o jogo só tem 8 cores). */
function nearestColorName(color: RgbColor): string {
  let best = COLOR_RESET_TAG;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const definition of Object.values(KNOWN_COLORS)) {
    const candidate = hexToRgb(definition.hex);
    if (!candidate) continue;
    const distance =
      (color.r - candidate.r) ** 2 + (color.g - candidate.g) ** 2 + (color.b - candidate.b) ** 2;
    if (distance < bestDistance) {
      bestDistance = distance;
      best = definition.name;
    }
  }
  return best;
}

/**
 * Normaliza um valor de cor (hex/rgb/hsl/nome) para o nome canônico da tag.
 * Usado na serialização: mark textStyle.color -> {CLR:NOME}.
 *
 * Nunca devolve valor fora da paleta: uma cor arbitrária (ex.: `rgb(...)` de
 * HTML colado) é aproximada para a cor do jogo mais próxima. Assim nenhuma
 * marca vira `{CLR:RGB(...)}`/`{CLR:#...}` — que o backend não sabe codificar
 * (ColorToByte devolve 0) e que o parser transformaria em chip na releitura.
 */
export function getColorTagName(colorValue?: string | null): string | null {
  if (!colorValue) return null;
  const value = colorValue.trim();
  if (!value) return null;
  const lower = value.toLowerCase();
  // FFX casa por NOME ou hex; a paleta do FF13 só entra por HEX exato — os
  // nomes dela não podem capturar um valor CSS do FFX ("gold" é YELLOW lá).
  for (const definition of Object.values(KNOWN_COLORS)) {
    if (definition.name.toLowerCase() === lower || definition.hex.toLowerCase() === lower) {
      return definition.name;
    }
  }
  for (const definition of Object.values(FF13_KNOWN_COLORS)) {
    if (definition.hex.toLowerCase() === lower) {
      return definition.name;
    }
  }
  const rgb = parseCssColor(value);
  if (!rgb) return null;
  const hex = rgbToHex(rgb);
  for (const definition of Object.values(KNOWN_COLORS)) {
    if (definition.hex.toLowerCase() === hex) return definition.name;
  }
  return nearestColorName(rgb);
}

/**
 * Resolve o nome da tag de cor para CSS. WHITE = reset.
 * Retorna null se não for tag de cor.
 *
 * Aceita o que o decoder do backend emite (`{CLR:88}`, `{CLR:41}`) e o que
 * versões anteriores do editor gravaram por engano (`{CLR:#RRGGBB}` e
 * `{CLR:RGB(...)}`): o rascunho antigo volta a ser cor em vez de chip.
 *
 * Conhece as DUAS paletas: a do FFX (`{CLR:BLUE}`) e a da trilogia FF13
 * (`{COLOR:IceBlue}`) — os nomes não colidem, então um mapa só serve.
 */
export function getResolvedColorFromTag(
  tagName: string,
): { hex: string | null; isReset: boolean } | null {
  const normalized = tagName.trim().toUpperCase();
  if (!normalized) return null;
  if (normalized === COLOR_RESET_TAG) return { hex: null, isReset: true };
  const known =
    FF13_KNOWN_COLORS[normalized] ?? KNOWN_COLORS[normalized];
  if (known) {
    if (known.name.toUpperCase() === COLOR_RESET_TAG) return { hex: null, isReset: true };
    return { hex: known.hex, isReset: false };
  }
  const byteName = COLOR_NAME_BY_BYTE[normalized];
  if (byteName) {
    if (byteName === COLOR_RESET_TAG) return { hex: null, isReset: true };
    return { hex: KNOWN_COLORS[byteName].hex, isReset: false };
  }
  if (/^#[0-9A-F]{3}$/.test(normalized) || /^#[0-9A-F]{6}$/.test(normalized)) {
    if (normalized === '#FFF' || normalized === '#FFFFFF') {
      return { hex: null, isReset: true };
    }
    return { hex: normalized, isReset: false };
  }
  const rgb = parseCssColor(normalized);
  if (rgb) {
    const hex = rgbToHex(rgb);
    return hex === '#ffffff' ? { hex: null, isReset: true } : { hex, isReset: false };
  }
  return null;
}

// Prefixos cujos chips são BLOQUEADOS (valores calculados no jogo,
// nunca editáveis): CMD, HEX e qualquer UNK*.
const LOCKED_ASIS_PREFIXES = ['CMD:', 'HEX:', 'UNKCHR:', 'UNKDBLCHR:', 'UNKTPLCHR:'];

// Classes do FF13 cujo valor é calculado no jogo: o tradutor não escolhe o
// byte, então o chip é bloqueado (VAR já era).
const FF13_LOCKED_PREFIXES = [
  'VAR:',
  'CTR:',
  'KEY:',
  'RAW:',
  'B:',
  'PAGE:',
  'NL:',
  'TAB:',
];

/**
 * Indica se a tag é bloqueada (CMD, HEX, UNK ou VAR).
 */
export function isLockedTagInner(inner: string): boolean {
  const upper = inner.trim().toUpperCase();
  if (upper === 'VAR' || upper.startsWith('VAR:')) return true;
  if (FF13_LOCKED_PREFIXES.some((p) => upper.startsWith(p))) return true;
  return LOCKED_ASIS_PREFIXES.some((p) => upper.startsWith(p));
}

/**
 * Extrai o label do chip a partir do conteúdo interno da tag:
 * - CMD/HEX/UNK*: exibe como está (label = tag completa).
 * - VAR: exibe apenas "Variável".
 * - Demais: último segmento após ':', sem aspas duplas.
 *   ex: MCR:s06:l3F:"Yevon" -> Yevon | PC:01:YUNA -> YUNA | PAUSE -> PAUSE
 *
 * Tags do FF13 sem segmento de nome (`{VAR:F4:40}`, `{RAW:81:40}`) mostram a
 * tag inteira: o último segmento seria o segundo byte do par, que não diz
 * nada a quem lê.
 */
export function extractChipLabel(inner: string): string {
  const trimmed = inner.trim();
  const upper = trimmed.toUpperCase();
  if (LOCKED_ASIS_PREFIXES.some((p) => upper.startsWith(p))) {
    return `{${trimmed}}`;
  }
  const segments = trimmed.split(':');
  // FF13 sem nome (`{VAR:F4:40}`, `{RAW:81:40}`, `{B:FF}`): o último segmento
  // seria o 2º byte do par, que não diz nada — mostra a tag inteira. Vem
  // ANTES do atalho do FFX, cujo `{VAR:01}` tem um byte só.
  const head = segments[0].toUpperCase();
  const pairClass = ['VAR', 'CTR', 'KEY', 'PAGE', 'NL', 'TAB', 'RAW', 'ICON', 'BTN'].includes(head);
  const namelessFf13 =
    (pairClass && segments.length === 3) || (head === 'B' && segments.length === 2);
  if (namelessFf13) return `{${trimmed}}`;
  if (upper === 'VAR' || upper.startsWith('VAR:')) {
    return 'Variável';
  }
  if (segments.length === 1) return trimmed || `{${trimmed}}`;
  const last = segments[segments.length - 1].trim();
  const unquoted = last.replace(/^"|"$/g, '').trim();
  return unquoted || trimmed;
}

/**
 * Classes CSS dos tiles na sprite `public/pad_icon.png` (glifos do Xbox,
 * como o jogo renderiza no PC) para tags {BUTTON:XX:Nome}. Espelha o
 * buttonMap do backend (encoding_players.go):
 *   30 TRIANGLE→Y · 31 X→A · 32 CIRCLE→B · 33 SQUARE→X (layout Xbox);
 *   34–37→LB/RB/LT/RT · 38 START→▶ · 39 SELECT→◀.
 * Os direcionais (0x40–0x4F) viram SEQUÊNCIAS de setas na ordem do nome do
 * código (same do jogo): Up+Right = ↑,→. "Direcional" (0x40) e "All"
 * (0x4F) usam o cursor ✛ do pad (sem setas = todas as direções).
 *
 * Códigos sem glifo (Dummy/Dummy2, 0x001+) devolvem [] → chip textual.
 */
const BUTTON_CODE_SPRITES: Record<number, string[]> = {
  32: ['gb-lb'], // ?L1 (SWITCH)
  48: ['gb-y'],
  49: ['gb-a'],
  50: ['gb-b'],
  51: ['gb-x'],
  52: ['gb-lb'],
  53: ['gb-rb'],
  54: ['gb-lt'],
  55: ['gb-rt'],
  56: ['gb-start'],
  57: ['gb-back'],
  64: ['gb-cursor'], // Direcional (generico: cruz de direções)
  65: ['gb-arrow-up'],
  66: ['gb-arrow-right'],
  67: ['gb-arrow-up', 'gb-arrow-right'],
  68: ['gb-arrow-down'],
  69: ['gb-arrow-up', 'gb-arrow-down'],
  70: ['gb-arrow-down', 'gb-arrow-right'],
  71: ['gb-arrow-up', 'gb-arrow-right', 'gb-arrow-down'],
  72: ['gb-arrow-left'],
  73: ['gb-arrow-up', 'gb-arrow-left'],
  74: ['gb-arrow-left', 'gb-arrow-right'],
  75: ['gb-arrow-up', 'gb-arrow-left', 'gb-arrow-right'],
  76: ['gb-arrow-left', 'gb-arrow-down'],
  77: ['gb-arrow-up', 'gb-arrow-left', 'gb-arrow-down'],
  78: ['gb-arrow-left', 'gb-arrow-down', 'gb-arrow-right'],
  79: ['gb-cursor'], // Direcional All
};

/**
 * Sequência de classes de sprite para o chip de {BUTTON:XX:…}; lista vazia
 * quando o código não tem glifo (chip segue textual). Aceita tanto o inner
 * ("BUTTON:41:…", do parser) quanto a tag completa com chaves (attrs.value
 * do node no renderHTML).
 */
export function buttonSpriteClasses(inner: string): string[] {
  const match = /^BUTTON:([0-9A-Fa-f]{1,2})(?::|$)/i.exec(inner.trim().replace(/^\{|\}$/g, ''));
  if (!match) return [];
  const code = parseInt(match[1], 16);
  if (Number.isNaN(code)) return [];
  return BUTTON_CODE_SPRITES[code] ?? [];
}
