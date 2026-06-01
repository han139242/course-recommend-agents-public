param(
  [switch]$SkipDocker
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$Logs = Join-Path $Root "artifacts/logs"
$FrontendDir = Join-Path $Root "frontend"
$BackendDir = Join-Path $Root "backend"
$ComposeFile = Join-Path $Root "docker/neo4j/docker-compose.yml"

New-Item -ItemType Directory -Force -Path $Logs | Out-Null

function Test-HttpOk {
  param([string]$Url)
  try {
    $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
    return [int]$response.StatusCode -ge 200 -and [int]$response.StatusCode -lt 300
  } catch {
    return $false
  }
}

function Wait-HttpOk {
  param(
    [string]$Url,
    [int]$Retries = 30,
    [int]$DelaySeconds = 2
  )

  for ($i = 1; $i -le $Retries; $i += 1) {
    if (Test-HttpOk $Url) { return $true }
    Start-Sleep -Seconds $DelaySeconds
  }
  return $false
}

function Start-DockerDesktop {
  docker info *> $null
  if ($LASTEXITCODE -eq 0) { return }

  $candidates = @(
    "$env:ProgramFiles/Docker/Docker/Docker Desktop.exe",
    "$env:LOCALAPPDATA/Programs/Docker/Docker/Docker Desktop.exe"
  )
  $dockerDesktop = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
  if (-not $dockerDesktop) {
    throw "Docker Desktop is not running and Docker Desktop.exe was not found."
  }

  Start-Process -FilePath $dockerDesktop -WindowStyle Hidden
  for ($i = 1; $i -le 36; $i += 1) {
    docker info *> $null
    if ($LASTEXITCODE -eq 0) { return }
    Start-Sleep -Seconds 5
  }
  throw "Docker engine did not become ready in time."
}

function Wait-ContainersHealthy {
  $targets = @("course-recommend-mysql", "course-recommend-redis", "course-recommend-neo4j")
  for ($i = 1; $i -le 36; $i += 1) {
    $states = docker inspect $targets --format "{{.Name}} {{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}"
    if (($states -join "`n") -notmatch "starting|created|restarting") { return }
    Start-Sleep -Seconds 5
  }
  throw "Docker dependencies did not become healthy in time."
}

function Start-Backend {
  if (Test-HttpOk "http://127.0.0.1:8080/healthz") { return }

  $stdout = Join-Path $Logs "backend.out.log"
  $stderr = Join-Path $Logs "backend.err.log"
  Start-Process -FilePath "go" `
    -ArgumentList @("run", "./cmd/server") `
    -WorkingDirectory $BackendDir `
    -WindowStyle Hidden `
    -RedirectStandardOutput $stdout `
    -RedirectStandardError $stderr

  if (-not (Wait-HttpOk "http://127.0.0.1:8080/healthz" 40 2)) {
    throw "Backend did not become healthy. See $stderr"
  }
}

function Build-Frontend {
  $npm = (Get-Command npm.cmd -ErrorAction Stop).Source
  Push-Location $FrontendDir
  try {
    & $npm run build
    if ($LASTEXITCODE -ne 0) {
      throw "Frontend build failed."
    }
  } finally {
    Pop-Location
  }
}

if (-not $SkipDocker) {
  Start-DockerDesktop
  docker compose -f $ComposeFile up -d mysql redis neo4j
  Wait-ContainersHealthy
}

Build-Frontend
Start-Backend

$ready = Wait-HttpOk "http://127.0.0.1:8080/readyz" 10 2
if (-not $ready) {
  throw "Backend /readyz is not ready."
}

Write-Host ""
Write-Host "Demo environment is ready."
Write-Host "App:    http://127.0.0.1:8080/"
Write-Host "Show:   http://127.0.0.1:8080/showcase"
Write-Host "Health: http://127.0.0.1:8080/healthz"
Write-Host "Ready:  http://127.0.0.1:8080/readyz"
Write-Host "Logs: $Logs"
