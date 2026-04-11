param(
    [string]$OutputRoot = "",
    [switch]$SkipProjectZip,
    [switch]$SkipMySQLDump,
    [switch]$SkipNeo4jDump,
    [switch]$SkipRedisDump
)

$ErrorActionPreference = "Stop"

function Write-Step([string]$Message) {
    Write-Host ""
    Write-Host "== $Message ==" -ForegroundColor Cyan
}

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Invoke-TextCommand([scriptblock]$ScriptBlock) {
    $output = & $ScriptBlock
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code $LASTEXITCODE"
    }
    return ($output | Out-String).Trim()
}

function Wait-HttpReady([string]$Url, [int]$TimeoutSeconds = 90) {
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 5
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
                return
            }
        } catch {
        }
        Start-Sleep -Seconds 2
    }
    throw "Timed out waiting for $Url"
}

function Wait-ContainerHealthy([string]$ContainerName, [int]$TimeoutSeconds = 120) {
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $status = (& docker inspect --format "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}" $ContainerName 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -eq 0 -and ($status -eq "healthy" -or $status -eq "running")) {
            return
        }
        Start-Sleep -Seconds 2
    }
    throw "Timed out waiting for container $ContainerName to become healthy"
}

function Copy-RelativePath([string]$RepoRoot, [string]$RelativePath, [string]$SnapshotRoot) {
    $source = Join-Path $RepoRoot $RelativePath
    if (-not (Test-Path $source)) {
        return
    }

    $destination = Join-Path $SnapshotRoot $RelativePath
    if ((Get-Item $source) -is [System.IO.DirectoryInfo]) {
        Ensure-Dir $destination
        Copy-Item -Path (Join-Path $source "*") -Destination $destination -Recurse -Force
        return
    }

    Ensure-Dir (Split-Path -Parent $destination)
    Copy-Item -Path $source -Destination $destination -Force
}

function Get-RepoSnapshotPaths([string]$RepoRoot) {
    $tracked = @(& git -C $RepoRoot -c core.quotepath=false ls-files)
    if ($LASTEXITCODE -ne 0) {
        throw "git ls-files failed"
    }
    $others = @(& git -C $RepoRoot -c core.quotepath=false ls-files --others --exclude-standard)
    if ($LASTEXITCODE -ne 0) {
        throw "git ls-files --others failed"
    }

    $manualIncludes = @(
        "backend/web/dist",
        "docker/neo4j/.env.example"
    )
    $excludes = @(
        "backend/server.exe",
        ".tmp-playwright"
    )

    $all = @($tracked + $others + $manualIncludes) |
        Where-Object { $_ -and ($_ -notin $excludes) -and (-not $_.StartsWith("backend/backend-local")) } |
        Sort-Object -Unique

    return $all
}

function Export-ProjectSnapshot([string]$RepoRoot, [string]$BundleRoot) {
    Write-Step "Export project snapshot"

    $snapshotRoot = Join-Path $BundleRoot "project-src"
    $archivePath = Join-Path $BundleRoot "project-src.zip"
    Ensure-Dir $snapshotRoot

    $paths = Get-RepoSnapshotPaths $RepoRoot
    foreach ($relativePath in $paths) {
        Copy-RelativePath $RepoRoot $relativePath $snapshotRoot
    }

    if (Test-Path $archivePath) {
        Remove-Item $archivePath -Force
    }
    Compress-Archive -Path (Join-Path $snapshotRoot "*") -DestinationPath $archivePath -Force
    Remove-Item $snapshotRoot -Recurse -Force

    return $archivePath
}

function Export-MySQLDump([string]$OutputFile) {
    Write-Step "Export MySQL dump"
    $dump = Invoke-TextCommand {
        docker exec course-recommend-mysql sh -lc "mysqldump -uapp -papp_dev_pass --single-transaction --routines --triggers --no-tablespaces --default-character-set=utf8mb4 course_recommend"
    }
    [System.IO.File]::WriteAllText($OutputFile, $dump, [System.Text.UTF8Encoding]::new($false))
}

function Export-Neo4jSummary([string]$OutputFile) {
    Write-Step "Export Neo4j summary"
    $summary = Invoke-TextCommand {
        docker exec course-recommend-neo4j cypher-shell -u neo4j -p 123456789 "MATCH (n:KnowledgeNode) RETURN count(n) AS nodes; MATCH ()-[r]->() RETURN count(r) AS relations; MATCH (n:KnowledgeNode) WHERE NOT (n)--() RETURN count(n) AS isolated; MATCH (n:KnowledgeNode) WHERE trim(coalesce(n.description,'')) = '' RETURN count(n) AS missing_description; MATCH (n:KnowledgeNode) RETURN coalesce(n.source_kind,'<null>') AS source_kind, count(*) AS cnt ORDER BY source_kind; MATCH (n:KnowledgeNode) RETURN coalesce(n.quality_tier,'<null>') AS quality_tier, count(*) AS cnt ORDER BY quality_tier;"
    }
    [System.IO.File]::WriteAllText($OutputFile, $summary, [System.Text.UTF8Encoding]::new($false))
}

