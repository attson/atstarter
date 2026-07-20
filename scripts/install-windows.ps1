# Install a new atstarter windows build. Invoked by the running app right
# before it exits. Arguments (positional):
#   -Asset  <path>  validated NSIS .exe on disk
#   -Target <path>  target install directory (ignored — the NSIS installer
#                   knows the install location from its own metadata)
#   -Exec   <path>  exec path used to relaunch after install
param(
  [Parameter(Mandatory = $true)][string]$Asset,
  [Parameter(Mandatory = $true)][string]$Target,
  [Parameter(Mandatory = $true)][string]$Exec
)
$ErrorActionPreference = "Stop"

function log($msg) { Write-Host "[atstarter-install] $msg" }

if (-not (Test-Path -LiteralPath $Asset)) {
  log "asset missing: $Asset"; exit 1
}

# NSIS silent install: /S. The installer overwrites the existing files in
# place. Wait for it so we know when the new binary is ready.
log "running installer $Asset"
Start-Process -FilePath $Asset -ArgumentList "/S" -Wait

# Relaunch. Start-Process detaches so the caller (which is about to exit)
# doesn't hold onto the child.
if (Test-Path -LiteralPath $Exec) {
  log "relaunching $Exec"
  Start-Process -FilePath $Exec | Out-Null
} else {
  log "post-install exec not found at $Exec; skipping relaunch"
}
