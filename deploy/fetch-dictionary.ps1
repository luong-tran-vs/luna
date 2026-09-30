# Downloads the offline English-Vietnamese dictionary (minhqnd/dictionary v2.0.0, MIT) used by
# the Reading step, into deploy/data/dictionary/dictionary.db, and verifies its SHA-256.
$ErrorActionPreference = 'Stop'

$Url = 'https://github.com/minhqnd/dictionary/releases/download/v2.0.0/dictionary.db'
$Sha256 = '9259403f0675b2991a1bd0ef6d0dbc5933afdb135632af095a60662f09bbf1d3'
$Dir = Join-Path $PSScriptRoot 'data\dictionary'
$File = Join-Path $Dir 'dictionary.db'
$Part = "$File.part"

New-Item -ItemType Directory -Force -Path $Dir | Out-Null
if ((Test-Path $File) -and ((Get-FileHash $File -Algorithm SHA256).Hash.ToLower() -eq $Sha256)) {
    Write-Host "Dictionary already present: $File"
    exit 0
}

Write-Host 'Downloading dictionary (~180 MB)...'
$ProgressPreference = 'SilentlyContinue'  # much faster Invoke-WebRequest
Invoke-WebRequest -Uri $Url -OutFile $Part
if ((Get-FileHash $Part -Algorithm SHA256).Hash.ToLower() -ne $Sha256) {
    Remove-Item $Part -Force
    throw 'Checksum mismatch, download removed.'
}
Move-Item $Part $File -Force
Write-Host "Dictionary ready: $File"