function Export-Neo4jDump([string]$OutputDir) {
    Write-Step "Export Neo4j full dump"
    Ensure-Dir $OutputDir

    $neo4jImage = Invoke-TextCommand {
        docker inspect --format "{{.Config.Image}}" course-recommend-neo4j
    }

    $resolvedOutputDir = (Resolve-Path $OutputDir).Path
    $wasRunning = ((& docker inspect --format "{{.State.Running}}" course-recommend-neo4j | Out-String).Trim()) -eq "true"

    if ($wasRunning) {
        docker stop course-recommend-neo4j | Out-Null
    }

    try {
        & docker run --rm --volumes-from course-recommend-neo4j -v "${resolvedOutputDir}:/backups" $neo4jImage neo4j-admin database dump neo4j --to-path=/backups --overwrite-destination=true
        if ($LASTEXITCODE -ne 0) {
            throw "neo4j-admin database dump failed"
        }
    } finally {
        if ($wasRunning) {
            docker start course-recommend-neo4j | Out-Null
            Wait-ContainerHealthy "course-recommend-neo4j"
            Wait-HttpReady "http://127.0.0.1:8080/readyz"
        }
    }
}

function Export-RedisDump([string]$OutputFile) {
    Write-Step "Export Redis RDB snapshot"
    docker exec course-recommend-redis redis-cli SAVE | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "redis SAVE failed"
    }
    & docker cp "course-recommend-redis:/data/dump.rdb" $OutputFile
    if ($LASTEXITCODE -ne 0) {
        throw "docker cp redis dump failed"
    }
}

function Write-Manifest([string]$RepoRoot, [string]$BundleRoot, [string]$ProjectArchivePath) {
    Write-Step "Write manifest"

    $health = ""
    $ready = ""
    try { $health = (Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:8080/healthz" -TimeoutSec 5).Content } catch {}
    try { $ready = (Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:8080/readyz" -TimeoutSec 5).Content } catch {}

    $dockerPs = Invoke-TextCommand { docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" }
    $gitStatus = Invoke-TextCommand { git -C $RepoRoot status --short }
    $goVersion = ""
    $nodeVersion = ""
    $npmVersion = ""
    try { $goVersion = Invoke-TextCommand { go version } } catch {}
    try { $nodeVersion = Invoke-TextCommand { node --version } } catch {}
    try { $npmVersion = Invoke-TextCommand { npm --version } } catch {}

    $manifest = @{
        createdAt = (Get-Date).ToString("s")
        projectRoot = $RepoRoot
        bundleRoot = $BundleRoot
        projectArchive = $ProjectArchivePath
        healthz = $health
        readyz = $ready
        dockerPs = $dockerPs
        gitStatus = $gitStatus
        versions = @{
            go = $goVersion
            node = $nodeVersion
            npm = $npmVersion
        }
    }

    $manifestPath = Join-Path $BundleRoot "handoff-manifest.json"
    $manifest | ConvertTo-Json -Depth 6 | Set-Content -Encoding UTF8 $manifestPath
}

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if (-not $OutputRoot) {
    $OutputRoot = Join-Path $repoRoot "artifacts\handoff"
}

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$bundleRoot = Join-Path $OutputRoot "mid_version_$timestamp"
$dataRoot = Join-Path $bundleRoot "data"
$mysqlDir = Join-Path $dataRoot "mysql"
$neo4jDir = Join-Path $dataRoot "neo4j"
$redisDir = Join-Path $dataRoot "redis"

Ensure-Dir $bundleRoot
Ensure-Dir $dataRoot
Ensure-Dir $mysqlDir
Ensure-Dir $neo4jDir
Ensure-Dir $redisDir

$projectArchivePath = ""
if (-not $SkipProjectZip) {
    $projectArchivePath = Export-ProjectSnapshot $repoRoot $bundleRoot
}

if (-not $SkipMySQLDump) {
    Export-MySQLDump (Join-Path $mysqlDir "course_recommend.sql")
}

Export-Neo4jSummary (Join-Path $neo4jDir "graph_summary.txt")

if (-not $SkipNeo4jDump) {
    Export-Neo4jDump $neo4jDir
}

if (-not $SkipRedisDump) {
    Export-RedisDump (Join-Path $redisDir "dump.rdb")
}

Write-Manifest $repoRoot $bundleRoot $projectArchivePath

Write-Host ""
Write-Host "Bundle ready: $bundleRoot" -ForegroundColor Green
