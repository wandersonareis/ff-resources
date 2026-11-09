<#
  migrate-from-ffx.ps1

  Porta o backend Go do projeto existente FFX_Resources para a nova estrutura
  do monorepo ff-resources, SEM alterar o projeto original (somente leitura).

  Regras:
   - Nada de build/ e nem frontend gerado (out/, dist/, .next/, node_modules/):
     build/ e UI default são gerados por wails3 init, por projeto.
   - testData NAO é copiado aqui (é trazido manualmente para testdata/ffx). As
     importacoes de teste ja sao reescritas para ffresources/testdata/ffx.
   - Somente os CAMINHOS de import sao reescritos; os nomes de pacote ficam
     como estao (risco minimo). Renomeios cosméticos ficam para depois.
#>
[CmdletBinding()]
param(
  [string]$Src = 'F:\ffxWails\FFX_Resources',
  [string]$Dst = 'F:\ff-resources'
)

$ErrorActionPreference = 'Stop'

function Copy-Tree($from, $to, $excludeDirs = @()) {
  $srcPath = Join-Path $Src $from
  if (-not (Test-Path $srcPath)) { Write-Warning "origem ausente: $from"; return }
  $dstPath = Join-Path $Dst $to
  New-Item -ItemType Directory -Force -Path $dstPath | Out-Null
  $args = @($srcPath, $dstPath, '/E', '/NFL', '/NDL', '/NJH', '/NJS', '/NP')
  if ($excludeDirs.Count) { $args += '/XD'; $args += ($excludeDirs | ForEach-Object { Join-Path $srcPath $_ }) }
  & robocopy @args | Out-Null
  # robocopy exit codes 0..7 are success/normal
}

# 1) whole-subtree package moves  (src\rel -> dst\rel)
$moves = @(
  @{ s = 'backend\dto';                 d = 'internal\shared\dto' }
  @{ s = 'backend\loggingService';      d = 'internal\shared\logging' }
  @{ s = 'backend\progress';            d = 'internal\shared\progress' }
  @{ s = 'backend\builders';            d = 'internal\families\ffx\builders' }
  @{ s = 'backend\services';             d = 'internal\families\ffx\service' }
  @{ s = 'backend\appcore';             d = 'internal\families\ffx\appcore' }
  @{ s = 'backend\common';              d = 'internal\families\ffx\profile\common' }
  @{ s = 'backend\interactions';        d = 'internal\families\ffx\profile\interactions' }
  @{ s = 'backend\models';              d = 'internal\families\ffx\profile\models' }
  @{ s = 'backend\datastore';           d = 'internal\families\ffx\profile\datastore' }
  @{ s = 'backend\interfaces';          d = 'internal\families\ffx\profile\interfaces' }
  @{ s = 'backend\data';                d = 'internal\families\ffx\profile\data' }
  @{ s = 'backend\uilog';               d = 'internal\platform\uilog' }
  @{ s = 'backend\appmenu';             d = 'internal\platform\appmenu' }
  @{ s = 'backend\appwindows';          d = 'internal\platform\appwindows' }
  @{ s = 'backend\appupdater';          d = 'internal\platform\appupdater' }
)

Write-Host '== copy package subtrees =='
foreach ($m in $moves) { Copy-Tree $m.s $m.d; Write-Host ("  {0,-40} -> {1}" -f $m.s, $m.d) }

# 2) core + fileFormats merge into codecs
Write-Host '== merge core + fileFormats -> codecs =='
Copy-Tree 'backend\core' 'internal\families\ffx\codecs'
Copy-Tree 'backend\fileFormats' 'internal\families\ffx\codecs'

# 3) formatters -> artifact, but formatters/hash -> shared/hash
Write-Host '== formatters -> artifact (hash separated) =='
Copy-Tree 'backend\formatters\hash' 'internal\shared\hash'
Copy-Tree 'backend\formatters' 'internal\shared\artifact'
$hashInArtifact = Join-Path $Dst 'internal\shared\artifact\hash'
if (Test-Path $hashInArtifact) { Remove-Item -Recurse -Force $hashInArtifact }

# 4) apps/ffx package-main files
Write-Host '== apps/ffx (package main) =='
$mainFiles = 'main.go','app.go','service_config.go','service_image.go','service_log.go','service_text.go','service_update.go','service_vbf.go','service_windows.go'
New-Item -ItemType Directory -Force -Path (Join-Path $Dst 'apps\ffx') | Out-Null
foreach ($f in $mainFiles) {
  $src = Join-Path $Src $f
  if (Test-Path $src) { Copy-Item $src (Join-Path $Dst "apps\ffx\$f") -Force }
  else { Write-Warning "main file ausente: $f" }
}

