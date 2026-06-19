import katex from 'katex'

// A small, dependency-free Markdown renderer used to display generated learning
// resources. It supports the subset the resource agents emit: headings, lists
// (nested), ordered lists, code fences, GFM tables, blockquotes, horizontal
// rules, inline bold/italic/code/links and KaTeX math ($inline$ and $$block$$).
//
// All raw text is HTML-escaped and only a fixed set of tags is emitted, so the
// output is safe to use with v-html even though resources are model-generated.

function escapeHtml(text = '') {
  return String(text ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
}

function escapeAttr(text = '') {
  return escapeHtml(text).replaceAll('"', '&quot;').replaceAll("'", '&#39;')
}

function safeUrl(url = '') {
  const trimmed = String(url || '').trim()
  if (/^(javascript|data|vbscript):/i.test(trimmed)) return '#'
  return trimmed
}

function renderMath(expr, displayMode) {
  try {
    return katex.renderToString(expr, { displayMode, throwOnError: false })
  } catch {
    return escapeHtml(expr)
  }
}

// renderInline converts a single line of inline markdown. It first splits out
// KaTeX math spans, renders those, and applies markup to the escaped remainder.
function renderInline(text = '') {
  const parts = String(text || '').split(/(\$\$[\s\S]+?\$\$|\$[^$\n]+\$)/g)
  return parts
    .map((part) => {
      if (part.startsWith('$$') && part.endsWith('$$')) return renderMath(part.slice(2, -2), true)
      if (part.startsWith('$') && part.endsWith('$')) return renderMath(part.slice(1, -1), false)
      return applyInlineMarkup(escapeHtml(part))
    })
    .join('')
}

function applyInlineMarkup(escaped) {
  return escaped
    .replace(/`([^`]+)`/g, (_, code) => `<code>${code}</code>`)
    .replace(
      /!?\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g,
      (_, label, url) =>
        `<a href="${escapeAttr(safeUrl(url))}" target="_blank" rel="noopener noreferrer">${label}</a>`,
    )
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/__([^_]+)__/g, '<strong>$1</strong>')
    .replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
    .replace(/(^|[^_])_([^_\n]+)_/g, '$1<em>$2</em>')
}

function isBlockStart(line, next) {
  if (/^\s*```/.test(line)) return true
  if (/^#{1,6}\s+/.test(line)) return true
  if (/^\s*([-*_])\1{2,}\s*$/.test(line)) return true
  if (/^\s*>\s?/.test(line)) return true
  if (/^\s*([-*+]|\d+\.)\s+/.test(line)) return true
  if (line.includes('|') && next && next.includes('-') && /^\s*\|?[\s:|-]+\|?\s*$/.test(next)) return true
  return false
}

function splitRow(line) {
  return line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((cell) => cell.trim())
}

function renderTable(header, rows) {
  const head = header.map((cell) => `<th>${renderInline(cell)}</th>`).join('')
  const body = rows
    .map((row) => `<tr>${header.map((_, idx) => `<td>${renderInline(row[idx] || '')}</td>`).join('')}</tr>`)
    .join('')
  return `<table class="md-table"><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`
}

// buildList renders consecutive list items into nested <ul>/<ol> based on each
// item's indent. pos.i is advanced as items are consumed.
function buildList(items, pos) {
  const indent = items[pos.i].indent
  const tag = items[pos.i].ordered ? 'ol' : 'ul'
  let html = `<${tag}>`
  while (pos.i < items.length && items[pos.i].indent === indent) {
    const text = items[pos.i].text
    pos.i++
    let child = ''
    if (pos.i < items.length && items[pos.i].indent > indent) {
      child = buildList(items, pos)
    }
    html += `<li>${renderInline(text)}${child}</li>`
  }
  return `${html}</${tag}>`
}

function parseList(lines, start) {
  const items = []
  let i = start
  while (i < lines.length) {
    const match = lines[i].match(/^(\s*)([-*+]|\d+\.)\s+(.*)$/)
    if (!match) {
      if (!lines[i].trim()) break
      if (items.length && /^\s+\S/.test(lines[i])) {
        items[items.length - 1].text += ` ${lines[i].trim()}`
        i++
        continue
      }
      break
    }
    items.push({
      indent: match[1].replace(/\t/g, '  ').length,
      ordered: /\d+\./.test(match[2]),
      text: match[3],
    })
    i++
  }
  return { listHtml: buildList(items, { i: 0 }), next: i }
}

