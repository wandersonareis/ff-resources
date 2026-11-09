# ff-resources

Suíte de editores de recursos de FINAL FANTASY em **Go + Wails v3**, migrada do
projeto `F:\ffxWails\FFX_Resources` (usado apenas como referência — **não foi
alterado**). Um módulo Go raiz (`ffresources`), três apps Wails e código
compartilhado em `internal/`.

Plano completo: `C:\Users\wande\.opencode\plan\ff-resources-ffx-ff13-launcher.md`.
Decisões de migração e status: `docs/architecture.md`.

## Apps

| App | Pasta | Estado |
|---|---|---|
| **ffx** | `apps/ffx` | backend + UI migrados; frontend **Vite + React puro**; `wails3 build` verde |
| **ff13** | `apps/ff13` | UI copiada do FFX em **Vite + React puro** + codecs White + texto `data/partN`/`mods/partN`; **compilado** |
| **launcher** | `apps/launcher` | `wails3 init` (UI inicial) + `internal/shared/launcher`; **compilado** |

## Estrutura

```
internal/shared/      dto, artifact, hash, family, launcher, logging, progress   (não conhece jogos)
internal/families/ffx  codecs, builders, service, appcore, profile               (FFX/X-2)
internal/families/white family, titles/ff13 (part1/2/3), codecs (filelist/ZTR/IMGB/WPD)
internal/platform/    menus, janelas, updater, uilog
apps/ffx | apps/ff13 | apps/launcher   (package main + bindings Wails)
apps/ffx/frontend/    UI do FFX em Vite + React puro (out/ gerado pelo Vite)
apps/ff13/frontend/   UI copiada do FFX (Vite + React puro) + adaptador ffxresources
scripts/ff13import/   originais -> bin/data/partN; 7z -> bin/data/mods/partN
apps/ff13/bin/data/partN       originais extraídos do filelist/white bin
apps/ff13/bin/data/mods/partN  traduções do 7z normalizadas para o caminho interno
frontend/shared/      pacote local @ff-resources/ui: tema, GameApi, primitivas UI e editor
assets/               capas do launcher, ícones (a fornecer)
testdata/ffx/         fixtures Go do FFX (a trazer manualmente)
```

## Build

```powershell
# Gate rápido (compila tudo, sem UI):
go build ./...

# App FFX (UI Vite + React puro):
wails3 build -dir apps/ffx

# App FF13 (UI copiada do FFX + backend White):
wails3 build -dir apps/ff13
wails3 build -dir apps/launcher

# Importar dados FF13 (7z -> mods/partN; filelist -> data/partN):
go run ./scripts/ff13import -part part1 ^
  -game "D:\Steam\steamapps\common\FINAL FANTASY XIII" ^
  -data "F:\ff-resources\apps\ff13\bin\data\part1" ^
  -mods "F:\ff-resources\apps\ff13\bin\data\mods\part1"

# App FFX (backend compila; UI real é a fase seguinte):
go build ./apps/ffx
```

## Notas

- `build/` e a UI default de `ff13`/`launcher` são **gerados por wails3 init**;
  nada foi copiado do `build/` do projeto antigo.
- A UI do FFX foi migrada de Next.js para **Vite + React puro**
  (`apps/ffx/frontend`), com o pacote local `frontend/shared`
  (`@ff-resources/ui`) para tema/GameApi. Os bindings do FFX são gerados como
  classes (`-ts`, sem `-i`) para manter a API `createFrom` da UI antiga.
- A UI do FF13 é a UI do FFX copiada (Next.js), com adaptador em
  `apps/ff13/frontend/bindings/ffxresources` e bindings gerados em
  `bindings-ff13`. O texto lê `data/partN` e `mods/partN`.
- O save em arquivos `system` está bloqueado até o `ztrlayout` preservar o
  bloco de chaves (ordenação de itens do menu).
- Testes Go dependem de `testdata/ffx` e do fixture de dados de jogo; até serem
  adicionados, `go test ./...` não roda limpo (compilação via `go build` já passa).

## Licenca

Este projeto e distribuido sob a GNU General Public License v3.0, sem clausula de versao posterior (GPL-3.0-only). Veja o arquivo [LICENSE](LICENSE).

