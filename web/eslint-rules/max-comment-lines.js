const MAX = 3

export default {
  meta: {
    type: 'suggestion',
    schema: [],
    messages: {
      tooLong: `A comment is at most ${MAX} lines. Shorten it or delete it.`
    }
  },
  create(context) {
    return {
      Program() {
        const blocks = []
        let prev
        for (const comment of context.sourceCode.getAllComments()) {
          const adjacent =
            prev?.type === 'Line' &&
            comment.type === 'Line' &&
            comment.loc.start.line === prev.loc.end.line + 1
          if (adjacent) {
            blocks[blocks.length - 1].end = comment.loc.end.line
          } else {
            blocks.push({ node: comment, end: comment.loc.end.line })
          }
          prev = comment
        }
        for (const { node, end } of blocks) {
          if (end - node.loc.start.line + 1 > MAX) {
            context.report({ node, messageId: 'tooLong' })
          }
        }
      }
    }
  }
}
