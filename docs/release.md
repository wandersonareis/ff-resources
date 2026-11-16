# Release e auto-update (ff-resources)

O monorepo publica **um release por lote** no repositório
`wandersonareis/ff-resources` (público). Cada app se atualiza sozinho; um
release pode conter só os apps que mudaram.

## Duas versões

| Arquivo | O que é | Usado por |
|---|---|---|
| `VERSION` (raiz) | versão do **release train** (semver, monotônica) | tag do GitHub e comparação do updater |
| `apps/<app>/VERSION` | versão **do app** (semver própria) | exibição (`AppVersion`) |

O build injeta as duas:

```
-ldflags "-X main.currentVersion=<VERSION> -X main.appVersion=<apps/app/VERSION>"
```

- O updater compara a **tag** com `currentVersion` (train).
- `CurrentVersion()` devolve o train; `AppVersion()` devolve a versão do app.
- Um app que não entra no release mantém o train/versão anteriores; quando
  volta a entrar, pula direto para o train atual (ex.: 1.0.0 → 2.7.0 no
  train, 1.0.0 → 1.1.0 na versão do app).

## Artefatos de um release

Um zip por app incluído + a suite + os hashes:

```
spira-<train>-portable-windows-amd64.zip     # FFX (spira.exe)
pulse-<train>-portable-windows-amd64.zip     # FF13 (pulse.exe)
launcher-<train>-portable-windows-amd64.zip  # launcher (quando tiver updater)
ff-resources-suite.zip                       # launcher + apps + manifest + assets
SHA256SUMS                                   # hash de cada zip publicado
```

- O updater de cada app só aceita o próprio prefixo (`spira-`, `pulse-`,
  `launcher-`). A suite não tem prefixo e é ignorada pelo updater.
- Release sem o asset do app = “sem atualização” para aquele app.
- Link fixo para quem ainda não tem nada:
  `https://github.com/wandersonareis/ff-resources/releases/latest/download/ff-resources-suite.zip`

## Fluxo

1. Defina o lote (apps que mudaram).
2. Suba `VERSION` (train) para um semver **maior** que a última tag.
3. Suba `apps/<app>/VERSION` dos apps do lote.
4. Rode a task:

```powershell
task release:ff13      # só FF13
task release:ffx       # só FFX
task release:all       # FFX + FF13
task release:launcher  # launcher
task release:suite     # só a suite (sem zips de app)
task release:dry       # monta zips/hashes do FF13 sem publicar
```

A script (`scripts/release.ps1`) builda os apps do lote com as versões,
zipa cada um, monta a suite, gera `SHA256SUMS` e publica com
`gh release create v<train>`. Use `-DryRun` para não publicar,
`-Draft` para publicar como rascunho (não aparece no `/releases/latest` até
publicar) e `-SkipBuild` para reutilizar binários.

## Dependência entre apps

| Mudou | Build/release |
|---|---|
| `apps/ffx` / `internal/families/ffx` | só `release:ffx` |
| `apps/ff13` / `internal/families/white` | só `release:ff13` |
| `internal/shared` / `internal/platform` | os apps que precisam herdar a mudança |
| `frontend/shared` | os apps que usam a UI compartilhada |
| launcher/manifest/assets | launcher/bundle (fora do updater até ter UI) |

Cada binário é autocontido: código compartilhado compilado no `spira.exe` não
chega ao `pulse.exe` sem rebuild. Não é obrigatório rebuildar tudo para um
release; só os apps que devem receber a mudança.

## Regras

- Tag/versão do train **monotônica**; pode pular (2.5.0 → 2.7.0), nunca
  diminuir.
- O train é global; os apps podem ficar em trains diferentes até serem
  incluídos num release.
- A suite precisa dos binários dos dois jogos e do launcher. Se algum não
  existir, a script builda (com o train atual) só para compor a suite; o zip
  de app fora do lote não é publicado.
- Pré-release: publique com `-Prerelease` para testar; o `/latest` do GitHub
  ignora prereleases.
- Rollback: publique uma versão **maior** com os binários anteriores; não
  rebaixe a tag.
- O repositório precisa ser **público** para o provider GitHub baixar releases
  sem token. (Foi tornado público.)
