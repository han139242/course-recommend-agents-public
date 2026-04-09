# API curl 全量探测（默认 http://127.0.0.1:8080）。登录与 POST 体使用 JSON 文件；Token 用正则提取，避免 PowerShell 对 UTF-8 响应解析失败。
$ErrorActionPreference = "Stop"
$base = if ($env:API_BASE) { $env:API_BASE } else { "http://127.0.0.1:8080" }
$here = Split-Path -Parent $MyInvocation.MyCommand.Path

function Get-AccessToken([string]$json) {
    if ($json -match '"accessToken"\s*:\s*"([^"]+)"') { return $matches[1] }
    return $null
}

function Invoke-Test {
    param([string]$Name, [string[]]$CurlArgs)
    Write-Host "=== $Name ===" -ForegroundColor Cyan
    $out = & curl.exe -s -w "`nHTTP_CODE:%{http_code}" @CurlArgs 2>&1
    Write-Output $out
    Write-Host ""
}

Invoke-Test "GET /healthz" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/healthz")
Invoke-Test "GET /readyz" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/readyz")
Invoke-Test "GET /api/ping" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/ping")

$loginTeacher = & curl.exe -s "$base/api/auth/login" -H "Content-Type: application/json" -d "@$here/login_teacher.json"
Write-Host "=== POST /api/auth/login (teacher) ===" -ForegroundColor Cyan
Write-Output $loginTeacher
$tokT = Get-AccessToken $loginTeacher
Write-Host ""

$loginStudent = & curl.exe -s "$base/api/auth/login" -H "Content-Type: application/json" -d "@$here/login_student.json"
Write-Host "=== POST /api/auth/login (student) ===" -ForegroundColor Cyan
Write-Output $loginStudent
$tokS = Get-AccessToken $loginStudent
Write-Host ""

Invoke-Test "GET /api/auth/me (teacher)" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/auth/me", "-H", "Authorization: Bearer $tokT")
Invoke-Test "GET /api/auth/me (student)" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/auth/me", "-H", "Authorization: Bearer $tokS")

Invoke-Test "POST /api/teacher/classes" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "-X", "POST", "$base/api/teacher/classes", "-H", "Content-Type: application/json", "-H", "Authorization: Bearer $tokT", "-d", "@$here/post_class.json")

Invoke-Test "GET /api/teacher/classes" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/teacher/classes?page=1&pageSize=20", "-H", "Authorization: Bearer $tokT")

Invoke-Test "POST /api/teacher/classes/1/students" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "-X", "POST", "$base/api/teacher/classes/1/students", "-H", "Content-Type: application/json", "-H", "Authorization: Bearer $tokT", "-d", "@$here/post_batch_students.json")

Invoke-Test "GET /api/teacher/classes/1/students" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/teacher/classes/1/students?page=1&pageSize=20", "-H", "Authorization: Bearer $tokT")
Invoke-Test "GET /api/teacher/students/2" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/teacher/students/2", "-H", "Authorization: Bearer $tokT")
Invoke-Test "GET /api/teacher/students/2/knowledge-status" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/teacher/students/2/knowledge-status", "-H", "Authorization: Bearer $tokT")

Invoke-Test "POST /api/teacher/questions/batch" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "-X", "POST", "$base/api/teacher/questions/batch", "-H", "Content-Type: application/json", "-H", "Authorization: Bearer $tokT", "-d", "@$here/post_import_questions.json")

Invoke-Test "GET /api/teacher/questions" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/teacher/questions?page=1&pageSize=20", "-H", "Authorization: Bearer $tokT")
Invoke-Test "GET /api/teacher/questions/Q-DEMO-001" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/teacher/questions/Q-DEMO-001", "-H", "Authorization: Bearer $tokT")

Invoke-Test "GET /api/learning/questions (student)" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/learning/questions?page=1&pageSize=20", "-H", "Authorization: Bearer $tokS")

Invoke-Test "POST /api/learning/answers" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "-X", "POST", "$base/api/learning/answers", "-H", "Content-Type: application/json", "-H", "Authorization: Bearer $tokS", "-d", "@$here/post_answer.json")

Invoke-Test "GET /api/learning/progress" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/learning/progress", "-H", "Authorization: Bearer $tokS")
Invoke-Test "GET /api/competency/profile" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/competency/profile", "-H", "Authorization: Bearer $tokS")
Invoke-Test "GET /api/recommend/path" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/recommend/path?limit=5&scene=study", "-H", "Authorization: Bearer $tokS")
Invoke-Test "GET /api/dashboard/summary" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/dashboard/summary", "-H", "Authorization: Bearer $tokS")

Invoke-Test "GET /api/frontier/briefing" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/frontier/briefing?knowledgeId=002edb92a1990f62")
Invoke-Test "GET /api/graph/subgraph" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/graph/subgraph?root=002edb92a1990f62&limit=12")
Invoke-Test "GET /api/knowledge/:id" @("-s", "-w", "`nHTTP_CODE:%{http_code}", "$base/api/knowledge/002edb92a1990f62")

Write-Host "Done." -ForegroundColor Green
