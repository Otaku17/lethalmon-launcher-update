<#
.SYNOPSIS
  Signs a built launcher .exe with an Authenticode code-signing certificate.

.DESCRIPTION
  Optional, opt-in step: nothing in the normal `wails build` flow calls this
  automatically, so it can't break the existing build. Run it manually after
  `wails build` once you have a code-signing certificate.

  Signing is the actual long-term fix for Windows Defender/SmartScreen
  flagging the launcher (e.g. "Trojan:Win32/Bearfoos.A!ml") — an unsigned
  .exe with little download history has no reputation for SmartScreen to
  trust, no matter how the file behaves. Behavioral tweaks to the updater
  script reduce false-positive triggers, but only a real certificate builds
  reputation over time.

.REQUIREMENTS
  - signtool.exe (ships with the Windows SDK; already on PATH if Visual
    Studio or the standalone Windows SDK is installed)
  - A code-signing certificate (.pfx) from a Certificate Authority, or a
    cloud signing service (e.g. Azure Trusted Signing) if you're not using
    a local .pfx file

.EXAMPLE
  ./build/windows/sign.ps1 -ExePath "build/bin/Lethal Launcher.exe" -CertPath "C:\path\to\cert.pfx" -CertPassword "..."
#>
param(
    [Parameter(Mandatory = $true)][string]$ExePath,
    [Parameter(Mandatory = $true)][string]$CertPath,
    [Parameter(Mandatory = $true)][string]$CertPassword,
    # Any RFC3161 timestamp server works; this is just a commonly used
    # public default — swap it for your CA's if they recommend another.
    [string]$TimestampUrl = "http://timestamp.digicert.com"
)

if (-not (Test-Path $ExePath)) {
    throw "File not found: $ExePath"
}

if (-not (Test-Path $CertPath)) {
    throw "Certificate not found: $CertPath"
}

& signtool sign /f "$CertPath" /p "$CertPassword" /fd SHA256 /tr "$TimestampUrl" /td SHA256 "$ExePath"

if ($LASTEXITCODE -ne 0) {
    throw "signtool failed with exit code $LASTEXITCODE"
}

Write-Host "Signed: $ExePath"
