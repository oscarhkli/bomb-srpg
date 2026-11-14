import { RuleTester } from 'eslint'
import rule from './max-comment-lines.js'

const ruleTester = new RuleTester()
const max2 = [{ max: 2 }]

ruleTester.run('max-comment-lines', rule, {
  valid: [
    { code: '// one\n// two\nconst a = 1', options: max2 },
    { code: '// a\n// b\n\n// c\n// d\nconst a = 1', options: max2 },
    { code: '/* a\n b */\nconst a = 1', options: max2 },
    { code: '// a\n// b\nconst a = 1\n// c\nconst b = 2', options: max2 },
    { code: '// a\n// b\n// c\nconst a = 1', options: [{ max: 3 }] }
  ],
  invalid: [
    {
      code: '// a\n// b\n// c\nconst a = 1',
      options: max2,
      errors: [{ messageId: 'tooLong', data: { max: 2 } }]
    },
    {
      code: '/**\n * a\n * b\n */\nconst a = 1',
      options: max2,
      errors: [{ messageId: 'tooLong' }]
    },
    {
      code: '/* a\n b\n c */\nconst a = 1',
      options: max2,
      errors: [{ messageId: 'tooLong' }]
    },
    {
      code: '// a\n// b\n// c\n// d\nconst a = 1',
      options: [{ max: 3 }],
      errors: [{ messageId: 'tooLong', data: { max: 3 } }]
    }
  ]
})
