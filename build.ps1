$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
New-Item -ItemType Directory -Force -Path build | Out-Null
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
go build -trimpath -ldflags '-s -w -H=windowsgui' -o build/MouseHub.exe .
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
Write-Output 'Built build/MouseHub.exe'
