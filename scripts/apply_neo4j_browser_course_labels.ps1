param(
  [string]$ContainerName = 'course-recommend-neo4j',
  [string]$User = 'neo4j',
  [string]$Password = '123456789'
)

$ErrorActionPreference = 'Stop'
[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$OutputEncoding = [System.Text.UTF8Encoding]::new($false)

function ConvertTo-CypherLiteral {
  param([string]$Text)

  $builder = [System.Text.StringBuilder]::new()
  foreach ($char in $Text.ToCharArray()) {
    $code = [int][char]$char
    if ($char -eq "'") {
      [void]$builder.Append("\\'")
      continue
    }
    if ($char -eq '\') {
      [void]$builder.Append("\\\\")
      continue
    }
    if ($code -ge 32 -and $code -le 126) {
      [void]$builder.Append($char)
      continue
    }
    [void]$builder.AppendFormat('\u{0:X4}', $code)
  }
  return "'$builder'"
}

$courseRules = @(
  @{
    Label = 'Course_Calculus'
    CourseName = '高等数学'
    Primary = '#4DB8C7'
    Light = '#E3F4F7'
    Ring = '#A0D8E1'
    Equals = @('高等数学')
    Contains = @('微积分', '数学分析')
  },
  @{
    Label = 'Course_LinearAlgebra'
    CourseName = '线性代数'
    Primary = '#5BA3D9'
    Light = '#E5F0FA'
    Ring = '#A5CBE8'
    Equals = @('线性代数', '线性代数基础')
    Contains = @()
  },
  @{
    Label = 'Course_ProbabilityStatistics'
    CourseName = '概率与统计'
    Primary = '#4DB8A5'
    Light = '#E3F5F0'
    Ring = '#A0D8CA'
    Equals = @('概率与统计', '概率统计')
    Contains = @()
  },
  @{
    Label = 'Course_DataStructures'
    CourseName = '数据结构'
    Primary = '#4DB87A'
    Light = '#E3F5EA'
    Ring = '#A0D8B2'
    Equals = @('数据结构')
    Contains = @()
  },
  @{
    Label = 'Course_PythonProgramming'
    CourseName = 'Python 编程'
    Primary = '#4F8ACC'
    Light = '#E5EFFB'
    Ring = '#A6C2EB'
    Equals = @('Python编程', 'Python程序设计')
    Contains = @('Python')
  },
  @{
    Label = 'Course_KnowledgeGraph'
    CourseName = '知识图谱'
    Primary = '#A570E0'
    Light = '#EFE4FB'
    Ring = '#C9A8EF'
    Equals = @('人工智能课程知识图谱', '知识图谱')
    Contains = @('知识图谱')
  },
  @{
    Label = 'Course_DeepLearning'
    CourseName = '深度学习'
    Primary = '#9B7FE6'
    Light = '#EDE8FB'
    Ring = '#C4B3F0'
    Equals = @('深度学习', '人工神经网络及其应用')
    Contains = @('深度学习', '神经网络')
  },
  @{
    Label = 'Course_NLP'
    CourseName = '自然语言处理'
    Primary = '#E0A542'
    Light = '#FCF1DE'
    Ring = '#EFC98A'
    Equals = @('自然语言处理', '自然语言处理及其应用')
    Contains = @('自然语言处理')
  },
  @{
    Label = 'Course_ComputerVision'
    CourseName = '计算机视觉'
    Primary = '#E07272'
    Light = '#FCE6E6'
    Ring = '#EFA5A5'
    Equals = @('计算机视觉')
    Contains = @('视觉')
  },
  @{
    Label = 'Course_ReinforcementLearning'
    CourseName = '强化学习'
    Primary = '#D97757'
    Light = '#FAE8E2'
    Ring = '#E8B19E'
    Equals = @('强化学习')
    Contains = @('强化学习')
  },
  @{
    Label = 'Course_LargeModels'
    CourseName = '大模型'
    Primary = '#E06090'
    Light = '#FCE1EB'
    Ring = '#EFA0BC'
    Equals = @('大模型')
    Contains = @('大模型')
  },
  @{
    Label = 'Course_AIEngineering'
    CourseName = 'AI 工程'
    Primary = '#6D9F5A'
    Light = '#EEF6E9'
    Ring = '#BDD8B1'
    Equals = @('AI工程')
    Contains = @('AI工程')
  },
  @{
    Label = 'Course_AIEthics'
    CourseName = 'AI 伦理'
    Primary = '#C98B3C'
    Light = '#F8EEDC'
    Ring = '#E6C48F'
    Equals = @('AI伦理')
    Contains = @('AI伦理')
  },
  @{
    Label = 'Course_MachineLearning'
    CourseName = '机器学习'
    Primary = '#6C8EEF'
    Light = '#E8EDFC'
    Ring = '#B0C4F5'
    Equals = @('机器学习')
    Contains = @('机器学习')
  },
  @{
    Label = 'Course_AIIntro'
    CourseName = '人工智能导论'
    Primary = '#7A8DA6'
    Light = '#EDF1F5'
    Ring = '#BEC9D5'
    Equals = @(
      '人工智能基础知识骨架',
      '专家系统与机器学习',
      '人工智能在游戏设计中的应用',
      '智能体与多智能体系统',
      '智能计算及其应用',
      '人工智能导论',
      '知识表示',
      '不确定性推理方法',
      '确定性推理方法',
      '搜索求解策略',
      '绪论'
    )
    Contains = @('人工智能')
  }
)

$allCourseLabels = @($courseRules.Label) + 'Course_Other'
$removeLabels = ($allCourseLabels | ForEach-Object { "  n:$_" }) -join ",`n"

$caseLines = foreach ($rule in $courseRules) {
  $parts = @()
  if ($rule.Equals.Count -gt 0) {
    $equalsValues = ($rule.Equals | ForEach-Object { ConvertTo-CypherLiteral $_ }) -join ', '
    $parts += "chapter IN [$equalsValues]"
  }
  foreach ($item in $rule.Contains) {
    $parts += "chapter CONTAINS $(ConvertTo-CypherLiteral $item)"
  }
  "  WHEN $($parts -join ' OR ') THEN '$($rule.Label)'"
}

$nameCaseLines = foreach ($rule in $courseRules) {
  "  WHEN courseLabel = '$($rule.Label)' THEN $(ConvertTo-CypherLiteral $rule.CourseName)"
}

$primaryCaseLines = foreach ($rule in $courseRules) {
  "  WHEN courseLabel = '$($rule.Label)' THEN '$($rule.Primary)'"
}

$lightCaseLines = foreach ($rule in $courseRules) {
  "  WHEN courseLabel = '$($rule.Label)' THEN '$($rule.Light)'"
}

$ringCaseLines = foreach ($rule in $courseRules) {
  "  WHEN courseLabel = '$($rule.Label)' THEN '$($rule.Ring)'"
}

$setLines = ($allCourseLabels | ForEach-Object {
  "FOREACH (_ IN CASE WHEN courseLabel = '$_' THEN [1] ELSE [] END | SET n:$_)"
}) -join "`n"

$relabelCypher = @"
MATCH (n:KnowledgeNode)
REMOVE
$removeLabels
WITH n, trim(coalesce(n.chapter_title, n.chapterTitle, '')) AS chapter
WITH n,
CASE
$($caseLines -join "`n")
  ELSE 'Course_Other'
END AS courseLabel
WITH n, courseLabel,
CASE
$($nameCaseLines -join "`n")
  ELSE '其他课程'
END AS courseName,
CASE
$($primaryCaseLines -join "`n")
  ELSE '#64748B'
END AS primaryColor,
CASE
$($lightCaseLines -join "`n")
  ELSE '#F1F5F9'
END AS lightColor,
CASE
$($ringCaseLines -join "`n")
  ELSE '#CBD5E1'
END AS ringColor
$setLines
SET n.browserCourseLabel = courseLabel,
    n.browserCourseName = courseName,
    n.browserPrimaryColor = primaryColor,
    n.browserLightColor = lightColor,
    n.browserRingColor = ringColor
RETURN count(n) AS updated_nodes;
"@

$summaryCypher = @"
MATCH (n:KnowledgeNode)
UNWIND labels(n) AS label
WITH label, count(*) AS count
WHERE label STARTS WITH 'Course_'
RETURN label, count
ORDER BY label;
"@

Write-Host 'Applying Neo4j Browser course labels...'
$relabelCypher | docker exec -i $ContainerName cypher-shell --format plain -u $User -p $Password

Write-Host ''
Write-Host 'Course label distribution:'
$summaryCypher | docker exec -i $ContainerName cypher-shell --format plain -u $User -p $Password
