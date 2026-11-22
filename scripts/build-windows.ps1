#Requires -Version 5.1
<#
.SYNOPSIS
  Build the Postwoman Windows x64 portable ZIP and NSIS installer.

.DESCRIPTION
  Requires Go, Node.js, the Wails CLI and NSIS (makensis). The version is read
  from the repository VERSION file and must match -Version when supplied. The
  script builds the frontend with npm ci, compiles the Wails application with
  embedded WebView2 bootstrapper support, creates the per-user NSIS installer,
  assembles the portable directory, and writes deterministic ZIP output.

.PARAMETER Version
  Expected repository version (MAJOR.MINOR.PATCH). Defaults to VERSION.

.PARAMETER OutputDir
  Directory for release assets. Defaults to dist\windows.

.PARAMETER SkipFrontend
  Reuse an existing frontend/dist build instead of running npm ci/build.

.PARAMETER NoInstaller
  Build only the portable executable, skipping NSIS.

.EXAMPLE
  pwsh ./scripts/build-windows.ps1 -Version 1.4.0
#>
[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$OutputDir = "dist\windows",
    [switch]$SkipFrontend,
    [switch]$NoInstaller
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Fail([string]$Message) {
    Write-Error "error: $Message"
    exit 1
}

function Require-Command([string]$Name, [string]$Hint) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Fail "$Name not found. $Hint"
    }
}

function Find-Executable([string]$Name, [string[]]$Candidates) {
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command -and $command.Source) { return $command.Source }
    foreach ($directory in $Candidates) {
        if (-not $directory) { continue }
        foreach ($file in @($Name, "$Name.exe")) {
            $candidate = Join-Path $directory $file
            if (Test-Path -LiteralPath $candidate) { return $candidate }
        }
    }
    return $null
}

function Add-ToolDirectory([string]$ExecutablePath) {
    if (-not $ExecutablePath) { return }
    $directory = Split-Path -Parent $ExecutablePath
    if ($directory -and (($env:Path -split ';') -notcontains $directory)) {
        $env:Path = "$directory;$env:Path"
    }
}

function Assert-File([string]$Path, [long]$MinBytes, [string]$Label) {
    if (-not (Test-Path -LiteralPath $Path)) { Fail "$Label is missing: $Path" }
    $item = Get-Item -LiteralPath $Path
    if ($item.Length -lt $MinBytes) { Fail "$Label is too small: $Path ($($item.Length) bytes)" }
    return $item
}

function Assert-PE([string]$Path, [string]$Label) {
    $bytes = [System.IO.File]::ReadAllBytes($Path)
    if ($bytes.Length -lt 2 -or $bytes[0] -ne 0x4D -or $bytes[1] -ne 0x5A) {
        Fail "$Label is not a Windows PE executable: $Path"
    }
}

$repoVersion = (Get-Content -LiteralPath (Join-Path $Root "VERSION") -Raw).Trim()
if (-not $Version) { $Version = $repoVersion }
if ($Version -notmatch '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$') {
    Fail "version must be MAJOR.MINOR.PATCH, got '$Version'"
}
if ($Version -ne $repoVersion) {
    Fail "requested version $Version does not match VERSION ($repoVersion); run scripts/update-version.mjs first"
}

Require-Command node "Install Node.js 20+."
Require-Command npm "Install Node.js 20+."
Require-Command go "Install Go 1.27+ (https://go.dev/dl/)."

$goBin = Join-Path (go env GOPATH) "bin"
$userGoBin = Join-Path $env:USERPROFILE "go\bin"
$wailsExe = Find-Executable "wails" @($goBin, $userGoBin)
if (-not $wailsExe) {
    Fail "wails.exe not found. Install it with: go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1"
}
Add-ToolDirectory $wailsExe
Write-Host "Using Wails: $wailsExe"

$gccExe = Find-Executable "gcc" @(
    $env:MINGW_BIN,
    "C:\msys64\mingw64\bin",
    "C:\mingw64\bin",
    "C:\ProgramData\chocolatey\bin"
)
if (-not $gccExe) {
    Fail "gcc.exe not found. Install MinGW-w64 (for example 'choco install mingw'); SQLite requires CGO."
}
Add-ToolDirectory $gccExe
$env:CGO_ENABLED = "1"
$env:CC = $gccExe
Write-Host "Using C compiler: $gccExe"

if (-not $NoInstaller) {
    $programFilesX86 = ${env:ProgramFiles(x86)}
    if (-not $programFilesX86) { $programFilesX86 = $env:ProgramFiles }
    $makensisExe = Find-Executable "makensis" @(
        $env:NSIS_DIR,
        (Join-Path $programFilesX86 "NSIS"),
        (Join-Path $env:ProgramFiles "NSIS"),
        "C:\ProgramData\chocolatey\lib\nsis\tools",
        "C:\ProgramData\chocolatey\bin"
    )
    if (-not $makensisExe) {
        Fail "makensis.exe not found. Install NSIS 3 from https://nsis.sourceforge.io/ or 'choco install nsis'."
    }
    Add-ToolDirectory $makensisExe
    Write-Host "Using NSIS: $makensisExe"
}

& node scripts/update-version.mjs --check $Version
if ($LASTEXITCODE -ne 0) { Fail "version metadata is inconsistent" }

if (-not $SkipFrontend) {
    Write-Host "==> building frontend"
    Push-Location (Join-Path $Root "frontend")
    try {
        & npm ci --no-audit --no-fund
        if ($LASTEXITCODE -ne 0) { Fail "npm ci failed" }
        & npm run build
        if ($LASTEXITCODE -ne 0) { Fail "frontend build failed" }
    } finally {
        Pop-Location
    }
}
Assert-File (Join-Path $Root "frontend\dist\index.html") 100 "frontend build" | Out-Null

