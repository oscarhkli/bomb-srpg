import { RuleTester } from 'eslint'
import rule from './max-comment-lines.js'

const ruleTester = new RuleTester()

ruleTester.run('max-comment-lines', rule, {
  valid: [
    { code: '// one\n// two\n// three\nconst a = 1' },
    { code: '// a\n// b\n// c\n\n// d\n// e\n// f\nconst a = 1' },
    { code: '/* a\n b\n c */\nconst a = 1' },
    { code: '// a\n// b\n// c\nconst a = 1\n// d\nconst b = 2' }
  ],
  invalid: [
    {
      code: '// a\n// b\n// c\n// d\nconst a = 1',
      errors: [{ messageId: 'tooLong', data: { max: 3 } }]
    },
    {
      code: '/**\n * a\n * b\n */\nconst a = 1',
      errors: [{ messageId: 'tooLong' }]
    },
    {
      code: '/* a\n b\n c\n d */\nconst a = 1',
      errors: [{ messageId: 'tooLong' }]
    }
  ]
})
