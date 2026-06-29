# =============================================================================
# sv.ps1 - StreamVault Server Manager
# Usage:
#   .\sv.ps1          - start backend + frontend, then monitor
#   .\sv.ps1 start    - same as above
#   .\sv.ps1 stop     - stop both servers
#   .\sv.ps1 restart  - stop then start
#   .\sv.ps1 status   - show what's running
#   .\sv.ps1 logs     - tail both log files live
#   .\sv.ps1 backend  - start backend only (no frontend)
#   .\sv.ps1 frontend - start frontend dev server only
# =============================================================================

$ROOT        = $PSScriptRoot                                      # streamvault/
$BINARY      = Join-Path $ROOT "streamvault.exe"
$WEB_DIR     = Join-Path $ROOT "web"
$DATA_DIR    = "C:\streamvault-data"
$DB_PATH     = "$DATA_DIR\streamvault.db"
$FFMPEG_BIN  = "$env:LOCALAPPDATA\ffmpeg\bin"
$GO_BIN      = "C:\go\bin"
$BACKEND_PORT  = 8096
$FRONTEND_PORT = 5174
$LOG_DIR     = Join-Path $ROOT "logs"
$BACKEND_LOG = Join-Path $LOG_DIR "backend.log"
$FRONTEND_LOG= Join-Path $LOG_DIR "frontend.log"
$PID_FILE    = Join-Path $LOG_DIR "sv.pids"

# Health check interval (seconds)
$CHECK_INTERVAL = 15

# =============================================================================

function Write-Header {
    Write-Host ""
    Write-Host "  StreamVault Server Manager" -ForegroundColor Cyan
    Write-Host "  Backend  : http://localhost:$BACKEND_PORT" -ForegroundColor DarkCyan
    Write-Host "  Frontend : http://localhost:$FRONTEND_PORT" -ForegroundColor DarkCyan
    Write-Host ""
}

function Log {
    param([string]$Msg, [string]$Color = "White")
    $ts = Get-Date -Format "HH:mm:ss"
    Write-Host "  [$ts] $Msg" -ForegroundColor $Color
}

function Ensure-Dirs {
    if (-not (Test-Path $LOG_DIR))  { New-Item -ItemType Directory -Path $LOG_DIR  -Force | Out-Null }
    if (-not (Test-Path $DATA_DIR)) { New-Item -ItemType Directory -Path $DATA_DIR -Force | Out-Null }
}

# ---------- PATH setup -------------------------------------------------------

function Set-SVPath {
    # Add Go and FFmpeg to PATH for this session
    $extra = @($GO_BIN, $FFMPEG_BIN) | Where-Object { $_ -notin ($env:PATH -split ";") }
    if ($extra) { $env:PATH = ($extra + $env:PATH) -join ";" }
}

# ---------- Binary check -----------------------------------------------------

function Assert-Binary {
    if (-not (Test-Path $BINARY)) {
        Log "streamvault.exe not found - building..." Yellow
        Set-SVPath
        $env:CGO_ENABLED = "0"
        & "$GO_BIN\go.exe" build -o $BINARY ./cmd/streamvault/ 2>&1 | Tee-Object -Append $BACKEND_LOG
        if ($LASTEXITCODE -ne 0) {
            Log "Build failed. Check $BACKEND_LOG" Red
            exit 1
        }
        Log "Build OK" Green
    }
}

# ---------- Port / process helpers -------------------------------------------

function Get-PidOnPort {
    param([int]$Port)
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($conn) { return $conn.OwningProcess } else { return $null }
}

function Kill-Port {
    param([int]$Port, [string]$Name)
    $portPid = Get-PidOnPort $Port
    if ($portPid) {
        try {
            Stop-Process -Id $portPid -Force -ErrorAction SilentlyContinue
            Log "Stopped $Name (PID $portPid on port $Port)" Yellow
        } catch {
            Log "Could not stop $Name on port $Port`: $_" Red
        }
    }
}

function Save-Pids {
    param([int]$BackendPid, [int]$FrontendPid)
    "$BackendPid`n$FrontendPid" | Set-Content $PID_FILE -Encoding UTF8
}

function Load-Pids {
    if (Test-Path $PID_FILE) {
        $lines = Get-Content $PID_FILE
        return @{ Backend = [int]$lines[0]; Frontend = [int]$lines[1] }
    }
    return @{ Backend = 0; Frontend = 0 }
}

# ---------- Health checks -----------------------------------------------------

function Test-Backend {
    try {
        $r = Invoke-WebRequest "http://localhost:$BACKEND_PORT/health" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
        return $r.StatusCode -eq 200
    } catch { return $false }
}

function Test-Frontend {
    try {
        $r = Invoke-WebRequest "http://localhost:$FRONTEND_PORT" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
        return $r.StatusCode -lt 500
    } catch { return $false }
}

# ---------- Start / Stop ------------------------------------------------------