# 5) apps/ffx frontend source (exclude generated/installed)
Write-Host '== apps/ffx/frontend (source only) =='
Copy-Tree 'frontend' 'apps\ffx\frontend' -excludeDirs @('node_modules', '.next', 'out', 'dist', 'coverage')

# 6) scripts + VERSION + docs logo
Write-Host '== scripts / VERSION / logo =='
Copy-Tree 'scripts' 'scripts'
if (Test-Path (Join-Path $Src 'VERSION')) { Copy-Item (Join-Path $Src 'VERSION') (Join-Path $Dst 'VERSION') -Force }
New-Item -ItemType Directory -Force -Path (Join-Path $Dst 'apps\ffx\assets') | Out-Null
$logo = Join-Path $Src 'docs\imagens\logo_us.png'
if (Test-Path $logo) { Copy-Item $logo (Join-Path $Dst 'apps\ffx\assets\logo_us.png') -Force }

# 7) rewrite import paths across every migrated .go (quoted, anchored, ordered)
Write-Host '== rewriting imports =='
$rules = [ordered]@{
  '"ffxresources/backend/formatters/hash' = '"ffresources/internal/shared/hash'
  '"ffxresources/backend/formatters'      = '"ffresources/internal/shared/artifact'
  '"ffxresources/backend/dto'             = '"ffresources/internal/shared/dto'
  '"ffxresources/backend/loggingService'  = '"ffresources/internal/shared/logging'
  '"ffxresources/backend/progress'        = '"ffresources/internal/shared/progress'
  '"ffxresources/backend/core/'           = '"ffresources/internal/families/ffx/codecs/'
  '"ffxresources/backend/fileFormats/'    = '"ffresources/internal/families/ffx/codecs/'
  '"ffxresources/backend/builders'        = '"ffresources/internal/families/ffx/builders'
  '"ffxresources/backend/services'        = '"ffresources/internal/families/ffx/service'
  '"ffxresources/backend/appcore'         = '"ffresources/internal/families/ffx/appcore'
  '"ffxresources/backend/datastore'       = '"ffresources/internal/families/ffx/profile/datastore'
  '"ffxresources/backend/interactions'    = '"ffresources/internal/families/ffx/profile/interactions'
  '"ffxresources/backend/models'          = '"ffresources/internal/families/ffx/profile/models'
  '"ffxresources/backend/interfaces'      = '"ffresources/internal/families/ffx/profile/interfaces'
  '"ffxresources/backend/uilog'           = '"ffresources/internal/platform/uilog'
  '"ffxresources/backend/appmenu'         = '"ffresources/internal/platform/appmenu'
  '"ffxresources/backend/appwindows'      = '"ffresources/internal/platform/appwindows'
  '"ffxresources/backend/appupdater'      = '"ffresources/internal/platform/appupdater'
  '"ffxresources/backend/common'          = '"ffresources/internal/families/ffx/profile/common'
  '"ffxresources/backend/data'            = '"ffresources/internal/families/ffx/profile/data'
  '"ffxresources/testData'                = '"ffresources/testdata/ffx'
  '"ffxresources/backend'                 = '"ffresources/internal/families/ffx/profile'
}

$goFiles = Get-ChildItem -Path (Join-Path $Dst 'internal'),(Join-Path $Dst 'apps\ffx') -Recurse -Filter *.go -File
$changed = 0
foreach ($gf in $goFiles) {
  $text = Get-Content -Raw -LiteralPath $gf.FullName
  if ($text -notmatch 'ffxresources/') { continue }
  $orig = $text
  foreach ($k in $rules.Keys) { $text = $text.Replace($k, $rules[$k]) }
  if ($text -ne $orig) { Set-Content -LiteralPath $gf.FullName -Value $text -NoNewline; $changed++ }
}
Write-Host ("  arquivos .go reescritos: {0}" -f $changed)

# 8) fix main.go embed path for the logo (docs -> assets)
$main = Join-Path $Dst 'apps\ffx\main.go'
if (Test-Path $main) {
  (Get-Content -Raw -LiteralPath $main).Replace('docs/imagens/logo_us.png','assets/logo_us.png') |
    Set-Content -LiteralPath $main -NoNewline
}

# 9) verify no unmapped ffxresources paths remain
Write-Host '== residuos ffxresources/ (devem listar apenas referencias a testData se houver) =='
$residual = Get-ChildItem -Path (Join-Path $Dst 'internal'),(Join-Path $Dst 'apps\ffx') -Recurse -Filter *.go -File |
  ForEach-Object { Select-String -Path $_.FullName -Pattern 'ffxresources/' -AllMatches | ForEach-Object { $_.Matches } } |
  ForEach-Object { $_.Value.Trim('"','/') } | Sort-Object -Unique
if ($residual) { $residual | ForEach-Object { Write-Host "  $_" } } else { Write-Host '  nenhum' }

Write-Host '== DONE =='
