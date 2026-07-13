param(
  [string]$Container = "course-recommend-mysql"
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")

$containerEnv = docker inspect $Container --format "{{range .Config.Env}}{{println .}}{{end}}"
$mysqlUser = (($containerEnv | Select-String '^MYSQL_USER=' | Select-Object -First 1).Line -replace '^MYSQL_USER=', '')
$mysqlPassword = (($containerEnv | Select-String '^MYSQL_PASSWORD=' | Select-Object -First 1).Line -replace '^MYSQL_PASSWORD=', '')
$mysqlDatabase = (($containerEnv | Select-String '^MYSQL_DATABASE=' | Select-Object -First 1).Line -replace '^MYSQL_DATABASE=', '')
if (-not $mysqlUser) { $mysqlUser = "app" }
if (-not $mysqlPassword) { $mysqlPassword = "app_dev_pass" }
if (-not $mysqlDatabase) { $mysqlDatabase = "course_recommend" }

$migrations = @(
  @{ Table = "student_goal_profiles"; File = "000007_create_student_study_plan.sql" },
  @{ Table = "student_learning_profiles"; File = "000008_create_learning_profiles.sql" },
  @{ Table = "resource_generation_jobs"; File = "000009_create_generated_resources.sql" },
  @{ Table = "resource_feedbacks"; File = "000010_create_resource_feedbacks.sql" },
  @{ Table = "learning_events"; File = "000011_create_learning_events_and_resource_push_items.sql" }
)

foreach ($migration in $migrations) {
  $query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='$mysqlDatabase' AND table_name='$($migration.Table)';"
  $exists = $query | docker exec -i $Container mysql -N -s "-u$mysqlUser" "-p$mysqlPassword" $mysqlDatabase
  if (($exists | Select-Object -Last 1).Trim() -eq "1") {
    Write-Host "Migration $($migration.File) already applied."
    continue
  }

  $path = Join-Path $Root "backend/migrations/$($migration.File)"
  Get-Content -Raw -Encoding UTF8 $path |
    docker exec -i $Container mysql "-u$mysqlUser" "-p$mysqlPassword" $mysqlDatabase
  if ($LASTEXITCODE -ne 0) {
    throw "Migration failed: $($migration.File)"
  }
  Write-Host "Applied migration $($migration.File)."
}
