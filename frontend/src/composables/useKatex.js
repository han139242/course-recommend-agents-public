import katex from 'katex'

/**
 * Render LaTeX in HTML string. Supports:
 * - Display math: $$ ... $$ or \[ ... \]
 * - Inline math:  $ ... $  or \( ... \)
 */
export function renderLatex(html) {
  if (!html) return ''

  // Display math: $$ ... $$
  html = html.replace(/\$\$([\s\S]+?)\$\$/g, (_, tex) => {
    try {
      return katex.renderToString(tex.trim(), { displayMode: true, throwOnError: false })
    } catch { return _ }
  })

  // Display math: \[ ... \]
  html = html.replace(/\\\[([\s\S]+?)\\\]/g, (_, tex) => {
    try {
      return katex.renderToString(tex.trim(), { displayMode: true, throwOnError: false })
    } catch { return _ }
  })

  // Inline math: \( ... \)
  html = html.replace(/\\\(([\s\S]+?)\\\)/g, (_, tex) => {
    try {
      return katex.renderToString(tex.trim(), { displayMode: false, throwOnError: false })
    } catch { return _ }
  })

  // Inline math: $ ... $ (single dollar, not greedy, avoid $$)
  html = html.replace(/(?<!\$)\$(?!\$)(.+?)(?<!\$)\$(?!\$)/g, (_, tex) => {
    try {
      return katex.renderToString(tex.trim(), { displayMode: false, throwOnError: false })
    } catch { return _ }
  })

  return html
}