Write-Host "==> building Windows application"
$ldflags = "-X main.appVersion=$Version"
$wailsArgs = @(
    "build",
    "-clean",
    "-platform", "windows/amd64",
    "-trimpath",
    "-webview2", "embed",
    "-ldflags", $ldflags,
    "-m",
    "-s"
)
if (-not $NoInstaller) {
    $wailsArgs += @("-nsis", "-installscope", "user")
}
& $wailsExe @wailsArgs
if ($LASTEXITCODE -ne 0) { Fail "wails build failed" }

$exe = Assert-File (Join-Path $Root "build\bin\Postwoman.exe") 1000000 "Windows executable"
Assert-PE $exe.FullName "Windows executable"
$exeVersion = (Get-Item -LiteralPath $exe.FullName).VersionInfo
$exeProduct = "$($exeVersion.ProductVersion)"
$exeFile = "$($exeVersion.FileVersion)"
if (-not ($exeProduct.StartsWith($Version) -or $exeFile.StartsWith($Version))) {
    Fail "executable version '$exeProduct' / '$exeFile' does not match $Version"
}
Write-Host "    executable version: product='$exeProduct' file='$exeFile'"

$installer = $null
if (-not $NoInstaller) {
    $candidates = Get-ChildItem -Path (Join-Path $Root "build\bin") -Filter "*installer*.exe" -File |
        Sort-Object LastWriteTime -Descending
    if (-not $candidates) { Fail "NSIS installer was not produced in build\bin" }
    $installer = Assert-File $candidates[0].FullName 1000000 "NSIS installer"
    Assert-PE $installer.FullName "NSIS installer"
    $installerVersion = (Get-Item -LiteralPath $installer.FullName).VersionInfo
    $installerProduct = "$($installerVersion.ProductVersion)"
    $installerFile = "$($installerVersion.FileVersion)"
    if (-not ($installerProduct.StartsWith($Version) -or $installerFile.StartsWith($Version))) {
        Fail "installer version '$installerProduct' / '$installerFile' does not match $Version"
    }
    Write-Host "    installer version:  product='$installerProduct' file='$installerFile'"
}

$out = Join-Path $Root $OutputDir
if (Test-Path -LiteralPath $out) { Remove-Item -LiteralPath $out -Recurse -Force }
New-Item -ItemType Directory -Path $out -Force | Out-Null

$portableName = "Postwoman-v$Version-windows-x64-portable"
$stage = Join-Path $out $portableName
New-Item -ItemType Directory -Path $stage -Force | Out-Null

Copy-Item -LiteralPath $exe.FullName -Destination (Join-Path $stage "Postwoman.exe")
Copy-Item -LiteralPath (Join-Path $Root "packaging\windows\PORTABLE-README.txt") -Destination (Join-Path $stage "README.txt")
Copy-Item -LiteralPath (Join-Path $Root "LICENSE") -Destination (Join-Path $stage "LICENSE")
Set-Content -LiteralPath (Join-Path $stage "portable.txt") -Encoding ASCII -Value @(
    "Postwoman portable mode",
    "Data is stored in the data folder next to Postwoman.exe.",
    "Keep Postwoman.exe, portable.txt and data together when moving this folder."
)
Write-Host "==> creating portable ZIP"
Add-Type -AssemblyName System.IO.Compression.FileSystem
$zipPath = Join-Path $out "$portableName.zip"
$archive = [System.IO.Compression.ZipFile]::Open($zipPath, [System.IO.Compression.ZipArchiveMode]::Create)
try {
    $files = Get-ChildItem -LiteralPath $stage -Recurse -File | Sort-Object FullName
    foreach ($file in $files) {
        $relative = $file.FullName.Substring($stage.Length + 1).Replace('\', '/')
        $entry = $archive.CreateEntry("$portableName/$relative", [System.IO.Compression.CompressionLevel]::Optimal)
        # Fixed timestamp keeps the archive reproducible across runners.
        $entry.LastWriteTime = [DateTimeOffset]::new(2020, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
        $sourceStream = [System.IO.File]::OpenRead($file.FullName)
        $destStream = $entry.Open()
        try { $sourceStream.CopyTo($destStream) } finally { $destStream.Dispose(); $sourceStream.Dispose() }
    }
} finally {
    $archive.Dispose()
}
Remove-Item -LiteralPath $stage -Recurse -Force

$zip = [System.IO.Compression.ZipFile]::OpenRead($zipPath)
try {
    $names = @($zip.Entries | ForEach-Object { $_.FullName })
} finally {
    $zip.Dispose()
}
foreach ($required in @("$portableName/Postwoman.exe", "$portableName/portable.txt", "$portableName/README.txt", "$portableName/LICENSE")) {
    if ($names -notcontains $required) { Fail "portable ZIP is missing $required" }
}

$setupPath = $null
if ($installer) {
    $setupPath = Join-Path $out "Postwoman-v$Version-windows-x64-setup.exe"
    Move-Item -LiteralPath $installer.FullName -Destination $setupPath
    Assert-PE $setupPath "renamed installer"
}

$assets = Get-ChildItem -LiteralPath $out -File | Sort-Object Name
$sums = foreach ($asset in $assets) {
    $hash = (Get-FileHash -LiteralPath $asset.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($asset.Name)"
}
Set-Content -LiteralPath (Join-Path $out "SHA256SUMS.txt") -Encoding ASCII -Value $sums

Write-Host ""
Write-Host "Windows assets in ${out}:"
$assets | ForEach-Object { Write-Host ("  {0}  ({1:N0} bytes)" -f $_.Name, $_.Length) }
Write-Host "  SHA256SUMS.txt"