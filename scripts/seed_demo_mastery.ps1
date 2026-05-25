param(
  [long[]]$StudentIds = @(),
  [string]$ApiBase = 'http://127.0.0.1:8080/api',
  [int]$SeedCount = 14,
  [int]$NeighborLimit = 6,
  [int]$MaxNodes = 36,
  [string]$MysqlContainer = 'course-recommend-mysql',
  [string]$RedisContainer = 'course-recommend-redis',
  [string]$MysqlUser = 'app',
  [string]$MysqlPassword = 'app_dev_pass',
  [string]$MysqlDatabase = 'course_recommend'
)

$ErrorActionPreference = 'Stop'

function Escape-SqlLiteral {
  param([string]$Value)
  $safeValue = if ($null -eq $Value) { '' } else { [string]$Value }
  return [string]::Concat("'", $safeValue.Replace("'", "''"), "'")
}

function Get-GraphPayload {
  param([string]$Url)
  $response = Invoke-RestMethod -Uri $Url -TimeoutSec 20
  if ($null -eq $response -or $null -eq $response.data) {
    throw "Invalid graph response from $Url"
  }
  return $response.data
}

function Get-ActiveStudentIds {
  $query = "SELECT id FROM users WHERE role='student' AND status='active' AND class_id IS NOT NULL ORDER BY id;"
  $raw = & docker exec $MysqlContainer mysql -N -B "-u$MysqlUser" "-p$MysqlPassword" -D $MysqlDatabase -e $query
  return @($raw | Where-Object { $_ -match '^\d+$' } | ForEach-Object { [int64]$_ })
}

function Get-DisplayName {
  param([object]$Node)
  foreach ($key in @('canonicalName', 'name', 'routeKnowledgeId', 'id')) {
    $value = [string]$Node.$key
    if (-not [string]::IsNullOrWhiteSpace($value)) {
      return $value.Trim()
    }
  }
  return 'unknown'
}

function Build-DemoRecords {
  param([object[]]$Nodes)

  $seedScores = @(96, 89, 83, 76, 68, 61, 54, 47, 39, 28, 19, 92, 73, 57)
  $neighborScores = @(91, 85, 78, 71, 64, 58, 51, 44, 36, 27, 18, 82, 67, 49, 33, 22)
  $records = New-Object System.Collections.Generic.List[object]
  $seedIndex = 0
  $neighborIndex = 0

  foreach ($node in $Nodes) {
    $isSeed = [bool]$node.isSeed
    if ($isSeed) {
      $slot = $seedIndex
      $seedIndex += 1
    } else {
      $slot = $neighborIndex
      $neighborIndex += 1
    }
    $score = if ($isSeed) {
      $seedScores[$slot % $seedScores.Count]
    } else {
      $neighborScores[$slot % $neighborScores.Count]
    }

    $answeredCount = if ($score -ge 80) {
      18 - ($slot % 4)
    } elseif ($score -ge 60) {
      12 - ($slot % 3)
    } elseif ($score -ge 30) {
      8 - ($slot % 3)
    } else {
      4 + ($slot % 2)
    }

    $correctCount = [math]::Round($answeredCount * ($score / 100.0))
    if ($score -gt 0) {
      $correctCount = [math]::Max(1, [int]$correctCount)
    }
    $correctCount = [math]::Min($answeredCount, [int]$correctCount)
    $correctRate = if ($answeredCount -gt 0) {
      [math]::Round($correctCount / $answeredCount, 4)
    } else {
      0
    }

    $status = if ($score -ge 80) {
      'solid'
    } elseif ($score -gt 0) {
      'progressing'
    } else {
      'not_started'
    }

    $lastAnsweredAt = (Get-Date).AddHours(-1 * ($slot + 2)).ToUniversalTime().ToString('yyyy-MM-dd HH:mm:ss')
    $records.Add([pscustomobject]@{
        KnowledgeID = [string]$node.knowledgeId
        CanonicalName = [string]$node.canonicalName
        CourseTitle = [string]$node.courseTitle
        IsSeed = $isSeed
        MasteryScore = [math]::Round($score, 2)
        AnsweredCount = [int]$answeredCount
        CorrectCount = [int]$correctCount
        CorrectRate = [math]::Round($correctRate, 4)
        Status = $status
        LastAnsweredAt = $lastAnsweredAt
      }) | Out-Null
  }

  return $records
}

function Write-MySqlProgress {
  param(
    [long]$StudentId,
    [System.Collections.Generic.List[object]]$Records
  )

  if ($Records.Count -eq 0) {
    return
  }

  $values = foreach ($record in $Records) {
    $knowledgeId = Escape-SqlLiteral $record.KnowledgeID
    $canonicalName = Escape-SqlLiteral $record.CanonicalName
    $status = Escape-SqlLiteral $record.Status
    $lastAnsweredAt = Escape-SqlLiteral $record.LastAnsweredAt
    "($StudentId, $knowledgeId, $canonicalName, $($record.MasteryScore), $status, $($record.CorrectRate), $($record.AnsweredCount), $lastAnsweredAt, NOW(), NOW())"
  }

  $sql = @"
INSERT INTO learning_progress (
  student_id,
  knowledge_id,
  canonical_name,
  mastery_score,
  status,
  correct_rate,
  answered_count,
  last_answered_at,
  created_at,
  updated_at
)
VALUES
$($values -join ",`n")
ON DUPLICATE KEY UPDATE
  canonical_name = VALUES(canonical_name),
  mastery_score = VALUES(mastery_score),
  status = VALUES(status),
  correct_rate = VALUES(correct_rate),
  answered_count = VALUES(answered_count),
  last_answered_at = VALUES(last_answered_at),
  updated_at = NOW();
"@

  & docker exec $MysqlContainer mysql "-u$MysqlUser" "-p$MysqlPassword" -D $MysqlDatabase -e $sql | Out-Null
}

