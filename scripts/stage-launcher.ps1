#Requires -Version 5.1
<#
.SYNOPSIS
  Deixa os builds de ffx e ff13 acessiveis na distribuicao do launcher.

.DESCRIPTION
  O launcher e a tela de escolha: ele le manifest.json ao lado do executavel e
  abre o app escolhido. Para isso funcionar em dev e em build, os exes dos jogos
  precisam estar ao lado de launcher.exe, em apps/launcher/bin/.

  Nomes de distribuicao (manifest.json):
    apps/ffx/bin/spira.exe   -> apps/launcher/bin/ffx/spira.exe
    apps/ff13/bin/pulse.exe  -> apps/launcher/bin/ff13/pulse.exe

  Tambem copia manifest.json e assets/ (capas) para a mesma pasta.
#>
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$root = (Resolve-Path "$PSScriptRoot/..").Path
$bin = Join-Path $root 'apps/launcher/bin'

$apps = @(
    @{ Src = 'apps/ffx/bin/spira.exe';  Dst = 'ffx/spira.exe' }
    @{ Src = 'apps/ff13/bin/pulse.exe'; Dst = 'ff13/pulse.exe' }
)

foreach ($app in $apps) {
    $src = Join-Path $root $app.Src
    if (-not (Test-Path -LiteralPath $src)) {
        throw ("build nao encontrado: {0}`nRode 'task build:ffx' e 'task build:ff13' antes de 'task stage:launcher'." -f $src)
    }
    $dst = Join-Path $bin $app.Dst
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dst) | Out-Null
    Copy-Item -LiteralPath $src -Destination $dst -Force
    $mb = [math]::Round((Get-Item -LiteralPath $dst).Length / 1MB, 1)
    Write-Host ("stage: {0} -> {1} ({2} MB)" -f $app.Src, $app.Dst, $mb)
}

# manifest.json: o launcher procura ao lado do executavel.
$manifestSrc = Join-Path $root 'manifest.json'
if (-not (Test-Path -LiteralPath $manifestSrc)) {
    throw "manifest.json nao encontrado na raiz: $manifestSrc"
}
Copy-Item -LiteralPath $manifestSrc -Destination (Join-Path $bin 'manifest.json') -Force

# Capas do launcher (assets/launcher/*.png referenciadas pelo manifest).
$assets = Join-Path $root 'assets'
if (Test-Path -LiteralPath $assets) {
    Copy-Item -LiteralPath $assets -Destination $bin -Recurse -Force
}

Write-Host "stage: pronto em $bin (manifest.json + ffx/spira.exe + ff13/pulse.exe ao lado de launcher.exe)"
