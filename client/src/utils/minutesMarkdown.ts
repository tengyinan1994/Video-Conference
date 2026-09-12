import MarkdownIt from 'markdown-it'

// html: false —— 纪要正文由大模型生成，禁用内联 HTML，避免正文里混入标签
// breaks: true —— 兼容历史纯文本纪要（当时靠 white-space: pre-wrap 换行）
const md = new MarkdownIt({
  html: false,
  linkify: false,
  breaks: true,
})

/** 把会议纪要 Markdown 渲染成 HTML；空内容返回空串。 */
export function renderMinutesMarkdown(source?: string | null): string {
  const text = (source || '').trim()
  if (!text) return ''
  return decorateTaskItems(md.render(text))
}

/** markdown-it 核心不含 GFM 待办语法，这里把「[ ] 动作」渲染成方框占位。 */
function decorateTaskItems(html: string): string {
  return html.replace(
    /<li>\[([ xX])\]\s*/g,
    (_match, mark: string) =>
      `<li class="md-task"><span class="md-task-box${mark.trim() ? ' is-done' : ''}"></span>`,
  )
}
