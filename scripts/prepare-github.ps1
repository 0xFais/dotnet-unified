[CmdletBinding()]param([Parameter(Mandatory)][string]$Owner,[string]$Repo='dotnet-unified')
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
$asset=Join-Path $root 'dist\dotnet-unified-0.3.1-win-x64.zip'
$manifest=[ordered]@{
 version='0.3.1';description='Native unified .NET SDK and runtime links from Scoop packages';homepage="https://github.com/$Owner/$Repo";license='MIT';architecture=@{'64bit'=@{url="https://github.com/$Owner/$Repo/releases/download/v0.3.1/dotnet-unified-0.3.1-win-x64.zip";hash=(Get-FileHash $asset -Algorithm SHA256).Hash.ToLowerInvariant()}};bin='dotnet-unified.exe';persist='data';notes=@('Run status, sync --dry-run, sync, doctor.','Re-run sync after SDK updates.')
}
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $root 'bucket\dotnet-unified.json') -Encoding UTF8
Write-Host 'GitHub manifest ready; publish the corresponding release asset.'
