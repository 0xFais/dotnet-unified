[CmdletBinding()]param()
$ErrorActionPreference='Stop'
$project=Split-Path $PSScriptRoot -Parent
$asset=Join-Path $project 'dist\dotnet-unified-0.3.1-win-x64.zip'
$manifest=[ordered]@{
 version='0.3.1';description='Native unified .NET SDK and runtime links from Scoop packages';homepage='https://github.com/CHANGE_ME/dotnet-unified';license='MIT';architecture=@{'64bit'=@{url=([Uri]$asset).AbsoluteUri;hash=(Get-FileHash $asset -Algorithm SHA256).Hash.ToLowerInvariant()}};bin='dotnet-unified.exe';persist='data';notes=@('Run status, sync --dry-run, sync, doctor.','Re-run sync after SDK updates.')
}
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $project 'bucket\dotnet-unified.json') -Encoding UTF8
Write-Host 'Local Scoop manifest is ready.'
