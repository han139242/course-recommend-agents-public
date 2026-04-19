param(
    [string]$BundlePath = "",
    [switch]$SkipRedis,
    [switch]$SkipBackend
)

$ErrorActionPreference = "Stop"

function Write-Step([string]$Message) {
    Write-Host ""
    Write-Host "== $Message ==" -ForegroundColor Cyan
}

function Resolve-AbsolutePath([string]$Path) {
    return (Resolve-Path $Path).Path
}

function Resolve-BundleRoot([string]$RepoRoot, [string]$BundlePath) {
    $normalized = ""
    if (-not [string]::IsNullOrWhiteSpace($BundlePath)) {
        $normalized = $BundlePath.Trim()
    }
    if ($normalized -ne "") {
        return Resolve-AbsolutePath (Join-Path $RepoRoot $normalized)
    }

    $handoffRoot = Join-Path $RepoRoot "handoff"
    if (-not (Test-Path $handoffRoot)) {
        throw "handoff directory not found: $handoffRoot"
    }

    $latestBundle = Get-ChildItem -Path $handoffRoot -Directory -Filter "mid_version_*" |
        Sort-Object Name -Descending |
        Select-Object -First 1

    if (-not $latestBundle) {
        throw "No handoff bundle found under $handoffRoot"
    }

    return $latestBundle.FullName
}

function Invoke-TextCommand([scriptblock]$ScriptBlock) {
    $output = & $ScriptBlock
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code $LASTEXITCODE"
    }
    return ($output | Out-String).Trim()
}

function Read-ExpectedGraphSummary([string]$SummaryPath) {
    if (-not (Test-Path $SummaryPath)) {
        return @{}
    }

    $lines = Get-Content $SummaryPath |
        ForEach-Object { $_.Trim() } |
        Where-Object { $_ -ne "" }

    $result = @{}
    for ($index = 0; $index -lt ($lines.Count - 1); $index++) {
        if ($lines[$index] -eq "nodes" -and $lines[$index + 1] -match "^\d+$") {
            $result["nodes"] = [int]$lines[$index + 1]
        }
        if ($lines[$index] -eq "relations" -and $lines[$index + 1] -match "^\d+$") {
            $result["relations"] = [int]$lines[$index + 1]
        }
    }

    return $result
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

function Wait-HttpReady([string]$Url, [int]$TimeoutSeconds = 120) {
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

function Ensure-File([string]$Path) {
    if (-not (Test-Path $Path)) {
        throw "Required file not found: $Path"
    }
}

$repoRoot = Resolve-AbsolutePath (Join-Path $PSScriptRoot "..")
$resolvedBundlePath = Resolve-BundleRoot $repoRoot $BundlePath
$composeFile = Join-Path $repoRoot "docker\neo4j\docker-compose.yml"
$composeEnv = Join-Path $repoRoot "docker\neo4j\.env"
$mysqlDump = Join-Path $resolvedBundlePath "data\mysql\course_recommend.sql"
$neo4jDir = Join-Path $resolvedBundlePath "data\neo4j"
$neo4jDump = Join-Path $neo4jDir "neo4j.dump"
$neo4jSummary = Join-Path $neo4jDir "graph_summary.txt"
$redisDump = Join-Path $resolvedBundlePath "data\redis\dump.rdb"

Ensure-File $composeFile
Ensure-File $composeEnv
Ensure-File $mysqlDump
Ensure-File $neo4jDump

Write-Step "Use snapshot bundle"
Write-Host $resolvedBundlePath -ForegroundColor Yellow

Write-Step "Start MySQL / Redis / Neo4j base services"
docker compose -f $composeFile --env-file $composeEnv up -d mysql redis neo4j
if ($LASTEXITCODE -ne 0) {
    throw "docker compose up for mysql/redis/neo4j failed"
}

Wait-ContainerHealthy "course-recommend-mysql"
Wait-ContainerHealthy "course-recommend-redis"
Wait-ContainerHealthy "course-recommend-neo4j"

Write-Step "Restore MySQL snapshot"
& docker cp $mysqlDump "course-recommend-mysql:/tmp/course_recommend.sql"
if ($LASTEXITCODE -ne 0) {
    throw "docker cp mysql dump failed"
}

& docker exec course-recommend-mysql sh -lc "mysql -uapp -papp_dev_pass --default-character-set=utf8mb4 course_recommend < /tmp/course_recommend.sql"
if ($LASTEXITCODE -ne 0) {
    throw "MySQL restore failed"
}

& docker exec course-recommend-mysql sh -lc "rm -f /tmp/course_recommend.sql" | Out-Null

Write-Step "Restore Neo4j snapshot"
$neo4jImage = (& docker inspect --format "{{.Config.Image}}" course-recommend-neo4j | Out-String).Trim()
if (-not $neo4jImage) {
    $neo4jImage = "neo4j:5.26.24"
}

docker stop course-recommend-neo4j | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Failed to stop course-recommend-neo4j"
}

try {
    & docker run --rm --volumes-from course-recommend-neo4j -v "${neo4jDir}:/backups" $neo4jImage neo4j-admin database load neo4j --from-path=/backups --overwrite-destination=true
    if ($LASTEXITCODE -ne 0) {
        throw "neo4j-admin database load failed"
    }
} finally {
    docker start course-recommend-neo4j | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to restart course-recommend-neo4j"
    }
}

