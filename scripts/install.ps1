$ErrorActionPreference = "Stop"
$RepoDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$BinDir = if ($env:VCSM_BIN_DIR) { $env:VCSM_BIN_DIR } else { Join-Path $env:LOCALAPPDATA "VCSM\bin" }

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go 1.24 or newer is required to build VCSM."
}

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
Push-Location $RepoDir
try {
    go build -trimpath -o (Join-Path $BinDir "vcsm.exe") ./cmd/vcsm
    go build -trimpath -o (Join-Path $BinDir "vcsm-broker.exe") ./cmd/vcsm-broker
} finally {
    Pop-Location
}

$Broker = Join-Path $BinDir "vcsm-broker.exe"
schtasks.exe /Delete /TN "VCSM Broker" /F 2>$null | Out-Null
schtasks.exe /Create /TN "VCSM Broker" /SC ONLOGON /RL LIMITED /TR "`"$Broker`"" /F | Out-Null
Start-Process -FilePath $Broker -WindowStyle Hidden

Write-Host "Installed vcsm and its per-user broker."
Write-Host "Add $BinDir to PATH, then run: vcsm init"
