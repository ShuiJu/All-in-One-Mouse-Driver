$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
if (-not $root) { $root = (Get-Location).Path }
# When script lives in build/, root is repo; when called from repo, PSScriptRoot may be build
Set-Location H:\CodeFromGithub\All-in-One-Mouse-Driver
New-Item -ItemType Directory -Force -Path build\ui-qa2 | Out-Null

# Kill previous QA instance if any
Get-Process MouseHub -ErrorAction SilentlyContinue | Where-Object { $_.Path -like '*All-in-One-Mouse-Driver*' } | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 400

Start-Process -FilePath .\build\MouseHub.exe -ArgumentList '--qa','--data','build\ui-qa2'
Start-Sleep -Seconds 2

Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32Cap {
  [StructLayout(LayoutKind.Sequential)]
  public struct RECT { public int Left, Top, Right, Bottom; }
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@

$proc = Get-Process MouseHub | Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1
if (-not $proc) {
  Write-Host 'No MouseHub window found'
  Get-Process MouseHub -ErrorAction SilentlyContinue | Format-Table Id, MainWindowTitle, Path
  exit 1
}
Write-Host "PID=$($proc.Id) Title='$($proc.MainWindowTitle)'"
[Win32Cap]::ShowWindow($proc.MainWindowHandle, 9) | Out-Null
[Win32Cap]::SetForegroundWindow($proc.MainWindowHandle) | Out-Null
Start-Sleep -Milliseconds 500

$r = New-Object Win32Cap+RECT
[Win32Cap]::GetWindowRect($proc.MainWindowHandle, [ref]$r) | Out-Null
$w = $r.Right - $r.Left
$h = $r.Bottom - $r.Top
Write-Host "Window: $($r.Left),$($r.Top) ${w}x${h}"
$bmp = New-Object System.Drawing.Bitmap $w, $h
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($r.Left, $r.Top, 0, 0, $bmp.Size)
$g.Dispose()
$out = Join-Path (Get-Location) 'build\ui-qa2\home.png'
$bmp.Save($out, [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
Write-Host "Saved $out"

if (Test-Path build\ui-qa2\config.json) {
  Write-Host '--- config.json ---'
  Get-Content build\ui-qa2\config.json
}
if (Test-Path build\ui-qa2\mousehub.log) {
  Write-Host '--- log ---'
  Get-Content build\ui-qa2\mousehub.log | Select-Object -Last 20
}
