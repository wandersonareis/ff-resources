# Arquitetura (resumo)

Monorepo **ff-resources** (Go + Wails v3, um módulo raiz `ffresources`) com três
apps Wails: `apps/ffx`, `apps/ff13`, `apps/launcher`.

Consulte o plano completo em
`C:\Users\wande\.opencode\plan\ff-resources-ffx-ff13-launcher.md`.

## Camadas (e a regra de dependência)

```
apps/fx, apps/ff13, apps/launcher   (package main, bindings Wails)
        │ importa
        ▼
internal/families/ffx, internal/families/white
        │ importa
        ▼
internal/shared/*  (dto, artifact, hash, family, launcher, logging, progress)
```

- `internal/shared` **nunca** importa `internal/families/*`.
- O launcher importa apenas `internal/shared/launcher`.
- `internal/platform` (menus, janelas, updater) é infraestrutura de app.

## Módulos Go

Um único `go.mod` na raiz (`ffresources`). Cada app é um `package main` em
`apps/<app>` (sem go.mod próprio) — validado: `wails3 build` funciona dentro do
módulo raiz. Fallback, se o Wails um dia exigir módulo por app: `go.work`.

## Status desta primeira etapa (scaffold)

Migrado de `F:\ffxWails\FFX_Resources` (somente leitura), reescrevendo apenas os
**caminhos de import**; os nomes de pacote ficaram como estavam (renome
cosmético fica para depois):

| Origem | Destino |
|---|---|
| `backend/dto` | `internal/shared/dto` |
| `backend/formatters` (exceto hash) | `internal/shared/artifact` |
| `backend/formatters/hash` | `internal/shared/hash` |
| `backend/loggingService` | `internal/shared/logging` |
| `backend/progress` | `internal/shared/progress` |
| `backend/core/*`, `backend/fileFormats/*` | `internal/families/ffx/codecs/*` |
| `backend/builders` | `internal/families/ffx/builders` |
| `backend/services` | `internal/families/ffx/service` |
| `backend/appcore` | `internal/families/ffx/appcore` |
| `backend/common` | `internal/families/ffx/profile/common` |
| `backend/interactions` | `internal/families/ffx/profile/interactions` |
| `backend/models`, `datastore`, `interfaces`, `data` | `internal/families/ffx/profile/*` |
| `backend/uilog`, `appmenu`, `appwindows`, `appupdater` | `internal/platform/*` |
| raiz `package main` | `apps/ffx` |
| `frontend/` | `apps/ffx/frontend` |
| `testData/` | `testdata/ffx` (trazido manualmente) |

`build/` e a UI default de `ff13`/`launcher` são **gerados por wails3 init**
(nada copiado do `build/` do projeto antigo).

## Codecs White (trazidos)

Os formatos Go de `F:\ff13crypt` já foram trazidos para
`internal/families/white/codecs` (ver o `README.md` da pasta):

| Pacote | Origem | Estado |
|---|---|---|
| `filelist` | `filelistgo/backend/core/filelist` | testes verdes (golden WhiteBinTools, rebuild) |
| `whitebin` | `filelistgo/backend/fs/whitebin` | testes verdes (extração real quando o jogo existe) |
| `ztr` | `ztrgo/backend/fs/ztr` | testes verdes (oráculos byte a byte, comprimido/não comprimido) |
| `ztr/encoding` | `ztrgo/backend/core/encoding` | testes verdes (9 pares jogo × codepage) |
| `wpd` | `wpdgo/backend/fs/wpd` | compila; sem testes no original |
| `imgb/core` | `imgbgo/backend/core/imgb` | testes verdes (PS3/GTEX) |
| `imgb/fs` | `imgbgo/backend/fs/imgb` | compila; repack ainda frágil |

Validação atual:

```powershell
go build ./...                                  # verde
go test ./internal/families/white/codecs/... -count=1   # verde
```

