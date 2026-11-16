#Requires -Version 5.1
<#
.SYNOPSIS
  Publica um release misto do monorepo ff-resources.

.DESCRIPTION
  Um release por lote:
    - VERSION (raiz) = versão do release train (semver, monotônica);
    - apps/<app>/VERSION = versão exibida do app;
    - um zip por app incluído + ff-resources-suite.zip + SHA256SUMS;
    - tag v<train> no repositório atual (GitHub).

  Os apps que não estão no lote continuam nos releases anteriores: o updater
  deles recebe o release novo, não acha o asset com o próprio prefixo e
  ignora (noAssetIsUpToDate).

.EXAMPLE
  task release:ff13
  task release:all
  powershell -File scripts/release.ps1 -Apps ffx -DryRun
#>
[CmdletBinding()]
param(
    # Apps incluídos no lote: ffx, ff13, launcher (separados por vírgula).
    [string]$Apps = 'ffx,ff13',
    # Monta/publica só a suite (sem zips de app).
    [switch]$SuiteOnly,
    # Faz build, zip e hash, mas não publica no GitHub.
    [switch]$DryRun,
    # Marca o release como prerelease no GitHub.
    [switch]$Prerelease,
    # Publica como draft (não aparece no /releases/latest até publicar).
    [switch]$Draft,
    # Reutiliza os binários já buildados (não roda wails3 build).
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'

$root = (Resolve-Path "$PSScriptRoot/..").Path
# O Windows PowerShell pode rodar sem o Git no PATH (o app usa o instalador
# padrão). Localiza o executável uma vez; gh lê o remote sozinho.
$gitExe = (Get-Command git -ErrorAction SilentlyContinue).Source
if (-not $gitExe) { $gitExe = "C:\Program Files\Git\cmd\git.exe" }
# O `gh` também exige o git no PATH (ele chama git internamente).
$gitDir = Split-Path -Parent $gitExe
if ($gitDir -and (($env:PATH -split ';') -notcontains $gitDir)) {
    $env:PATH = "$gitDir;$env:PATH"
}
$releaseDir = Join-Path $root 'dist/release'
$suiteDir = Join-Path $root 'dist/suite'
$versionFile = Join-Path $root 'VERSION'

function Read-Trimmed([string]$path) {
    if (-not (Test-Path -LiteralPath $path)) { throw "arquivo ausente: $path" }
    return (Get-Content -LiteralPath $path -Raw).Trim()
}

function Assert-Semver([string]$v, [string]$what) {
    if ($v -notmatch '^\d+\.\d+\.\d+$') {
        throw "$what inválida: '$v' (esperado X.Y.Z)"
    }
}

function Compare-Semver([string]$a, [string]$b) {
    $pa = $a.Split('.') | ForEach-Object { [int]$_ }
    $pb = $b.Split('.') | ForEach-Object { [int]$_ }
    for ($i = 0; $i -lt 3; $i++) {
        if ($pa[$i] -gt $pb[$i]) { return 1 }
        if ($pa[$i] -lt $pb[$i]) { return -1 }
    }
    return 0
}

$version = Read-Trimmed $versionFile
Assert-Semver $version 'VERSION (train)'
$tag = "v$version"

# Configuração de cada app: nome do zip/asset, exe e VERSION própria.
$appMap = @{
    ffx     = @{ Asset = 'spira';    Exe = 'spira.exe';    Dir = 'apps/ffx';     Version = 'apps/ffx/VERSION' }
    ff13    = @{ Asset = 'pulse';    Exe = 'pulse.exe';    Dir = 'apps/ff13';    Version = 'apps/ff13/VERSION' }
    launcher = @{ Asset = 'launcher'; Exe = 'launcher.exe'; Dir = 'apps/launcher'; Version = 'apps/launcher/VERSION' }
}

$selected = @()
if (-not $SuiteOnly) {
    $selected = $Apps.Split(',') | ForEach-Object { $_.Trim().ToLower() } | Where-Object { $_ -ne '' } | Sort-Object -Unique
    foreach ($a in $selected) {
        if (-not $appMap.ContainsKey($a)) { throw "app desconhecido: $a (use ffx, ff13, launcher)" }
    }
    if ($selected.Count -eq 0) { throw "nenhum app selecionado (use -SuiteOnly para só a suite)" }
}

# A tag tem de ser nova e maior que a última tag v*.
$existing = & $gitExe tag --list "$tag" 2>$null
if ($existing) { throw "tag $tag já existe" }
$lastTag = (& $gitExe tag --list 'v*' --sort=-v:refname 2>$null | Select-Object -First 1)
if ($lastTag) {
    $last = $lastTag.TrimStart('v')
    if ($last -match '^\d+\.\d+\.\d+$' -and (Compare-Semver $version $last) -le 0) {
        throw "VERSION $version precisa ser maior que a última tag $lastTag"
    }
}

Write-Host "release: train $version ($tag) apps: $($selected -join ', ')"
New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null

$assets = New-Object System.Collections.Generic.List[string]

# 1) Build dos apps do lote (a versão entra por -ldflags no Taskfile).
if (-not $SkipBuild) {
    foreach ($a in $selected) {
        $cfg = $appMap[$a]
        Write-Host "release: build $a ($($cfg.Dir))"
        Push-Location (Join-Path $root $cfg.Dir)
        try { & wails3 build } finally { Pop-Location }
    }
    # A suite precisa dos binários atuais dos dois jogos e do launcher; builda
    # o que faltar (sem publicar zip de app fora do lote).
    foreach ($a in @('ffx', 'ff13')) {
        $cfg = $appMap[$a]
        $exe = Join-Path $root (Join-Path $cfg.Dir (Join-Path 'bin' $cfg.Exe))
        if (-not (Test-Path -LiteralPath $exe)) {
            Write-Host "release: build $a (necessário para a suite)"
            Push-Location (Join-Path $root $cfg.Dir)
            try { & wails3 build } finally { Pop-Location }
        }
    }
    $launcherExe = Join-Path $root 'apps/launcher/bin/launcher.exe'
    if (-not (Test-Path -LiteralPath $launcherExe)) {
        Write-Host "release: build launcher (necessário para a suite)"
        & wails3 task build:launcher
        if ($LASTEXITCODE -ne 0) { throw "build:launcher falhou" }
    }
}

# 2) Zip de cada app do lote.
foreach ($a in $selected) {
    $cfg = $appMap[$a]
    $exe = Join-Path $root (Join-Path $cfg.Dir (Join-Path 'bin' $cfg.Exe))
    if (-not (Test-Path -LiteralPath $exe)) {
        throw "binário ausente: $exe (rode task build:$a)"
    }
    $appVersion = Read-Trimmed (Join-Path $root $cfg.Version)
    Assert-Semver $appVersion "$($cfg.Version)"
    $zipName = "$($cfg.Asset)-$version-portable-windows-amd64.zip"
    $zipPath = Join-Path $releaseDir $zipName
    if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
    Compress-Archive -LiteralPath $exe -DestinationPath $zipPath -CompressionLevel Optimal
    Write-Host "release: $zipName (app $a $appVersion)"
    $assets.Add($zipPath)
}

# 3) Suite: o launcher + os apps atuais + manifest/assets. Serve para quem
#    ainda não tem nada; os updaters ignoram (nome sem prefixo de app).
if (-not (Test-Path -LiteralPath $suiteDir)) { New-Item -ItemType Directory -Force -Path $suiteDir | Out-Null }
Get-ChildItem -LiteralPath $suiteDir -Force | Remove-Item -Recurse -Force
& wails3 task stage:launcher
if ($LASTEXITCODE -ne 0) { throw "stage:launcher falhou" }
Copy-Item -Path (Join-Path $root 'apps/launcher/bin/*') -Destination $suiteDir -Recurse -Force
Copy-Item -LiteralPath $versionFile -Destination (Join-Path $suiteDir 'VERSION') -Force

$versions = [ordered]@{
    release = $version
    launcher = Read-Trimmed (Join-Path $root 'apps/launcher/VERSION')
    ffx = Read-Trimmed (Join-Path $root 'apps/ffx/VERSION')
    ff13 = Read-Trimmed (Join-Path $root 'apps/ff13/VERSION')
}
$versions | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $suiteDir 'versions.json') -Encoding utf8