function Start-Backend {
    Kill-Port $BACKEND_PORT "old backend"
    Assert-Binary
    Set-SVPath

    Log "Starting backend on :$BACKEND_PORT ..." Cyan

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName               = $BINARY
    $psi.WorkingDirectory       = $ROOT
    $psi.UseShellExecute        = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError  = $true
    $psi.CreateNoWindow         = $true

    # Environment
    $psi.EnvironmentVariables["SV_DATABASE_URL"]    = $DB_PATH
    $psi.EnvironmentVariables["SV_STORAGE_DATA_DIR"]= $DATA_DIR
    $psi.EnvironmentVariables["SV_SERVER_PORT"]     = "$BACKEND_PORT"
    $psi.EnvironmentVariables["PATH"]               = $env:PATH

    $proc = New-Object System.Diagnostics.Process
    $proc.StartInfo = $psi

    # Async log capture - write to file AND live to console with prefix
    $backendProcRef = [ref]$proc
    $proc.OutputDataReceived += {
        param($s, $e)
        if ($e.Data) {
            $line = "  [backend] $($e.Data)"
            $line | Add-Content $BACKEND_LOG -Encoding UTF8
        }
    }
    $proc.ErrorDataReceived += {
        param($s, $e)
        if ($e.Data) {
            $line = "  [backend] $($e.Data)"
            $line | Add-Content $BACKEND_LOG -Encoding UTF8
        }
    }

    $proc.Start() | Out-Null
    $proc.BeginOutputReadLine()
    $proc.BeginErrorReadLine()

    Log "Backend PID $($proc.Id)" Green
    return $proc
}

function Start-Frontend {
    Kill-Port $FRONTEND_PORT "old frontend"

    if (-not (Test-Path "$WEB_DIR\node_modules")) {
        Log "Installing npm packages..." Yellow
        & npm install --prefix $WEB_DIR 2>&1 | Add-Content $FRONTEND_LOG -Encoding UTF8
    }

    Log "Starting frontend dev server on :$FRONTEND_PORT ..." Cyan

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName               = "cmd.exe"
    $psi.Arguments              = "/c npm run dev"
    $psi.WorkingDirectory       = $WEB_DIR
    $psi.UseShellExecute        = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError  = $true
    $psi.CreateNoWindow         = $true

    $proc = New-Object System.Diagnostics.Process
    $proc.StartInfo = $psi

    $proc.OutputDataReceived += {
        param($s, $e)
        if ($e.Data) { "  [frontend] $($e.Data)" | Add-Content $FRONTEND_LOG -Encoding UTF8 }
    }
    $proc.ErrorDataReceived += {
        param($s, $e)
        if ($e.Data) { "  [frontend] $($e.Data)" | Add-Content $FRONTEND_LOG -Encoding UTF8 }
    }

    $proc.Start() | Out-Null
    $proc.BeginOutputReadLine()
    $proc.BeginErrorReadLine()

    Log "Frontend PID $($proc.Id)" Green
    return $proc
}

function Stop-All {
    Log "Stopping all StreamVault processes..." Yellow
    Kill-Port $BACKEND_PORT  "backend"
    Kill-Port $FRONTEND_PORT "frontend"
    # Also kill any stray node/streamvault processes by name
    Get-Process -Name "streamvault" -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process -Name "node"        -ErrorAction SilentlyContinue |
        Where-Object { $_.MainWindowTitle -eq "" } |
        Stop-Process -Force -ErrorAction SilentlyContinue
    if (Test-Path $PID_FILE) { Remove-Item $PID_FILE -Force }
    Log "All stopped." Yellow
}

# ---------- Status ------------------------------------------------------------

function Show-Status {
    Write-Header
    $bPortPid = Get-PidOnPort $BACKEND_PORT
    $fPortPid = Get-PidOnPort $FRONTEND_PORT
    $bOk  = Test-Backend
    $fOk  = Test-Frontend

    $bStatus = if ($bOk) { "HEALTHY  :$BACKEND_PORT  PID $bPortPid" } `
               elseif ($bPortPid) { "RUNNING (not ready yet)  PID $bPortPid" } `
               else { "STOPPED" }
    $fStatus = if ($fOk) { "HEALTHY  :$FRONTEND_PORT  PID $fPortPid" } `
               elseif ($fPortPid) { "RUNNING (not ready yet)  PID $fPortPid" } `
               else { "STOPPED" }

    $bColor = if ($bOk) { "Green" } elseif ($bPortPid) { "Yellow" } else { "Red" }
    $fColor = if ($fOk) { "Green" } elseif ($fPortPid) { "Yellow" } else { "Red" }

    Write-Host "  Backend  : " -NoNewline; Write-Host $bStatus -ForegroundColor $bColor
    Write-Host "  Frontend : " -NoNewline; Write-Host $fStatus -ForegroundColor $fColor
    Write-Host ""
}

# ---------- Live log tail -----------------------------------------------------

