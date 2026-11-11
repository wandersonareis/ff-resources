#Requires -Version 5.1
<#
.SYNOPSIS
  Monta dist/ no layout de distribuicao do suite.

.DESCRIPTION
  Layout final (plano, secao 11.2):

    dist/
    |-- launcher.exe
    |-- manifest.json
    |-- assets/
    |   `-- launcher/*.png
    |-- ffx/
    |   `-- spira.exe
    `-- ff13/
        `-- pulse.exe

  A origem e apps/launcher/bin/, ja populado por stage:launcher durante
  'task build'. So remove dist/ quando ele e uma saida de bundle anterior.
#>
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$root = (Resolve-Path "$PSScriptRoot/..").Path
$src = Join-Path $root 'apps/launcher/bin'
$dist = Join-Path $root 'dist'

$launcherExe = Join-Path $src 'launcher.exe'
if (-not (Test-Path -LiteralPath $launcherExe)) {
    throw ("launcher.exe nao encontrado em {0}`nRode 'task build:launcher' antes de 'task bundle'." -f $src)
}

if (Test-Path -LiteralPath $dist) {
    $looksLikeBundle =
        (Test-Path -LiteralPath (Join-Path $dist 'launcher.exe')) -or
        (Test-Path -LiteralPath (Join-Path $dist 'ffx')) -or
        (Test-Path -LiteralPath (Join-Path $dist 'ff13')) -or
        (-not (Get-ChildItem -LiteralPath $dist -Force))
    if (-not $looksLikeBundle) {
        throw ("dist/ existe e nao parece saida de bundle: {0}`nRemova manualmente se tiver certeza." -f $dist)
    }
    Remove-Item -LiteralPath $dist -Recurse -Force
}

New-Item -ItemType Directory -Force -Path $dist | Out-Null

$copied = @()
foreach ($item in 'launcher.exe', 'manifest.json', 'assets', 'ffx', 'ff13') {
    $from = Join-Path $src $item
    if (-not (Test-Path -LiteralPath $from)) { continue }
    Copy-Item -LiteralPath $from -Destination $dist -Recurse -Force
    $copied += $item
}

Write-Host ("bundle: dist/ pronto em {0}" -f $dist)
Write-Host ("bundle: conteudo: {0}" -f ($copied -join ', '))
