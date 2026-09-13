$ErrorActionPreference = 'Stop'
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class UI {
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
  [StructLayout(LayoutKind.Sequential)] public struct POINT { public int X, Y; }
  [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr hWnd, out RECT rect);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
  [DllImport("user32.dll")] public static extern bool ClientToScreen(IntPtr hWnd, ref POINT pt);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extra);
  [DllImport("user32.dll")] public static extern IntPtr FindWindowW(string cls, string title);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdc, uint flags);
}
"@
Add-Type -AssemblyName System.Drawing

$proc = Get-Process MouseHub -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1
if (-not $proc) { $proc = Get-Process 'MouseHub*' -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1 }
if (-not $proc) { throw 'QA process not found' }
$h = $proc.MainWindowHandle
if ($h -eq 0) { throw 'QA window not found' }
Write-Host "HWND=$h"
[UI]::ShowWindow($h, 9) | Out-Null
[UI]::SetForegroundWindow($h) | Out-Null
Start-Sleep -Milliseconds 400

$dir = 'H:\CodeFromGithub\All-in-One-Mouse-Driver\build\ui-qa-new'
New-Item -ItemType Directory -Force -Path $dir | Out-Null

function Capture([string]$name) {
  [UI]::ShowWindow($h, 9) | Out-Null
  [UI]::SetForegroundWindow($h) | Out-Null
  Start-Sleep -Milliseconds 250
  $r = New-Object UI+RECT
  [UI]::GetWindowRect($h, [ref]$r) | Out-Null
  $w = $r.Right - $r.Left; $ht = $r.Bottom - $r.Top
  $bmp = New-Object System.Drawing.Bitmap $w, $ht
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $hdc = $g.GetHdc()
  # PW_RENDERFULLCONTENT = 2 captures DWM content
  [UI]::PrintWindow($h, $hdc, 2) | Out-Null
  $g.ReleaseHdc($hdc)
  $g.Dispose()
  $path = Join-Path $dir $name
  $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
  $bmp.Dispose()
  Write-Host "Saved $name"
}

function ClickLogical([int]$lx, [int]$ly) {
  $cr = New-Object UI+RECT
  [UI]::GetClientRect($h, [ref]$cr) | Out-Null
  $cw = $cr.Right - $cr.Left; $ch = $cr.Bottom - $cr.Top
  $px = [int]($lx * $cw / 1280)
  $py = [int]($ly * $ch / 800)
  $pt = New-Object UI+POINT
  $pt.X = $px; $pt.Y = $py
  [UI]::ClientToScreen($h, [ref]$pt) | Out-Null
  Write-Host "Click logical($lx,$ly) -> screen $($pt.X),$($pt.Y) client ${cw}x${ch}"
  [UI]::SetCursorPos($pt.X, $pt.Y) | Out-Null
  Start-Sleep -Milliseconds 80
  [UI]::mouse_event(0x0002, 0, 0, 0, [UIntPtr]::Zero) | Out-Null
  Start-Sleep -Milliseconds 40
  [UI]::mouse_event(0x0004, 0, 0, 0, [UIntPtr]::Zero) | Out-Null
  Start-Sleep -Milliseconds 700
}

Capture '01-home.png'
# 我的配置: x=148,y=20,w=230,h=47 -> center 263,43
ClickLogical 263 43
Capture '02-profiles.png'
# 设备设置 via home: click 主页 first then device card
ClickLogical 83 43
Start-Sleep -Milliseconds 400
Capture '03-home2.png'
# Device card: 290,122,700,505 -> center 640,374
ClickLogical 640 374
Capture '04-performance.png'
# 参数 tab: 420+150=570, y=748, w=140,h=40 -> center 640,768
ClickLogical 640 768
Capture '05-parameters.png'
# 设备信息 tab: 420+300=720 -> center 790,768
ClickLogical 790 768
Capture '06-overview.png'
# back to profiles
ClickLogical 263 43
Start-Sleep -Milliseconds 500
Capture '07-profiles-final.png'

Write-Host 'Done'
Get-ChildItem $dir\*.png | Format-Table Name, Length
