param(
  [int]$MaxPerCategory = 5,
  [int]$SinceDays = 14,
  [switch]$UseAI,
  [switch]$IncludeHuggingFace,
  [switch]$IncludeGitHub,
  [switch]$KeepExisting,
  [switch]$ImportNeo4j
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$ScraperDir = Join-Path $Root "pipelines/frontier-scraper"

$sources = @("arxiv")
if ($IncludeHuggingFace) { $sources += "huggingface" }
if ($IncludeGitHub) { $sources += "github" }

$argsList = @(
  "main.py",
  "--source"
) + $sources + @(
  "--max-per-category", "$MaxPerCategory",
  "--since-days", "$SinceDays",
  "--run-txt2neo4j"
)

if (-not $UseAI) {
  $argsList += "--skip-ai"
}

if ($KeepExisting) {
  $argsList += "--keep-existing"
}

if ($ImportNeo4j) {
  if ($env:FRONTIER_IMPORT_NEO4J_CONFIRMED -ne "true") {
    throw "Neo4j import is a database write operation. Set FRONTIER_IMPORT_NEO4J_CONFIRMED=true only after explicit confirmation."
  }
  $argsList += "--import-neo4j"
}

Push-Location $ScraperDir
try {
  python @argsList
} finally {
  Pop-Location
}
