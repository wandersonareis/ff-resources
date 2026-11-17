'use client';

import type { YkdItem, YkdResource, YkdScreen, YkdText, YkdTexture } from '@ff-resources/ui/dto';
import { useSelector } from '@tanstack/react-store';
import { Maximize2, Minus, Plus } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import type { EntryView } from '@/components/entry-view/entry-view-store';
import { Button } from '@/components/ui/button';

/**
 * Preview de uma TELA .ykd: desenha os sprites dos blocos (posição, escala,
 * rotação, pivô e flip) e placeholders dos nós de texto. O layout base vem do
 * transform dos blocos; as animações (in/loop/out) ainda não são tocadas.
 *
 * O backend já resolveu as texturas do próprio container e, quando preciso,
 * do índice global de GUI da part — o que não resolveu vira placeholder e
 * aparece na lista de "texturas ausentes".
 */
export function YkdPanel({ view }: { view: EntryView }) {
  const entry = useSelector(view.store, (s) => s.selectedEntry);
  const screen = useSelector(view.store, (s) => s.ykd);
  const loading = useSelector(view.store, (s) => s.loading);

  const [zoom, setZoom] = useState(0.6);
  const [showIds, setShowIds] = useState(false);
  const [usePivot, setUsePivot] = useState(false);
  const [wireframe, setWireframe] = useState(false);
  const [selectedBlock, setSelectedBlock] = useState<number | null>(null);
  const [hiddenBlocks, setHiddenBlocks] = useState<Set<number>>(() => new Set());
  const [draftWidth, setDraftWidth] = useState(0);
  const [applying, setApplying] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  const fit = useCallback(() => {
    const element = wrapRef.current;
    if (!element || !screen) return;
    const rect = element.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) return;
    const next = Math.min(
      (rect.width - 48) / screen.canvasWidth,
      (rect.height - 48) / screen.canvasHeight,
    );
    setZoom(Math.max(0.15, Math.min(2, next)));
  }, [screen]);

  // Encaixa ao abrir/trocar de arquivo (e quando o painel ganha tamanho).
  useEffect(() => {
    fit();
  }, [fit]);

  const resources = useMemo(() => {
    const map = new Map<number, YkdResource>();
    for (const resource of screen?.resources ?? []) map.set(resource.index, resource);
    return map;
  }, [screen]);

  const textItems = useMemo(
    () => (screen?.items ?? []).filter((item) => item.kind === 'text' && item.text),
    [screen],
  );
  const selectedTextItem =
    selectedBlock === null
      ? null
      : (textItems.find((item) => item.blockIndex === selectedBlock) ?? null);
  const selectedText = selectedTextItem?.text ?? null;

  // Sincroniza o rascunho da largura ao trocar de caixa ou após aplicar.
  useEffect(() => {
    if (selectedText) setDraftWidth(selectedText.width);
  }, [selectedBlock, selectedText?.width]);

  if (!entry) return null;
  if (loading && !screen) return <p className="mt-8 opacity-70">Carregando tela…</p>;
  if (!screen) return <p className="mt-8 opacity-70">Não foi possível carregar a tela .ykd.</p>;

  const canvasWidth = screen.canvasWidth * zoom;
  const canvasHeight = screen.canvasHeight * zoom;
  const items = screen.items ?? [];
  const blocks = screen.blocks ?? [];
  const missing = screen.missingTextures ?? [];

  const zoomBy = (delta: number) =>
    setZoom((current) => Math.max(0.15, Math.min(2, Number((current + delta).toFixed(2)))));

  const applyWidth = async () => {
    if (selectedBlock === null || !selectedText) return;
    const width = Math.round(draftWidth);
    if (!Number.isFinite(width) || width < 1 || width > selectedText.maxWidth) {
      toast.error(`A largura deve ficar entre 1 e ${selectedText.maxWidth}px.`);
      return;
    }
    setApplying(true);
    try {
      await view.actions.applyYkdText(selectedBlock, { width });
      toast.success(`Largura da caixa atualizada para ${width}px (gravado em mods/).`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    } finally {
      setApplying(false);
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col gap-3 pt-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="mr-auto flex min-w-0 flex-col">
          <span className="truncate font-medium">{entry.label}</span>
          <span className="text-muted-foreground text-xs">
            {screen.stats.blocks} blocos · {screen.stats.resources} recursos ·{' '}
            {screen.stats.texturedItems} sprites texturizados · {screen.stats.textNodes} textos
            {missing.length > 0 ? ` · ${missing.length} textura(s) ausente(s)` : ''}
          </span>
          <span className="text-muted-foreground text-[10px]">
            camada 2D do .ykd — selecione uma caixa de texto para ajustar a largura; o fundo 3D e
            o conteúdo dos textos não estão no arquivo
          </span>
        </div>

        {screen.siblings && screen.siblings.length > 1 ? (
          <select
            value={entry.id}
            onChange={(event) => {
              const sibling = screen.siblings?.find((item) => item.ref === event.target.value);
              if (!sibling) return;
              void view.actions.selectEntry({
                kind: 'ykd',
                id: sibling.ref,
                key: sibling.ref,
                label: sibling.name,
                source: entry.source,
              });
            }}
            className="max-w-[240px] rounded border bg-background px-1 py-0.5 text-xs"
            title="Trocar de tela .ykd do mesmo container"
          >
            {screen.siblings.map((sibling) => (
              <option key={sibling.ref} value={sibling.ref}>
                {sibling.name}
              </option>
            ))}
          </select>
        ) : null}

        <label className="flex cursor-pointer items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={wireframe}
            onChange={(event) => setWireframe(event.target.checked)}
          />
          wireframe
        </label>
        <label className="flex cursor-pointer items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={usePivot}
            onChange={(event) => setUsePivot(event.target.checked)}
          />
          pivô
        </label>
        <label className="flex cursor-pointer items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={showIds}
            onChange={(event) => setShowIds(event.target.checked)}
          />
          IDs
        </label>

        <div className="flex items-center gap-1">
          <Button
            size="icon"
            variant="ghost"
            className="size-7"
            title="Diminuir zoom"
            onClick={() => zoomBy(-0.1)}
          >
            <Minus size={16} />
          </Button>
          <span className="w-12 text-center text-xs tabular-nums">{Math.round(zoom * 100)}%</span>
          <Button
            size="icon"
            variant="ghost"
            className="size-7"
            title="Aumentar zoom"
            onClick={() => zoomBy(0.1)}
          >
            <Plus size={16} />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 gap-1 px-2 text-xs"
            title="Encaixar a tela no painel"
            onClick={fit}
          >
            <Maximize2 size={14} />
            Encaixar
          </Button>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 gap-3">
        <div
          ref={wrapRef}
          className="relative min-w-0 flex-1 overflow-auto rounded-md border bg-muted/40 p-4"
        >
          <div
            style={{
              width: canvasWidth,
              height: canvasHeight,
              position: 'relative',
              overflow: 'hidden',
              background:
                'repeating-conic-gradient(#17171d 0% 25%, #101015 0% 50%) 50% / 24px 24px',
            }}
          >
            {items.map((item, index) => (
              <YkdDrawItem
                key={`${item.blockIndex}:${index}`}
                item={item}
                screen={screen}
                resources={resources}
                zoom={zoom}
                usePivot={usePivot}
                showId={showIds}
                wireframe={wireframe}
                selected={selectedBlock !== null && item.blockIndex === selectedBlock}
                hidden={hiddenBlocks.has(item.blockIndex)}
                onSelect={(blockIndex) =>
                  setSelectedBlock((current) => (current === blockIndex ? null : blockIndex))
                }
              />
            ))}
          </div>
        </div>

        <aside className="flex w-72 shrink-0 flex-col gap-3 overflow-auto border-l pl-3 text-xs">
          {selectedText ? (
            <section className="rounded-md border p-2">
              <h3 className="mb-1 font-medium">Caixa de texto</h3>
              <p className="text-muted-foreground mb-2 leading-4">
                bloco #{selectedBlock} · {selectedText.fontSize}px · altura{' '}
                {selectedText.height > 0 ? `${selectedText.height}px` : 'auto'} · flags{' '}
                {selectedText.flags}
              </p>
              <label className="flex items-center gap-2">
                <span className="w-14">Largura</span>
                <input
                  type="number"
                  min={1}
                  max={selectedText.maxWidth}
                  value={draftWidth}
                  aria-label="Largura da caixa de texto"
                  onChange={(event) => setDraftWidth(Number(event.target.value))}
                  className="w-20 rounded border bg-background px-1 py-0.5 text-xs tabular-nums"
                />
                <span className="opacity-60">máx. {selectedText.maxWidth}</span>
              </label>
              <input
                type="range"
                min={1}
                max={selectedText.maxWidth}
                value={Math.min(Math.max(draftWidth, 1), selectedText.maxWidth)}
                onChange={(event) => setDraftWidth(Number(event.target.value))}
                className="mt-1 w-full"
              />
              <div className="mt-1 flex items-center gap-2">
                <Button
                  size="sm"
                  className="h-7 px-2 text-xs"
                  disabled={applying || Math.round(draftWidth) === selectedText.width}
                  onClick={() => void applyWidth()}
                >
                  {applying ? 'Gravando…' : 'Aplicar'}
                </Button>
                <button
                  type="button"
                  className="text-[11px] underline opacity-70"
                  onClick={() => setDraftWidth(selectedText.originalWidth)}
                >
                  restaurar
                </button>
              </div>
              <p className="mt-1 text-[10px] leading-3 opacity-60">
                original {selectedText.originalWidth}px · atual {selectedText.width}px · teto 2×
                (ou +64 px) e borda direita da tela
              </p>
            </section>
          ) : (
            <p className="text-muted-foreground text-[11px]">
              Clique numa caixa de texto (tracejada) para ajustar a largura.
            </p>
          )}

          <section>
            <h3 className="mb-1 font-medium">Texturas ausentes</h3>
            {missing.length === 0 ? (
              <p className="text-muted-foreground">Todas as texturas resolveram.</p>
            ) : (
              <ul className="space-y-0.5">
                {missing.map((name) => (
                  <li key={name} className="text-amber-500">
                    {name}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="min-h-0">
            <div className="mb-1 flex items-center justify-between">
              <h3 className="font-medium">Blocos</h3>
              {hiddenBlocks.size > 0 ? (
                <button
                  type="button"
                  className="text-primary text-[11px] hover:underline"
                  onClick={() => setHiddenBlocks(new Set())}
                >
                  Mostrar tudo
                </button>
              ) : (
                <span className="text-muted-foreground text-[11px]">checkbox oculta</span>
              )}
            </div>
            <div className="flex flex-col gap-0.5">
              {blocks.map((block, index) => {
                const active = selectedBlock === index;
                const visible = !hiddenBlocks.has(index);
                return (
                  <div key={`${index}:${block.index}`} className="flex items-start gap-1">
                    <input
                      type="checkbox"
                      className="mt-1 shrink-0"
                      checked={visible}
                      aria-label={`Mostrar camada ${index} (${ykdTypeLabel(block.type)} ${block.index})`}
                      title="Mostrar/ocultar esta camada no preview"
                      onChange={() =>
                        setHiddenBlocks((current) => {
                          const next = new Set(current);
                          if (next.has(index)) next.delete(index);
                          else next.add(index);
                          return next;
                        })
                      }
                    />
                    <button
                      type="button"
                      onClick={() => setSelectedBlock((current) => (current === index ? null : index))}
                      className={`min-w-0 flex-1 rounded px-1.5 py-0.5 text-left transition-colors hover:bg-primary/10 ${
                        active ? 'bg-primary/15 text-primary' : ''
                      } ${visible ? '' : 'opacity-40'}`}
                    >
                      <span className="tabular-nums opacity-60">#{index}</span>{' '}
                      <span className="font-medium">{ykdTypeLabel(block.type)}</span>{' '}
                      <span className="opacity-70">idx {block.index}</span>
                      {block.parentIndex !== 0xffffffff ? (
                        <span className="opacity-50"> · pai {block.parentIndex}</span>
                      ) : null}
                      {block.resourceIndexes && block.resourceIndexes.length > 0 ? (
                        <span className="opacity-50"> · rec {block.resourceIndexes.join(',')}</span>
                      ) : null}
                      {block.text ? (
                        <span className="opacity-50">
                          {' '}
                          · texto {block.text.width}×{block.text.height > 0 ? block.text.height : 'auto'} ·{' '}
                          {block.text.fontSize}px
                        </span>
                      ) : null}
                    </button>
                  </div>
                );
              })}
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}

function YkdDrawItem({
  item,
  screen,
  resources,
  zoom,
  usePivot,
  showId,
  wireframe,
  selected,
  hidden,
  onSelect,
}: {
  item: YkdItem;
  screen: YkdScreen;
  resources: Map<number, YkdResource>;
  zoom: number;
  usePivot: boolean;
  showId: boolean;
  wireframe: boolean;
  selected: boolean;
  hidden: boolean;
  onSelect: (blockIndex: number) => void;
}) {
  if (hidden) return null;
  if (item.kind === 'text') {
    return (
      <YkdTextItem
        item={item}
        zoom={zoom}
        selected={selected}
        onSelect={() => onSelect(item.blockIndex)}
      />
    );
  }

  const resource = resources.get(item.resourceIndex);
  const texture = item.texture ? screen.textures?.[item.texture] : undefined;
  const label = item.texture ?? resource?.name ?? `recurso ${item.resourceIndex}`;

  if (!texture) {
    // Placeholder do recurso sem textura: mantém o tamanho nominal para o
    // usuário enxergar onde a peça entraria.
    const width = Math.max(8, (item.viewWidth || 96) * Math.abs(item.scaleX) * zoom);
    const height = Math.max(8, (item.viewHeight || 32) * Math.abs(item.scaleY) * zoom);
    const left = (usePivot ? item.x - item.pivotX : item.x) * zoom;
    const top = (usePivot ? item.y - item.pivotY : item.y) * zoom;
    return (
      <div
        style={{ left, top, width, height }}
        className={`absolute flex cursor-pointer items-center justify-center overflow-hidden rounded-sm border border-dashed border-amber-500/70 bg-amber-500/10 text-[9px] text-amber-400 ${
          selected ? 'ring-1 ring-primary' : ''
        }`}
        title={`${label} (bloco ${item.blockIndex})`}
        onClick={() => onSelect(item.blockIndex)}
      >
        <span className="truncate px-1">{label}</span>
      </div>
    );
  }

  return (
    <YkdSpriteItem
      item={item}
      texture={texture}
      label={label}
      zoom={zoom}
      usePivot={usePivot}
      showId={showId}
      wireframe={wireframe}
      selected={selected}
      onSelect={() => onSelect(item.blockIndex)}
    />
  );
}

function YkdSpriteItem({
  item,
  texture,
  label,
  zoom,
  usePivot,
  showId,
  wireframe,
  selected,
  onSelect,
}: {
  item: YkdItem;
  texture: YkdTexture;
  label: string;
  zoom: number;
  usePivot: boolean;
  showId: boolean;
  wireframe: boolean;
  selected: boolean;
  onSelect: () => void;
}) {
  const viewWidth = item.viewWidth || texture.width;
  const viewHeight = item.viewHeight || texture.height;
  const sourceWidth = item.sourceWidth || texture.width;
  const sourceHeight = item.sourceHeight || texture.height;
  const scaleX = item.scaleX;
  const scaleY = item.scaleY;
  if (viewWidth <= 0 || viewHeight <= 0 || sourceWidth <= 0 || sourceHeight <= 0) return null;
  if (scaleX === 0 || scaleY === 0) return null;

  // Quantos pixels de TELA cada pixel da FONTE ocupa. A imagem inteira é
  // posicionada para que o recorte (sourceX/Y/W/H) caia no box.
  const pixelX = (viewWidth * Math.abs(scaleX)) / sourceWidth;
  const pixelY = (viewHeight * Math.abs(scaleY)) / sourceHeight;

  const flipX = item.flipX || scaleX < 0;
  const flipY = item.flipY || scaleY < 0;
  const width = viewWidth * Math.abs(scaleX) * zoom;
  const height = viewHeight * Math.abs(scaleY) * zoom;
  const left = (usePivot ? item.x - item.pivotX : item.x) * zoom;
  const top = (usePivot ? item.y - item.pivotY : item.y) * zoom;

  if (wireframe) {
    return (
      <div
        style={{ position: 'absolute', left, top, width, height }}
        className={`flex cursor-pointer flex-col justify-center overflow-hidden rounded-sm border border-dashed border-sky-500/60 bg-sky-500/5 px-1 text-[9px] leading-3 text-sky-300 ${
          selected ? 'ring-1 ring-primary' : ''
        }`}
        title={`${label} (bloco ${item.blockIndex}, ${viewWidth}×${viewHeight})`}
        onClick={onSelect}
      >
        <span className="truncate">{label}</span>
        <span className="opacity-70">
          {viewWidth}×{viewHeight}
        </span>
      </div>
    );
  }

  return (
    <div
      style={{
        position: 'absolute',
        left,
        top,
        width,
        height,
        overflow: 'hidden',
        transform: `rotate(${item.rotation}rad) scale(${flipX ? -1 : 1}, ${flipY ? -1 : 1})`,
        transformOrigin: `${item.pivotX * zoom}px ${item.pivotY * zoom}px`,
        outline: selected ? '1px solid var(--primary)' : undefined,
        imageRendering: 'pixelated',
      }}
      title={`${label} (bloco ${item.blockIndex}, recurso ${item.resourceIndex})`}
      onClick={onSelect}
    >
      {/* biome-ignore lint/performance/noImgElement: data URL gerado pelo Go */}
      <img
        src={texture.pngData}
        alt={label}
        draggable={false}
        style={{
          position: 'absolute',
          left: -item.sourceX * pixelX * zoom,
          top: -item.sourceY * pixelY * zoom,
          width: texture.width * pixelX * zoom,
          height: texture.height * pixelY * zoom,
          maxWidth: 'none',
          imageRendering: 'pixelated',
        }}
      />
      {showId ? (
        <span className="pointer-events-none absolute left-0 top-0 bg-black/70 px-1 text-[9px] leading-3 text-white">
          #{item.blockIndex} t{item.blockType}
        </span>
      ) : null}
    </div>
  );
}

function YkdTextItem({
  item,
  zoom,
  selected,
  onSelect,
}: {
  item: YkdItem;
  zoom: number;
  selected: boolean;
  onSelect: () => void;
}) {
  const text = item.text as YkdText | null | undefined;
  if (!text) return null;

  // A posição vem do BLOCO; a cauda só define a CAIXA (largura/altura/fonte).
  // O pivô é o alinhamento da caixa em relação à posição do bloco.
  const width = Math.max(1, text.width) * zoom;
  const rawHeight = text.height > 0 ? text.height : Math.max(text.fontSize, 18);
  const height = Math.max(8, rawHeight * zoom);
  const left = (item.x - item.pivotX) * zoom;
  const top = (item.y - item.pivotY) * zoom;
  const fontSize = Math.max(8, Math.min(52, text.fontSize * zoom));

  return (
    <div
      style={{
        position: 'absolute',
        left,
        top,
        width,
        height,
        fontSize,
        lineHeight: 1.1,
        color: bgraToCss(text.color1) ?? 'rgba(255,255,255,0.9)',
      }}
      className={`cursor-pointer overflow-hidden border border-dashed px-0.5 ${
        selected ? 'border-primary bg-primary/10 ring-1 ring-primary' : 'border-violet-400/70'
      }`}
      title={`Texto (bloco ${item.blockIndex}) — caixa ${text.width}×${
        text.height > 0 ? text.height : 'auto'
      }, fonte ${text.fontSize}px, máx. ${text.maxWidth}px`}
      onClick={onSelect}
    >
      <span className="block truncate">texto</span>
      <span className="block truncate text-[9px] leading-3 opacity-60">
        {text.width}×{text.height > 0 ? text.height : 'auto'} · {text.fontSize}px
      </span>
    </div>
  );
}

function ykdTypeLabel(type: number): string {
  switch (type) {
    case 0:
      return 'sprite';
    case 1:
      return 'grupo';
    case 2:
      return 'tipo 2';
    case 3:
      return 'tipo 3';
    case 4:
      return 'lista';
    case 5:
      return 'texto';
    case 6:
      return 'texto+';
    case 2000:
    case 2001:
      return 'tela';
    default:
      return `tipo ${type}`;
  }
}

/** BGRA (uint32 do jogo) → CSS rgba. */
function bgraToCss(value: number | undefined): string | null {
  if (value === undefined) return null;
  const b = value & 0xff;
  const g = (value >>> 8) & 0xff;
  const r = (value >>> 16) & 0xff;
  const a = (value >>> 24) & 0xff;
  if (a === 0) return null;
  return `rgba(${r},${g},${b},${(a / 255).toFixed(3)})`;
}