Wait-ContainerHealthy "course-recommend-neo4j"

$expectedGraph = Read-ExpectedGraphSummary $neo4jSummary
if ($expectedGraph.ContainsKey("nodes") -or $expectedGraph.ContainsKey("relations")) {
    Write-Step "Verify Neo4j snapshot"
    if ($expectedGraph.ContainsKey("nodes")) {
        $actualNodes = ((Invoke-TextCommand {
            docker exec course-recommend-neo4j cypher-shell -u neo4j -p 123456789 --format plain "MATCH (n:KnowledgeNode) RETURN count(n) AS cnt"
        }) -split "`r?`n" | Select-Object -Last 1).Trim()
        if ([int]$actualNodes -ne [int]$expectedGraph["nodes"]) {
            throw "KnowledgeNode count mismatch after restore. expected=$($expectedGraph["nodes"]) actual=$actualNodes"
        }
        Write-Host "KnowledgeNode count verified: $actualNodes" -ForegroundColor Green
    }
    if ($expectedGraph.ContainsKey("relations")) {
        $actualRelations = ((Invoke-TextCommand {
            docker exec course-recommend-neo4j cypher-shell -u neo4j -p 123456789 --format plain "MATCH ()-[r]->() RETURN count(r) AS cnt"
        }) -split "`r?`n" | Select-Object -Last 1).Trim()
        if ([int]$actualRelations -ne [int]$expectedGraph["relations"]) {
            throw "Relation count mismatch after restore. expected=$($expectedGraph["relations"]) actual=$actualRelations"
        }
        Write-Host "Relation count verified: $actualRelations" -ForegroundColor Green
    }
}

if ((-not $SkipRedis) -and (Test-Path $redisDump)) {
    Write-Step "Restore Redis snapshot"
    & docker cp $redisDump "course-recommend-redis:/data/dump.rdb"
    if ($LASTEXITCODE -ne 0) {
        throw "docker cp redis dump failed"
    }

    & docker exec course-recommend-redis redis-cli SHUTDOWN NOSAVE
    docker start course-recommend-redis | Out-Null
    Wait-ContainerHealthy "course-recommend-redis"
}

if (-not $SkipBackend) {
    Write-Step "Start backend container"
    docker compose -f $composeFile --env-file $composeEnv up -d backend
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up for backend failed"
    }
    Wait-HttpReady "http://127.0.0.1:8080/readyz"
}

Write-Host ""
Write-Host "Snapshot restore completed." -ForegroundColor Green
Write-Host "Bundle: $resolvedBundlePath"
if (-not $SkipBackend) {
    Write-Host "App URL: http://127.0.0.1:8080"
    Write-Host "Neo4j Browser: http://127.0.0.1:7474"
}
