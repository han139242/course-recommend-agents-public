import katex from 'katex'

function escapeHtml(text = '') {
  return String(text || '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
}

export function renderWithKatex(content = '') {
  const raw = String(content || '')
  if (!raw) return ''

  const blocks = raw.split(/(\$\$[\s\S]+?\$\$|\$[^$\n]+\$)/g)
  return blocks
    .map((block) => {
      if (block.startsWith('$$') && block.endsWith('$$')) {
        return katex.renderToString(block.slice(2, -2), { displayMode: true, throwOnError: false })
      }
      if (block.startsWith('$') && block.endsWith('$')) {
        return katex.renderToString(block.slice(1, -1), { displayMode: false, throwOnError: false })
      }
      return escapeHtml(block).replace(/\n/g, '<br/>')
    })
    .join('')
}