O **texto** já está ligado a um serviço mínimo (`apps/ff13/textservice.go`):
lista os `.ztr` de `apps/ff13/bin/data/partN`, decodifica com `ztr/encoding`, lê a tradução
de `apps/ff13/bin/data/mods/partN` e grava edições em `bin/data/mods/` (exceto `system`, bloqueado até o
`ztrlayout`). O `ztrlayout` mission/system, a compressão e o serviço de imagem
ainda não estão integrados.

## Dados FF13 (bin/data/partN + bin/data/mods/partN)

O importador `scripts/ff13import` usa a lista de arquivos extraída do 7z,
casa cada caminho com o filelist por sufixo e:

1. extrai o **original** do white bin para `apps/ff13/bin/data/partN/<caminho interno>`;
2. normaliza a **tradução** do 7z para `apps/ff13/bin/data/mods/partN/<caminho interno>`. 

```powershell
go run ./scripts/ff13import -part part1 ^
  -game "D:\Steam\steamapps\common\FINAL FANTASY XIII" ^
  -data "F:\ff-resources\data\part1" ^
  -mods "F:\ff-resources\mods\part1"
```

Resultado atual (part1, com o esquema de zona):

| Part | Casados | Não casados | Observação |
|---|---:|---:|---|
| part1 | 617 | 102 | 88 índices de zona (cópias byte a byte) + 14 externos (`.exe/.cmd/.png`) |
| part2 | 575 | 3 | ok |
| part3 | 505 | 12 | ok |

O layout de `data/partN` é o **caminho interno** do filelist; o conteúdo de
zona ganha uma pasta por zona (sem o sufixo de idioma `c`/`u`):

```
data/part1/
  filelist/            cópias dos índices (globais + zone/) — o app não
                       precisa da pasta do jogo para listar a árvore
  gui/ txtres/ vfx/    conteúdo GLOBAL (por caminho interno)
  zone/<zona>/...      conteúdo de ZONA (uma pasta por zona)
mods/part1/            espelho com os arquivos traduzidos
```

A tradução é externa: entra só pela linha de comando do importador
(`-7z`/`-translation`). O app nunca lê de fora de `data/`, `mods/` e da
pasta do jogo.

`apps/ff13/bin/data/partN` e `apps/ff13/bin/data/mods/partN` são criados automaticamente pelo serviço e pelo
importador, mesmo quando ainda estão vazios. O app resolve a raiz do
monorepo pelo `go.mod`; para rodar de outro lugar, `FF_RESOURCES_ROOT`,
`FF_RESOURCES_DATA` e `FF_RESOURCES_MODS` sobrescrevem a raiz.

## Núcleo do FF13 (`internal/families/white`)

O app `apps/ff13` ficou como **casca Wails**: `main.go` (embeds/janela) e
`backend/service` + `backend/dialog`. O núcleo do jogo mora em `internal`,
espelhando o FFX:

- **`internal/families/white/codecs`** — codecs de formato (ZTR, WPD, filelist,
  whitebin, imgb).
- **`internal/families/white/imagery`** — conversão DDS→PNG usada pelo serviço
  de imagem.
- **`internal/families/white/profile/config`** — configuração de execução:
  raiz de dados/mods (`bin/data`), raiz de instalação por part, snapshot do
  config local e preferências (part ativa e `enableMods`). Inclui o
  `profile/appconfig` (arquivo local `ff13.config.json`).
- **`internal/families/white/profile/parts`** — particularidades de cada jogo
  (part1/part2/part3).
- **`internal/families/white/profile/common`** — DTOs compartilhados do
  navegador de filelist.
- **`internal/families/white/service`** — orquestração: unidades de texto,
  filelists (descoberta, índice, árvore, zona e extração), imagens (WPD/TRB/
  faces), coleção cacheada de `data/partN`, export e catálogo de tags.
- **`internal/families/white/family`** — contratos de família/perfil (hoje só
  o White implementa).

No app, **`backend/service`** só registra o `FF13Service` (que EMBUTE o núcleo
e promove os métodos de domínio para o binding), mais `Parts`, `LocalConfig`
e os seletores de imagem. **`backend/dialog`** abre os seletores nativos
(pasta/arquivo/salvar), como no FFX.