function Show-Logs {
    Log "Tailing logs (Ctrl+C to stop)..." Cyan
    Log "Backend  log: $BACKEND_LOG" DarkGray
    Log "Frontend log: $FRONTEND_LOG" DarkGray
    Write-Host ""

    # Use Get-Content -Wait on both files interleaved
    $jobs = @(
        Start-Job -ScriptBlock {
            param($f, $prefix)
            Get-Content $f -Wait -Tail 20 | ForEach-Object { Write-Output "  [$prefix] $_" }
        } -ArgumentList $BACKEND_LOG,  "backend"

        Start-Job -ScriptBlock {
            param($f, $prefix)
            Get-Content $f -Wait -Tail 20 | ForEach-Object { Write-Output "  [$prefix] $_" }
        } -ArgumentList $FRONTEND_LOG, "frontend"
    )

    try {
        while ($true) {
            $jobs | ForEach-Object {
                Receive-Job $_ | ForEach-Object {
                    $color = if ($_ -match "\[backend\]") { "Cyan" } else { "Magenta" }
                    Write-Host $_ -ForegroundColor $color
                }
            }
            Start-Sleep -Milliseconds 300
        }
    } finally {
        $jobs | Remove-Job -Force
    }
}

# ---------- Monitor loop ------------------------------------------------------

function Start-Monitor {
    param($BackendProc, $FrontendProc)

    Log "Monitoring (Ctrl+C to stop, servers keep running)..." Cyan
    Write-Host ""

    $backendRestarts  = 0
    $frontendRestarts = 0

    try {
        while ($true) {
            $bAlive = $BackendProc  -and -not $BackendProc.HasExited
            $fAlive = $FrontendProc -and -not $FrontendProc.HasExited
            $bOk    = Test-Backend
            $fOk    = Test-Frontend

            # Backend
            $bLabel = if ($bOk) { "OK" } elseif ($bAlive) { "STARTING" } else { "DOWN" }
            $bColor = if ($bOk) { "Green" } elseif ($bAlive) { "Yellow" } else { "Red" }

            # Frontend
            $fLabel = if ($fOk) { "OK" } elseif ($fAlive) { "STARTING" } else { "DOWN" }
            $fColor = if ($fOk) { "Green" } elseif ($fAlive) { "Yellow" } else { "Red" }

            $ts = Get-Date -Format "HH:mm:ss"
            Write-Host "  [$ts]  backend:" -NoNewline
            Write-Host " $bLabel" -ForegroundColor $bColor -NoNewline
            Write-Host "   frontend:" -NoNewline
            Write-Host " $fLabel" -ForegroundColor $fColor

            # Auto-restart backend if crashed
            if (-not $bAlive) {
                $backendRestarts++
                Log "Backend crashed - restarting (#$backendRestarts)..." Red
                $BackendProc = Start-Backend
                Start-Sleep -Seconds 5
            }

            # Auto-restart frontend if crashed
            if (-not $fAlive -and $FrontendProc) {
                $frontendRestarts++
                Log "Frontend crashed - restarting (#$frontendRestarts)..." Red
                $FrontendProc = Start-Frontend
                Start-Sleep -Seconds 3
            }

            Start-Sleep -Seconds $CHECK_INTERVAL
        }
    } finally {
        Log "Monitor stopped. Servers are still running." Yellow
    }
}

# =============================================================================
# Entry point
# =============================================================================

Write-Header
Ensure-Dirs

$cmd = if ($args.Count -gt 0) { $args[0].ToLower() } else { "start" }

switch ($cmd) {

    "stop" {
        Stop-All
    }

    "restart" {
        Stop-All
        Start-Sleep -Seconds 2
        $b = Start-Backend
        Start-Sleep -Seconds 3
        $f = Start-Frontend
        Start-Sleep -Seconds 3
        Start-Monitor $b $f
    }

    "status" {
        Show-Status
    }

    "logs" {
        Show-Logs
    }

    "backend" {
        $b = Start-Backend
        Start-Sleep -Seconds 3
        Start-Monitor $b $null
    }

    "frontend" {
        $f = Start-Frontend
        Start-Sleep -Seconds 2
        Start-Monitor $null $f
    }

    { $_ -in "start", "" } {
        $b = Start-Backend
        Log "Waiting for backend to be ready..." DarkGray
        $deadline = (Get-Date).AddSeconds(20)
        while (-not (Test-Backend) -and (Get-Date) -lt $deadline) { Start-Sleep -Seconds 1 }

        if (Test-Backend) {
            Log "Backend is ready" Green
        } else {
            Log "Backend not responding yet (still starting...)" Yellow
        }

        $f = Start-Frontend
        Start-Sleep -Seconds 2
        Start-Monitor $b $f
    }

    default {
        Write-Host "  Usage: .\sv.ps1 [start|stop|restart|status|logs|backend|frontend]" -ForegroundColor Yellow
        Write-Host ""
    }
}
