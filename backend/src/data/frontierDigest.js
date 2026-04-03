const frontierDigest = {
  梯度下降: {
    knowledge: '梯度下降',
    papers: [
      {
        title: 'Adaptive Gradient Methods Revisited',
        venue: 'ICLR',
        year: 2024,
        summary: '重新审视自适应优化方法在训练稳定性与泛化表现之间的权衡。',
        relation: '帮助学生理解为什么“梯度下降”会进一步演化为 Adam、AdaFactor 等优化家族。',
      },
    ],
    frontierTopics: ['优化器设计', '训练稳定性', '学习率调度'],
  },
  卷积神经网络: {
    knowledge: '卷积神经网络',
    papers: [
      {
        title: 'Visual Representation Learning in the Foundation Model Era',
        venue: 'CVPR',
        year: 2024,
        summary: '讨论卷积结构与视觉基础模型之间的关系，以及视觉表示学习的新范式。',
        relation: '帮助学生把课内 CNN 基础迁移到视觉大模型与多模态任务。',
      },
    ],
    frontierTopics: ['视觉基础模型', '表征学习', '多模态视觉'],
  },
  检索增强生成: {
    knowledge: '检索增强生成',
    papers: [
      {
        title: 'RAG Systems: Retrieval, Grounding and Tool Use',
        venue: 'arXiv',
        year: 2025,
        summary: '总结 RAG 系统在检索、引用溯源和工具调用上的系统设计趋势。',
        relation: '帮助学生将课内知识库问答、向量检索与大模型应用实践连接起来。',
      },
    ],
    frontierTopics: ['引用溯源', 'Agent 工作流', '知识库增强'],
  },
  知识图谱: {
    knowledge: '知识图谱',
    papers: [
      {
        title: 'Knowledge Graph Reasoning with Large Language Models',
        venue: 'ACL Findings',
        year: 2024,
        summary: '探索大语言模型与知识图谱推理协同工作的最新路径。',
        relation: '帮助学生理解课内知识图谱如何与问答、推理和智能体结合。',
      },
    ],
    frontierTopics: ['图谱推理', 'LLM + KG', '可解释问答'],
  },
}

function getFrontierBriefing(knowledgeName = '') {
  const normalized = String(knowledgeName || '').trim()

  if (!normalized) {
    return {
      knowledge: '',
      ready: false,
      message: '请先指定知识点，再查看对应的课后扩展内容。',
      papers: [],
      frontierTopics: [],
    }
  }

  const matched = frontierDigest[normalized]

  if (!matched) {
    return {
      knowledge: normalized,
      ready: false,
      message: '当前知识点的前沿扩展条目尚未补齐，后续会通过论文抓取与知识抽取链路持续更新。',
      papers: [],
      frontierTopics: [],
    }
  }

  return {
    ...matched,
    ready: true,
    message: '已为该知识点准备第一版课后扩展条目，后续可切换为自动抓取与定期更新。',
  }
}

module.exports = {
  getFrontierBriefing,
}