export function renderMarkdown(markdown = '') {
  const lines = String(markdown || '')
    .replace(/\r\n/g, '\n')
    .replace(/\r/g, '\n')
    .split('\n')
  const html = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]

    const fence = line.match(/^\s*```(.*)$/)
    if (fence) {
      const lang = fence[1].trim()
      const buffer = []
      i++
      while (i < lines.length && !/^\s*```/.test(lines[i])) {
        buffer.push(lines[i])
        i++
      }
      i++
      const cls = lang ? ` class="language-${escapeAttr(lang)}"` : ''
      html.push(`<pre class="code-block"><code${cls}>${escapeHtml(buffer.join('\n'))}</code></pre>`)
      continue
    }

    if (!line.trim()) {
      i++
      continue
    }

    const heading = line.match(/^(#{1,6})\s+(.*)$/)
    if (heading) {
      const level = heading[1].length
      html.push(`<h${level}>${renderInline(heading[2].trim())}</h${level}>`)
      i++
      continue
    }

    if (/^\s*([-*_])\1{2,}\s*$/.test(line)) {
      html.push('<hr/>')
      i++
      continue
    }

    if (line.includes('|') && i + 1 < lines.length && lines[i + 1].includes('-') && /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1])) {
      const header = splitRow(line)
      i += 2
      const rows = []
      while (i < lines.length && lines[i].trim() && lines[i].includes('|')) {
        rows.push(splitRow(lines[i]))
        i++
      }
      html.push(renderTable(header, rows))
      continue
    }

    if (/^\s*>\s?/.test(line)) {
      const buffer = []
      while (i < lines.length && /^\s*>\s?/.test(lines[i])) {
        buffer.push(lines[i].replace(/^\s*>\s?/, ''))
        i++
      }
      html.push(`<blockquote>${renderInline(buffer.join(' '))}</blockquote>`)
      continue
    }

    if (/^\s*([-*+]|\d+\.)\s+/.test(line)) {
      const { listHtml, next } = parseList(lines, i)
      html.push(listHtml)
      i = next
      continue
    }

    const buffer = []
    while (i < lines.length && lines[i].trim() && !isBlockStart(lines[i], lines[i + 1])) {
      buffer.push(lines[i])
      i++
    }
    html.push(`<p>${renderInline(buffer.join(' '))}</p>`)
  }

  return html.join('\n')
}

// stripMindShape removes Mermaid mindmap node-shape decorations such as
// `root((text))`, `id[text]`, `id(text)` and `id{{text}}`, keeping inner text.
function stripMindShape(token) {
  const match = token.match(/^[\w-]*\s*[([{]+\s*([^)\]}]*?)\s*[)\]}]+\s*$/)
  if (match && match[1]) return match[1]
  return token
}

function buildMindTree(nodes, pos) {
  const indent = nodes[pos.i].indent
  let html = '<ul>'
  while (pos.i < nodes.length && nodes[pos.i].indent === indent) {
    const text = nodes[pos.i].text
    pos.i++
    let child = ''
    if (pos.i < nodes.length && nodes[pos.i].indent > indent) {
      child = buildMindTree(nodes, pos)
    }
    html += `<li><span class="mindmap-node">${escapeHtml(text)}</span>${child}</li>`
  }
  return `${html}</ul>`
}

// renderMindMap turns Mermaid `mindmap` indentation syntax into a styled nested
// tree. It is dependency-free and tolerant of malformed input.
export function renderMindMap(content = '') {
  const lines = String(content || '')
    .replace(/\r\n/g, '\n')
    .split('\n')
  const nodes = []
  for (const raw of lines) {
    if (!raw.trim()) continue
    if (/^\s*mindmap\s*$/i.test(raw)) continue
    const indent = (raw.match(/^\s*/)[0] || '').replace(/\t/g, '  ').length
    nodes.push({ indent, text: stripMindShape(raw.trim()) })
  }
  if (!nodes.length) return '<p class="mindmap-empty">（暂无思维导图内容）</p>'
  return `<div class="mindmap-tree">${buildMindTree(nodes, { i: 0 })}</div>`
}
