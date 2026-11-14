export default {
  meta: {
    type: 'suggestion',
    docs: {
      description: 'Limit the number of lines in a comment block'
    },
    schema: [
      {
        type: 'object',
        properties: { max: { type: 'integer', minimum: 1 } },
        additionalProperties: false
      }
    ],
    defaultOptions: [{ max: 2 }],
    messages: {
      tooLong: 'A comment is at most {{max}} lines. Shorten it or delete it.'
    }
  },
  create(context) {
    const [{ max }] = context.options
    const { sourceCode } = context
    return {
      Program() {
        const blocks = []
        let prev
        for (const comment of sourceCode.getAllComments()) {
          const { start, end } = comment.loc
          const before = sourceCode.lines[start.line - 1].slice(0, start.column)
          if (before.trim() !== '') {
            prev = undefined
            continue
          }
          if (prev && start.line === prev.loc.end.line + 1) {
            blocks[blocks.length - 1].end = end
          } else {
            blocks.push({ start, end })
          }
          prev = comment
        }
        for (const { start, end } of blocks) {
          if (end.line - start.line + 1 > max) {
            context.report({ loc: { start, end }, messageId: 'tooLong', data: { max } })
          }
        }
      }
    }
  }
}