function Reset-RedisProgress {
  param([long]$StudentId)

  $baseKeys = @(
    "uid:$StudentId:kp:v1",
    "uid:$StudentId:kp:answered:v1",
    "uid:$StudentId:kp:correct:v1",
    "uid:$StudentId:kp:last:v1",
    "uid:$StudentId:competency:v1"
  )

  & docker exec $RedisContainer redis-cli DEL @baseKeys | Out-Null

  $clearRecommendCommand = "for key in `$(redis-cli --scan --pattern 'uid:$StudentId:recommend:*'); do redis-cli DEL `"`$key`" >/dev/null; done"
  & docker exec $RedisContainer sh -lc $clearRecommendCommand | Out-Null
}

function Write-RedisProgress {
  param(
    [long]$StudentId,
    [System.Collections.Generic.List[object]]$Records
  )

  if ($Records.Count -eq 0) {
    return
  }

  Reset-RedisProgress -StudentId $StudentId

  foreach ($record in $Records) {
    & docker exec $RedisContainer redis-cli HSET "uid:$StudentId:kp:v1" $record.KnowledgeID ('{0:0.00}' -f $record.MasteryScore) | Out-Null
    & docker exec $RedisContainer redis-cli HSET "uid:$StudentId:kp:answered:v1" $record.KnowledgeID ([string]$record.AnsweredCount) | Out-Null
    & docker exec $RedisContainer redis-cli HSET "uid:$StudentId:kp:correct:v1" $record.KnowledgeID ([string]$record.CorrectCount) | Out-Null
    & docker exec $RedisContainer redis-cli HSET "uid:$StudentId:kp:last:v1" $record.KnowledgeID ([string]$record.LastAnsweredAt) | Out-Null
  }
}

$seedPayload = Get-GraphPayload -Url "$ApiBase/graph/seeds?count=$SeedCount"
$seedNodes = @($seedPayload.nodes)
if ($seedNodes.Count -eq 0) {
  throw "No graph seeds returned from $ApiBase"
}

if ($StudentIds.Count -eq 0) {
  $StudentIds = @(Get-ActiveStudentIds)
}

if ($StudentIds.Count -eq 0) {
  throw 'No active student accounts found.'
}

$selectedNodes = New-Object System.Collections.Generic.List[object]
$seenKnowledgeIds = New-Object 'System.Collections.Generic.HashSet[string]'

foreach ($seed in $seedNodes) {
  $knowledgeId = if (-not [string]::IsNullOrWhiteSpace([string]$seed.routeKnowledgeId)) {
    [string]$seed.routeKnowledgeId
  } else {
    [string]$seed.id
  }
  if ([string]::IsNullOrWhiteSpace($knowledgeId)) {
    continue
  }
  if ($seenKnowledgeIds.Add($knowledgeId)) {
    $selectedNodes.Add([pscustomobject]@{
        knowledgeId = $knowledgeId
        canonicalName = Get-DisplayName -Node $seed
        courseTitle = [string]$seed.chapterTitle
        isSeed = $true
      }) | Out-Null
  }
}

foreach ($seed in $seedNodes) {
  if ($selectedNodes.Count -ge $MaxNodes) {
    break
  }

  $seedId = if (-not [string]::IsNullOrWhiteSpace([string]$seed.id)) {
    [string]$seed.id
  } else {
    [string]$seed.routeKnowledgeId
  }
  if ([string]::IsNullOrWhiteSpace($seedId)) {
    continue
  }

  $neighborPayload = Get-GraphPayload -Url "$ApiBase/graph/neighbors?nodeId=$seedId&depth=1&direction=both&limit=$NeighborLimit"
  foreach ($neighbor in @($neighborPayload.nodes)) {
    if ($selectedNodes.Count -ge $MaxNodes) {
      break
    }

    $knowledgeId = if (-not [string]::IsNullOrWhiteSpace([string]$neighbor.routeKnowledgeId)) {
      [string]$neighbor.routeKnowledgeId
    } else {
      [string]$neighbor.id
    }
    if ([string]::IsNullOrWhiteSpace($knowledgeId) -or $knowledgeId -eq $seedId) {
      continue
    }

    if ($seenKnowledgeIds.Add($knowledgeId)) {
      $selectedNodes.Add([pscustomobject]@{
          knowledgeId = $knowledgeId
          canonicalName = Get-DisplayName -Node $neighbor
          courseTitle = [string]$neighbor.chapterTitle
          isSeed = $false
        }) | Out-Null
    }
  }
}

$records = Build-DemoRecords -Nodes $selectedNodes

foreach ($studentId in $StudentIds) {
  Write-MySqlProgress -StudentId $studentId -Records $records
  Write-RedisProgress -StudentId $studentId -Records $records
}

Write-Host ''
Write-Host "Seeded demo mastery for students: $($StudentIds -join ', ')"
Write-Host "Knowledge nodes: $($records.Count)"
Write-Host ''
$records |
  Select-Object -First 16 KnowledgeID, CanonicalName, CourseTitle, IsSeed, MasteryScore, AnsweredCount |
  Format-Table -AutoSize
