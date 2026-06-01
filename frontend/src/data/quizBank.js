export const QUIZ_BANK = {
  '视觉 Transformer': [
    {
      question: 'Transformer 中自注意力（Self-Attention）机制的计算复杂度与序列长度（L）呈什么关系？',
      options: ['线性关系 O(L)', '对数关系 O(log L)', '平方关系 O(L²)', '指数关系 O(2^L)'],
      answer: 2
    },
    {
      question: 'Vision Transformer (ViT) 是如何处理图像输入以适配一维 Sequence 格式的？',
      options: [
        '直接将图像像素拉平输入',
        '将图像划分为不重叠的小图块 (Patches) 并进行线性投影',
        '先经过深层 CNN 提取高维特征图作为输入',
        '使用 RNN 逐像素扫描特征'
      ],
      answer: 1
    },
    {
      question: '在海量数据集上预训练时，Vision Transformer (ViT) 相比 CNN 具有什么核心优势？',
      options: [
        '具有极强的局部归纳偏置 (Inductive Bias)',
        '不需要任何数据增强即可快速收敛',
        '缺少局部偏置，在大数据下能学习到更具普适性的全局特征表征',
        '参数量和计算量通常比 CNN 小很多'
      ],
      answer: 2
    }
  ],
  '检索增强生成': [
    {
      question: '检索增强生成（RAG）核心解决了大语言模型（LLM）的什么典型痛点？',
      options: ['模型推理速度慢', '幻觉问题与时效性限制', '模型参数量过大导致部署困难', '无法进行长文本上下文理解'],
      answer: 1
    },
    {
      question: '在 RAG 管道中，将非结构化文档转为向量的核心步骤通常是？',
      options: ['分词 (Tokenization)', '文本分块 (Chunking) 与向量嵌入 (Embedding)', '句法树解析 (Dependency Parsing)', '直接用大模型改写文本'],
      answer: 1
    },
    {
      question: '什么是在检索阶段常用的“混合检索（Hybrid Search）”？',
      options: [
        '结合了本地检索和云端检索的搜索机制',
        '结合了稠密向量检索（语义）与稀疏向量检索（关键词 BM25）的策略',
        '结合了文本检索和图像多模态检索的搜索',
        '结合了人工干预与自动搜索的方法'
      ],
      answer: 1
    }
  ],
  '大语言模型': [
    {
      question: '主流的现代大语言模型（如 GPT、Llama）大多采用以下哪种 Transformer 架构变体？',
      options: ['Encoder-only (如 BERT)', 'Decoder-only (如 GPT)', 'Encoder-Decoder (如 T5)', 'Cross-Attention only'],
      answer: 1
    },
    {
      question: '在 LLM 微调阶段，LoRA (Low-Rank Adaptation) 是通过什么方式实现参数高效微调的？',
      options: [
        '随机冻结模型中 90% 的参数',
        '在冻结的原模型旁边引入低秩分解矩阵，仅训练这些新增的小矩阵',
        '使用更低精度的 FP8 格式训练全量参数',
        '将长文本输入裁剪为更短的切片'
      ],
      answer: 1
    },
    {
      question: '大语言模型在生成文本时，Temperature（温度）参数主要用于调节什么？',
      options: ['模型的推理速度', '模型输出的确定性与创造力（随机度）', '模型的显存开销', '上下文的最大支持长度'],
      answer: 1
    }
  ]
}

export const DEFAULT_QUIZ = [
  {
    question: '在构建人工智能与深度学习系统时，为了使神经网络能够拟合复杂的非线性函数，必须引入以下哪个结构？',
    options: ['权重衰减 (Weight Decay)', '激活函数 (Activation Function)', '全连接层 (Dense Layer)', '批量归一化 (Batch Normalization)'],
    answer: 1
  },
  {
    question: '在机器学习中，为了防止模型在训练集上过度拟合（Overfitting）导致泛化能力下降，常用的策略是？',
    options: ['增大训练轮数 (Epochs)', '引入正则化 (Regularization) 或 Dropout', '删除验证集', '增加模型的参数复杂度'],
    answer: 1
  },
  {
    question: '对于科学探索和专业知识学习，构建分层展开的“知识图谱（Knowledge Graph）”主要为了提供什么样的体验？',
    options: ['消除一切学习困难', '实现高维概念的逐层探索与跨课程关联理解', '替代传统的教科书和纸质考试', '单纯展示酷炫的动画效果'],
    answer: 1
  }
]

export function getQuizForNode(nodeId) {
  const normalized = String(nodeId || '').trim()
  if (QUIZ_BANK[normalized]) {
    return QUIZ_BANK[normalized]
  }

  // Try partial match
  const matchedKey = Object.keys(QUIZ_BANK).find(key => normalized.includes(key) || key.includes(normalized))
  if (matchedKey) {
    return QUIZ_BANK[matchedKey]
  }

  return DEFAULT_QUIZ
}
