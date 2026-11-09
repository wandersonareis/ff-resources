# @ff-resources/ui

Pacote local, **Wails-agnostic**, com os componentes/contratos React + TS
compartilhados entre os apps `ffx` e `ff13`.

Regra de ouro: este pacote **não importa** `bindings/*` gerados pelo Wails.
Cada app injeta um adapter (`GameApi`) e o pacote só conhece a interface.

## O que já vive aqui

| Módulo | Conteúdo |
|---|---|
| `src/components/ui/*` | primitivas shadcn (alert, badge, button, checkbox, collapsible, context-menu, dialog, dropdown-menu, input, label, popover, progress, resizable, scroll-area, separator, sheet, sonner, table, tabs, tooltip) |
| `src/editor/*` | editor de texto do jogo + helpers: `game-text-editor`, extensões do Tiptap, `game-text-parser`, `game-text-format`, `game-text-tags`, `tag-catalog`, `plain-text`, `protected-tags`, `find-in-text`, `leading-padding`, `blank-us`, `bytes`, `clipboard` |
| `src/app/*` | stores e helpers de app (agnósticos de jogo): `statusbar-store`, `progress-store`, `find-store`, `shortcuts`, `layout-storage`, `tab-storage`, `use-persisted-layout`, `use-wails-event`, `error-handler`, `hash-ref`, `display-names`, `tree-data`, `export-selection`, `edit-draft`, `save-all`, `prune-confirm-store`, `toast-logged` |
| `src/components/game-text-view.tsx` | renderizador de texto do jogo (chips de tag) |
| `src/components/progress/progress-modal.tsx` | modal de progresso |
| `src/components/entry-view/**` | `entry-table-empty`, `content-tree/search-control`, `image-panel/{use-image-preview,image-preview}` |
| `src/dto.ts` | tipos canônicos do DTO de texto (espelham `internal/shared/dto` no Go) + fábricas `textRow`/`fileEntry`/`metadata` |
| `src/lib/utils.ts` | `cn` |
| `src/theme.tsx` | `ThemeProvider`/`useTheme` (API compatível com `next-themes`) |
| `src/api/GameApi.ts` | contrato de UI que cada app implementa (inclui os tipos de `TagCatalog`) |
| `src/api/ApiProvider.tsx` | `ApiProvider` + `useApi()` — injeção do adapter |
| `src/api/registry.ts` | `setCurrentApi`/`getCurrentApi` para os módulos fora do React (stores/loaders) |
| `src/api/appTypes.ts` | `PrunedRowView`, `TextSearchMatch/Result/Response` |
| `src/app/content-tree/model.ts` | `SideNode` (contrato da árvore) + `entryNodeId`/`highlightedNodeId` |
| `src/components/entry-view/content-tree/*` | `TreeList` (recebe `TreeGroup[]` e só renderiza), `TreeItem`, `tree-context` |
| `src/components/entry-view/entry-table/*` | `table-types`, `entry-table-header`, `entry-table-body`, `table-bar` e as colunas puras (`editable`, `index`, `linked`, `name`, `original`) |
| `src/app/vbf-selection.ts` | seleção de exportação de nós .vbf (FFX) |
| `vite-aliases.mjs` | mapa de aliases compartilhado entre Vite e Vitest |

## Como os apps consomem

FFX e FF13 são **Vite + React puro** (react@latest = 19.3.1); os dois usam o
mesmo mapa de aliases (`vite-aliases.mjs`):

```ts
import { sharedAliases } from '../../../frontend/shared/vite-aliases.mjs';

resolve: {
  dedupe: ['react', 'react-dom', 'cn', 'radix-ui', '@tiptap/core', ...],
  alias: {
    ...sharedAliases(here),
    '@/bindings': path.resolve(here, 'bindings'), // do app
  },
}
```

Assim os componentes continuam importando `@/components/ui/...`,
`@/components/editor/...` e `@/lib/ffx/<helper>` — mas a implementação vem
do pacote, sem duplicação.

O adapter de cada app fica em `src/api.ts` e é injetado no `main.tsx`:

```tsx
<Providers>
  <ApiProvider api={ffxApi /* ou ff13Api */}>
    <App />
  </ApiProvider>
</Providers>
```

> **Tailwind:** o pacote fica fora da raiz do app, então o CSS de cada app
> declara `@source "../../../../../frontend/shared/src";` para o scanner do
> Tailwind v4 ver as classes dos componentes compartilhados.

> **Fontes são por app:** o FFX usa Malva + Geist Mono (`apps/ffx/frontend/src/app/fonts.css`);
> o FF13 usa NewRodin Pro DB (fonte do jogo) + Fira Code (`apps/ff13/frontend/src/app/fonts.css`).
> Os arquivos ficam em `public/fonts/` de cada app.

## Árvore (data-driven)

- `SideNode` carrega `type` (não traduzido): `folder`, `file`, `system`,
  `mission`, `event`, `zone`, `book`, `review`, `ac`, `text`…
- O backend manda só os dados + `type`; o frontend monta os nós
  (árvore de unidades no FF13; grupos de evento no FFX).
- Render: `TreeList` recebe `TreeGroup[]`; o app injeta `typeLabels`/
  `typeIcons` no `ContentTreeProvider` (i18n) e, se quiser controle total,
  um `renderItem(node, depth, defaultContent)`.
- Nenhum texto de exibição vem do backend: o backend permanece em inglês.

## Próximos passos

- A tabela já está no pacote (tipos, header/body e colunas puras). Falta levar
  `entry-view-store`, `translated-column` (ainda usam o store do app) e o
  `image-panel`/`dialogs`.
- Manter por app só o que é específico de jogo: `vbf.ts` (FFX), `config-store`,
  `search-store`, `event-group-names` e os adapters de `src/api.ts`.
