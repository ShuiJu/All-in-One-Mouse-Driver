param([int]$Seconds = 30, [string]$Output = 'research/idle-performance.json')
$ErrorActionPreference = 'Stop'
if ($Seconds -lt 5 -or $Seconds -gt 60) { throw 'Use a 5–60 second sample.' }
$process = Get-Process MouseHub
if (@($process).Count -ne 1) { throw 'Expected one MouseHub process.' }
$sampleId = $process.Id
$before = $process.CPU
$start = Get-Date
Start-Sleep -Seconds $Seconds
$after = Get-Process -Id $sampleId
$elapsed = ((Get-Date)-$start).TotalSeconds
$result = [pscustomobject]@{
    Process = $after.ProcessName
    ProcessId = $sampleId
    DurationSeconds = $elapsed
    CpuSeconds = $after.CPU - $before
    SingleCorePercent = 100 * ($after.CPU - $before) / $elapsed
    WorkingSetMiB = $after.WorkingSet64 / 1MB
    PrivateMiB = $after.PrivateMemorySize64 / 1MB
    Handles = $after.HandleCount
}
$result | ConvertTo-Json | Set-Content -LiteralPath $Output
$result
