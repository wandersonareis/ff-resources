import { ChevronDown, Palette, RotateCcw } from 'lucide-react';
import { Button } from '../components/ui/button';
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '../components/ui/popover';
import type { ColorDefinition } from './game-text-tags';

export interface ColorPopoverProps {
  /** Paleta do idioma; null = indisponível. */
  colors: readonly ColorDefinition[] | null;
  /** Tag da cor no caret (para marcar o swatch ativo). */
  activeTag: string | null;
  /** Erro de paleta (ex.: idioma sem tabela). */
  error?: string | null;
  /** Idioma da paleta (usado nas mensagens). */
  lang: string;
  onSelect: (color: ColorDefinition) => void;
  onReset: () => void;
}

/**
 * Botão único de cor com popover. A paleta útil (nomeadas) fica no grid
 * principal; as Ex — semitransparentes e majoritariamente sem uso — ficam
 * num grupo recolhido, com xadrez/opacidade para indicar transparência.
 */
export function ColorPopover({
  colors,
  activeTag,
  error = null,
  lang,
  onSelect,
  onReset,
}: ColorPopoverProps) {
  // White é RESET (volta à cor natural), não uma cor de paleta — igual ao FFX.
  // Ela fica fora do grid e o botão Resetar usa a tag White do idioma.
  const white =
    colors?.find((color) => color.reset === true) ??
    colors?.find((color) => color.name.toUpperCase() === 'WHITE');
  const main = colors?.filter((color) => color.group !== 'ex' && color !== white) ?? [];
  const ex = colors?.filter((color) => color.group === 'ex') ?? [];
  const unavailable =
    error ?? (colors === null ? `Paleta de cores indisponível para o idioma ${lang}.` : null);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          aria-label="Cor do texto"
          title={unavailable ?? 'Cor do texto'}
        >
          <Palette size={20} />
          <ChevronDown size={14} />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-96">
        <PopoverHeader>
          <PopoverTitle>Cor do texto</PopoverTitle>
          <PopoverDescription>{colorHint(main.length, error, lang)}</PopoverDescription>
        </PopoverHeader>

        {error ? (
          <div
            role="alert"
            className="rounded-md border border-destructive/45 bg-destructive/10 p-2 text-xs text-destructive"
          >
            {error}
          </div>
        ) : !colors || colors.length === 0 ? (
          <div
            role="alert"
            className="rounded-md border border-destructive/45 bg-destructive/10 p-2 text-xs text-destructive"
          >
            Paleta de cores indisponível para o idioma {lang}.
          </div>
        ) : (
          <>
            {main.length > 0 ? (
              <ColorGrid colors={main} activeTag={activeTag} onSelect={onSelect} />
            ) : (
              <div className="text-xs text-muted-foreground">Nenhuma cor nomeada.</div>
            )}

            {ex.length > 0 && (
              <details className="rounded-md border border-border p-1.5">
                <summary className="cursor-pointer select-none text-xs font-medium text-muted-foreground">
                  Ex — semitransparentes ({ex.length})
                </summary>
                <div className="mt-1.5">
                  <ColorGrid colors={ex} activeTag={activeTag} onSelect={onSelect} maxHeight="15rem" />
                </div>
              </details>
            )}

            <div className="flex items-center justify-between gap-2 border-t pt-2">
              <span className="min-w-0 truncate text-xs text-muted-foreground">
                {activeTag ?? 'Sem cor'}
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={onReset}
                aria-label="Resetar cor"
                title={white?.tag ?? 'Resetar cor (White)'}
              >
                <RotateCcw size={14} />
                Resetar (White)
              </Button>
            </div>
          </>
        )}
      </PopoverContent>
    </Popover>
  );
}

function ColorGrid({
  colors,
  activeTag,
  onSelect,
  maxHeight = '20rem',
}: {
  colors: readonly ColorDefinition[];
  activeTag: string | null;
  onSelect: (color: ColorDefinition) => void;
  maxHeight?: string;
}) {
  return (
    <div
      className="grid grid-cols-4 gap-1 overflow-y-auto p-0.5"
      style={{ maxHeight }}
    >
      {colors.map((color) => {
        const active = activeTag !== null && activeTag === color.tag;
        return (
          <button
            key={color.tag ?? color.name}
            type="button"
            className="flex min-w-0 cursor-pointer flex-col items-center gap-1 rounded p-1 transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
            title={color.tag ?? `Cor ${color.label}`}
            aria-label={`Cor ${color.label}`}
            aria-pressed={active}
            onClick={() => onSelect(color)}
          >
            <span
              className="h-7 w-7 rounded border-2"
              style={{
                backgroundColor: color.hex,
                borderColor: active ? 'currentColor' : 'var(--border)',
                boxShadow: active ? '0 0 0 1px currentColor' : 'none',
                opacity: color.semiTransparent ? 0.72 : 1,
                backgroundImage: color.semiTransparent ? SEMI_TRANSPARENT : undefined,
                backgroundSize: color.semiTransparent ? '8px 8px' : undefined,
                backgroundPosition: color.semiTransparent ? '0 0, 4px 4px' : undefined,
              }}
            />
            <span className="w-full truncate text-center text-[10px] leading-tight">
              {color.label}
            </span>
          </button>
        );
      })}
    </div>
  );
}

const SEMI_TRANSPARENT =
  'linear-gradient(45deg, rgba(0,0,0,.18) 25%, transparent 25%, transparent 75%, rgba(0,0,0,.18) 75%), linear-gradient(45deg, rgba(0,0,0,.18) 25%, transparent 25%, transparent 75%, rgba(0,0,0,.18) 75%)';

function colorHint(mainCount: number, error: string | null, lang: string): string {
  if (error) return 'Paleta indisponível';
  return `${mainCount} cores · idioma ${lang}`;
}