$suiteZip = Join-Path $releaseDir 'ff-resources-suite.zip'
if (Test-Path -LiteralPath $suiteZip) { Remove-Item -LiteralPath $suiteZip -Force }
Compress-Archive -Path (Join-Path $suiteDir '*') -DestinationPath $suiteZip -CompressionLevel Optimal
Write-Host "release: ff-resources-suite.zip"
$assets.Add($suiteZip)

# 4) SHA256SUMS de todos os zips publicados.
$sumsPath = Join-Path $releaseDir 'SHA256SUMS'
$lines = foreach ($zip in $assets) {
    $hash = (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $(Split-Path -Leaf $zip)"
}
$lines | Set-Content -LiteralPath $sumsPath -Encoding ascii
Write-Host "release: SHA256SUMS"
$assets.Add($sumsPath)

Write-Host "release: artefatos:"
$assets | ForEach-Object { Write-Host "  $_" }

if ($DryRun) {
    Write-Host "release: DRY-RUN — não publicando $tag"
    exit 0
}

# 5) Publica no repo do origin (gh usa o remote atual).
$notesFile = Join-Path $releaseDir 'notes.md'
$ghArgs = @('release', 'create', $tag) + $assets + @('--title', $version)
if (Test-Path -LiteralPath $notesFile) {
    $ghArgs += @('--notes-file', $notesFile)
} else {
    $ghArgs += '--generate-notes'
}
if ($Prerelease) { $ghArgs += '--prerelease' }
if ($Draft) { $ghArgs += '--draft' }

Write-Host "release: publicando $tag"
& gh @ghArgs
if ($LASTEXITCODE -ne 0) { throw "gh release create falhou" }
Write-Host "release: publicado $tag"