Os testes ficam junto do pacote que exercitam (no `internal`); os caminhos
para `apps/ff13/bin/data` são relativos ao diretório do pacote.

## Backend do FFX (`apps/ffx/backend`)

O `main.go` fica na raiz do app (embeds, `application.New`, janela, menu e
updater). O resto é dividido por responsabilidade:

- **`backend/service`** — os serviços Wails por domínio: `App` (sessão e ciclo
  de vida), `ConfigService`, `TextService`, `ImageService`, `VbfService`,
  `LogService`, `UpdateService` e `WindowService`, mais o `Services()` que os
  registra. Cada struct vira um módulo de bindings próprio.
- **`backend/dialog`** — os seletores nativos (pasta, arquivo e "Salvar
  como"), compartilhados pelos serviços.

Os CODECS e o núcleo de dados do FFX continuam em `internal/families/ffx`
(codecs, metadata service, profile): o app é a casca Wails que orquestra esse
núcleo. O `profile/metadata` monta as chaves canônicas dos artefatos (event/
objects/help/macro/image) e o `artifactpaths` resolve os caminhos `edits/` —
ambos exclusivos do FFX.

## Serialização compartilhada (DTO + Options)

`internal/shared` é **neutro**: não importa família nenhuma. Os apps/famílias
entregam o `dto.Collection` pronto e um `formatters.Options` por chamada:

- **idiomas** (`SourceLang`, `TargetLang`, `Languages`): cada família fornece
  o seu catálogo; o shared não conhece `us`/`br`/nenhum jogo;
- **identidade** (`Identity`): `Index` (FFX: `Index+Name`, ordem importa) ou
  `Key` (FF13: `TextRow.Key`/`Name`, permite paralelizar);
- **dedup** (`DisableDedup`, default ligado): o colapso de texto repetido em
  `$hash` vale para todos; a preferência do appmenu/CLI/futuro seletor entra
  pelo orquestrador e `DedupEnabled()` devolve `false` quando desligado.
- `CheckWrite`: a família injeta a própria trava de escrita (o FFX passa
  `common.CheckWritablePath`; o White não precisa).

Campos exclusivos de um jogo (`Character`/`Kind`, `Metadata.Files`,
`ImageEntry.AssetPath/DDSData/...`, `Vbf*`) ficam **opcionais no DTO único**;
os serializadores só os copiam quando existem.

## UI do FF13

A UI em `apps/ff13/frontend` é uma **cópia da UI do FFX** (Vite + React puro) com:

- `GAME_VERSIONS` = `part1`/`part2`/`part3` (exibição FFXIII/FFXIII-2/LRFFXIII);
- **kinds por DIRETÓRIO** na árvore (ver seção "Kinds de texto do FF13"
  abaixo): `ac`, `resident`, `zone`, `event` — mais `scenario` no part3;
- `bindings/ffxresources/*` como **adaptador** para os bindings gerados em
  `bindings-ff13` — a UI copiada não conhece o backend FF13 diretamente;
- `FF13Service.ListTextEntries/GetTextEntry/ApplyTextCollection/ConfigSnapshot`
  como fonte de dados.
- `FF13Service.ListFilelistRoots/ListFilelistDir/ListFilelistZone` — navegador de
  filelist: a árvore sai do índice (nada é decodificado), diretórios expandem sob
  demanda e **clicar numa folha de texto abre o editor**, com paridade com a
  seção `/data` (mesma unidade, mesmo rascunho compartilhado, save em
  `mods/partN`). O backend manda só **dados** (o DTO `internal/families/white/profile/common`:
  `FilelistRoot`/`FilelistNode` com `kind`, `source`, `language`, `zone`,
  `mergeChain`, `unitId`, `text` e `container`); **toda a formatação de texto é
  do frontend** (`src/lib/ffx/filelist-common.ts` + `filelist-tree.ts`).
  Filelists embutidos (`filelist_zXXX*.win32.bin`) vêm marcados e apontam para
  white bins externos.
- **Texto no filelist**: folhas `.ztr` soltos e containers `.bin` WPD viram
  entradas de texto (`unitId` = `<dir>/<base>` da unidade). O pristine sai de
  `data/partN`; **se o arquivo não estiver lá, os bytes vêm do white bin**,
  lidos em memória sob demanda (`whitebin.ExtractBytes`) — nada é gravado em
  `data/`. Sem o jogo instalado, o fallback não existe e vale o comportamento
  antigo (erro "unidade de texto não encontrada").
- O código do FF13 é dividido por **partN**, porque cada jogo tem as suas
  particularidades de filelist e de `/data`:
  - backend `internal/families/white/profile/parts/{part1,part2,part3}.go`: pasta de dados
    (`white_data`/`alba_data`/`weiss_data`), diretórios de índice, game code e
    convenção do par filelist ↔ white bin;
  - frontend `src/lib/parts/{part1,part2,part3}.ts`: rótulos i18n, agrupamento
    da árvore de `/data`, a ordem das raízes de filelist e o rótulo do nó de
    zona (`Zone`); o nome de cada índice vem do `filelist-common` (comum aos
    três jogos);
  - a descoberta valida o par e registra **erro** para todo arquivo sem par
    (white bin sem filelist ou índice sem white bin).
- Os caminhos de instalação NÃO são hardcoded: vêm do arquivo local
  `apps/ff13/ff13.config.json` (gitignored, criado com placeholders quando
  falta). O backend lê esse arquivo (`internal/families/white/profile/appconfig`), os testes
  usam o MESMO arquivo e a interface mostra o caminho em
  Configuração → "Instalação do jogo (config local)".
- Os índices **embutidos** (um filelist dentro de um white bin, como
  `zone/filelist_z000u` dentro de `white_imgu`) são descobertos recursivamente:
  o índice é extraído do container (`whitebin`), lido e usado para casar os
  white bins que estavam sem par. No part1 o índice de zona alimenta o nó
  `Zone` (a união); no part2/part3 a UI abre esse índice pelo nó próprio
  (`pai#caminho/virtual`).
- Regras confirmadas pelo código do **Pulse** (Albeoris/Pulse, commit
  `a62bae0`): o par da raiz é `filelist`→`white_img`, `filelist_scr`→`white_scr`,
  `filelist_patch`→`white_patch`; o índice de zona mora no bin global e o
  conteúdo no bin de zona (`white_<zona>_img.win32.bin`); as entradas
  `*.bin2` são o 2º arquivo (índice de 1-2 entradas). Adotamos também a
  regra `IsUnexpectedEntry`: no part1, entradas `bg/locXXX` de outra zona
  dentro de um índice de zona são descartadas (`parts.Part.KeepZoneEntry`).

### Árvore do part1 (como aparece na tela)

```
Filelist                                  ← grupo 1 (texto editável)
├── filelistu (inglês)       ← filelistu.win32.bin      → white_imgu
├── filelistc (japonês)      ← filelistc.win32.bin      → white_imgc
├── filelist_scru (inglês)   ← filelist_scru.win32.bin  → white_scru
├── filelist_scrc (japonês)  ← filelist_scrc.win32.bin  → white_scrc
└── Zone                                  ← as ZONAS (só no part1)
    ├── z000                               ← união do índice de zona (c+u)
    ├── z002
    └── … 23 zonas (z000, z002, … z106)

Data                                    ← grupo 2 (editável)
├── gui/ txtres/ vfx/                  conteúdo global por caminho interno
└── zone/<zona>/txtres/…               conteúdo de zona (uma pasta por zona)
```

Regras do esquema:

- o nome exibido vem do **frontend** (`src/lib/ffx/filelist-common.ts`): o nome
  do índice sem a extensão `.win32.bin`, em minúsculas, com o idioma entre
  parênteses (`filelistu (inglês)`, `filelistc (japonês)`); o tooltip mostra a
  plataforma e a pasta da cópia vencedora (`PC · white_data`);
- a ordem também é do frontend (`PartUi.filelistRootOrder`): globais primeiro,
  inglês antes do japonês; as zonas ficam em ordem numérica;
- no **part1** (`PartUi.zoneAsTopLevel`) as zonas viram o nó de topo `Zone`,
  com um filho por zona: dentro da zona o conteúdo é a **união** do índice de
  zona (c/u) com as entradas `zone/<z>/…` dos demais índices
  (`ListFilelistZone`); no part2/part3 os índices de zona continuam no grupo
  `zone`, um nó por índice;
- diretórios com filho único são **fundidos** (regra do Pulse): o rótulo vira
  a cadeia (`empty/empty`) e o `path` é o diretório final;
- a pasta `zone/` dentro de um índice do part1 fica **oculta** (o conteúdo da
  zona vive só no nó `Zone`);
- `/data` mostra **todo** o conteúdo extraído: os `.ztr` são texto (editável),
  o resto (`.imgb`, `.xgr`, `.xfv`…) aparece como "fora do escopo" (`text=false`
  no sumário → folha sem `entry`, sem checkbox);
- no grupo **Filelist** valem as mesmas regras de texto: `.ztr` soltos e `.bin`
  que são containers WPD abrem a tabela (unidade completa, com os demais
  idiomas como referências); os outros formatos continuam "fora do escopo". O
  rótulo da folha de texto é o **nome do arquivo em minúsculas, com a
  extensão** (`auto_c01_us.bin`, `txtres_us.ztr`);
- a pasta `filelist/` de `data/partN` NÃO aparece em `/data`: os índices já
  estão no grupo "Filelist".

`wails3 build` do app FF13 passa (Vite build + bindings + exe). O binário fica
em `apps/ff13/bin/pulse.exe` (nome de distribuição; o launcher o copia para
`apps/launcher/bin/ff13/pulse.exe`).

## Kinds de texto do FF13 (a régua é o DIRETÓRIO)

O FF13 tem texto em **um formato só** (o `.ztr`). Dividir o escopo por tipo de
**arquivo** não separaria nada — a régua que separa é o **diretório**: o
`txtres` é praticamente o diretório pai de todos os subdiretórios com texto, e
a pasta logo abaixo dele é a **família** do conteúdo.

| kind | Diretório | part1 | part2 | part3 |
|---|---|---|---|---|
| `ac` | `txtres/ac` | só em zona | ✅ | — |
| `resident` | `txtres/resident` | ✅ | ✅ | ✅ |
| `zone` | `txtres/zone` | ✅ | ✅ | ✅ |
| `event` | `txtres/event` | ✅ | ✅ | ✅ |
| `scenario` | `db/ai/npc/pack` | — | — | ✅ |

Duas consequências dessa escolha, ambas verificadas contra os dados reais
(`apps/ff13/bin/data/part{1,2,3}` e os filelists dos três jogos instalados):

1. **O `scenario` é o único kind cujo texto NÃO é um `.ztr` solto**: o arquivo
   é um container `.bin` (WPD, magic `WPD\0`) que carrega o `.ztr` dentro
   (`auto_c00_us.bin` → `auto_c00.wdb` + `auto_c00_us.ztr`). No part3 o
   `db/ai/npc/pack` tem **3329** containers assim — e são os únicos dos três
   jogos (part1: 0 WPD; part2: 1619 WPD, **nenhum** com `.ztr`). Por isso o
   **importador não precisa de caminho especial por kind**: a rota
   `ApplyTextCollection` já reempacota o WPD preservando os demais registros,
   igual para todos os kinds.
2. **O `txtres/ac` do part1 não é global**: vive em
   `zone/zXXX/txtres/ac/…`. A regra do `txtres` pega os dois casos (o `txtres`
   global e o `txtres` de zona), então a mesma função serve para os dois.

**Duplicatas confirmadas por SHA-256 (só `/data`, nunca `/mods` — o `mods` está
traduzido e divergiria por construção):** no part1 há 17 diretórios com nome
repetido (`ev_hiku_495` em z003 e z004, `z003` em z003/z004/z015…) e os 17 têm
o `_us` **byte a byte idêntico**; part2 e part3 têm zero. As cópias são o mesmo
arquivo gravado em dois lugares, então um dedup por hash na exportação não
perde nada.

Onde a regra vive:

- **backend** `internal/families/white/profile/parts/kinds.go` — constantes dos kinds e
  `textKindByPath`/`scenarioKindByPath`; `Part.TextKind` (a interface
  `parts.Part` ganhou o método) classifica cada unidade e `Part.Kinds`
  (`parts.Kinds`) lista os que a part serve;
- **coleção** `DataEntry.Kind` + `white.TextUnitSummary.Kind` — o kind viaja
  até a UI junto do `Type` (que continua sendo a categoria de **conteúdo**, o
  ícone da linha; `kind` é o escopo — os dois eixos são independentes);
- **export** `internal/families/white/service/export.go` — `ExportArtifactBatch` filtra por kind e o
  artefato passa a se chamar `<kind>_all_localizations_<part>.<ext>`
  (`resident_all_localizations_part1.json`, `event_all_…`), então duas
  famílias nunca colidem no mesmo arquivo. O kind legado `ztr` continua
  aceito e significa "todos os kinds da part";
- **frontend** `src/lib/ffx/kinds.ts` (lista por part), `src/lib/ffx/text-kind.ts`
  (o mesmo mapeamento do backend, para a folha do filelist abrir a unidade com
  o kind certo antes de qualquer listagem) e `buildUnitTree`
  (`src/lib/parts/unit-tree.ts`), que monta **uma raiz por kind**
  (`kind:resident` → "Residentes (N)") e reagrupa por diretório dentro dela.

## UI em React puro (FFX)

O app FFX deste repositório foi migrado de Next.js para **Vite + React**:

- `apps/ffx/frontend/vite.config.ts` — React, Tailwind v4 (`@tailwindcss/vite`),
  `tsconfigPaths`, plugin Wails e um resolvedor de `.js` → `.ts` dos bindings;
- `index.html` + `w/main.tsx` (multi-page) e `src/main.tsx`/`src/App.tsx` —
  roteamento mínimo: `/w/*` = janelas filhas, resto = janela principal;
- `src/components/window/window-host.tsx` substitui a rota `/w` do Next;
- o tema vem do pacote compartilhado via alias de `next-themes` (ver abaixo);
- os bindings foram regenerados como **classes** (`wails3 generate bindings -ts`,
  sem `-i`) para manter a API `dto.TextRow.createFrom` usada pela UI antiga;
- os imports da UI foram reescritos de `bindings/ffxresources/...` para
  `bindings/ffresources/...` (o módulo real do monorepo).

O `wails3 build` do app FFX passa e o binário fica em `apps/ffx/bin/spira.exe`
(nome de distribuição; o launcher o copia para `apps/launcher/bin/ffx/spira.exe`).
Para abrir com a MESMA árvore do projeto antigo, o `apps/ffx/bin/config/`
recebeu o `config.json` antigo e `apps/ffx/bin/data` é um **junction** para
`F:\ffxWails\FFX_Resources\build\bin\data` (o projeto antigo não é alterado).

## Pacote compartilhado (`frontend/shared`)

Pacote local `@ff-resources/ui`, Wails-agnostic:

```
frontend/shared/
  package.json        @ff-resources/ui (exports . / ./theme / ./api/GameApi)
  vite-aliases.mjs   mapa de aliases comum (Vite + Vitest)
  src/index.ts        reexports
  src/theme.tsx       ThemeProvider/useTheme (API compatível com next-themes)
  src/api/GameApi.ts  contrato de UI + tipos de TagCatalog
  src/api/ApiProvider.tsx  ApiProvider/useApi (injeção do adapter)
  src/components/ui/  primitivas shadcn
  src/editor/         editor de texto + helpers (parser, tags, catálogo...)
  src/lib/utils.ts    cn
```

- **FFX e FF13** são Vite + React puro (react@latest = 19.3.1) e consomem o
  pacote pelo mesmo mapa de aliases (`frontend/shared/vite-aliases.mjs`).
- A árvore/tabela (`tree-data`) e o estado do editor usam o `GameApi` via `useApi()`/`getCurrentApi()` (registry): o pacote não importa bindings.
- O catálogo de tags deixou de importar bindings: o editor compartilhado pede
  o catálogo ao `GameApi` injetado (`ApiProvider`), e cada app monta o seu
  adapter em `src/api.ts` (FFX sobre os bindings Wails, FF13 sobre o adaptor
  local que fala com o `FF13Service`).
- As cópias locais de `components/ui`, `components/editor` e dos helpers de
  texto saíram dos dois apps — não há mais duplicação dessas peças.
- A **árvore é data-driven**: o shared define o contrato `SideNode`
  (`app/content-tree/model.ts`) e o componente de renderização; cada app monta
  os nós como fizer sentido — FFX agrupa por evento, FF13 usa a árvore de
  UNIDADES (`buildUnitTree`, em `apps/ff13/frontend/src/lib/parts/unit-tree.ts`:
  uma pasta com UMA unidade vira o nó clicável). O renderizador só desenha o que
  receber.
- **Backend manda dados + `type`** (system, mission, event, zone, book,
  review, text, folder, file) e **não traduz**. O frontend decide rótulo/ícone
  via `typeLabels`/`typeIcons` no `ContentTreeProvider`; para controle total, o
  chamador pode passar `renderItem(node, depth, defaultContent)`. Isso deixa o
  backend em inglês e a i18n no frontend.
- A seleção de **kinds é por app** (`src/lib/ffx/kinds.ts`): FFX tem
  events/objects/macro/...; FF13 tem `ztr`.
- O CSS de cada app declara `@source "../../../../../frontend/shared/src";`
  para o Tailwind v4 varrer as classes dos componentes compartilhados.
- **Fontes são por app**: FFX usa Malva + Geist Mono; FF13 usa NewRodin Pro DB
  (fonte do jogo) no corpo e Fira Code no slot mono (`src/app/fonts.css` +
  `public/fonts/` em cada app).
- **Ícone do FF13**: `apps/ff13/build/appicon.png` + ícones gerados por
  `wails3 generate icons` (rodando em `apps/ff13/build`).
- **Otimizações do Vite/React** (nos dois apps): React Compiler via
  `babel-plugin-react-compiler`, `resolve.dedupe` das deps do pacote,
  `optimizeDeps.include`, `server.fs.allow` para o pacote compartilhado,
  `build.target es2022` e chunks separados (`react`, `tiptap`) para o app shell
  recarregar menos.

## Pendente (próximos passos)

- **Contrato DTO/formato do plano**: `Key`/`Layout`/`Files` no DTO,
  `artifact.Options` com `SourceLang`/`TargetLang`/`Identity`, e o `Matcher`
  por família. Hoje o `internal/shared/artifact` ainda é a versão do FFX e
  importa `internal/families/ffx/profile/common` (dívida de acoplamento).
- **Testes Go do FFX**: dependem de `testdata/ffx` (a trazer) e do fixture de
  dados de jogo (`build/bin/data`, não copiado). Até lá, `go test ./...` não
  roda limpo; `go build ./...` passa.
- **Mover os componentes restantes para `frontend/shared`**: o pacote já tem
  tema + `GameApi` + `ApiProvider` + primitivas `components/ui` + a camada do
  editor (`src/editor`). Falta a árvore/tabela/imagem (`entry-view`, `dialogs`)
  e as libs por app (`display-names`, `event-group-names`, `shortcuts`), que
  devem passar a depender do `GameApi` em vez dos bindings.
- **`ztrlayout` mission/system**: o serviço de texto já lê/escreve ZTR
  padrão, mas o save em `system` está **bloqueado** para não quebrar a
  ordenação de itens; o writer precisa preservar o bloco de chaves.
- **Compressão**: a política `none`/`required`/`auto` (`auto_yus_us` e 256 KB)
  ainda não foi ligada ao fluxo de exportação/instalador.
- **Part1 `zone/FF13Z`**: 413 textos de zona + filelists aninhados não casaram
  com os pares principais; falta abrir os filelists de zona e extrair os
  originais correspondentes.
- **Launcher**: ainda é o template do `wails3 init`; a UI real (capas +
  `--profile`) é a fase seguinte.
