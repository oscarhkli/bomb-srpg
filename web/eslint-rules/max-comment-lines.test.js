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
    { code: '// a\n// b\n// c\nconst a = 1', options: [{ max: 3 }] },
    { code: '// a\n// b\nconst a = 1', options: [] },
    { code: 'const a = 1 // x\n// b\n// c\nconst b = 2', options: max2 }
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
    },
    {
      code: '// a\n// b\n// c\nconst a = 1',
      options: [],
      errors: [{ messageId: 'tooLong', data: { max: 2 } }]
    },
    {
      code: 'const a = 1 // x\n// b\n// c\n// d\nconst b = 2',
      options: max2,
      errors: [{ messageId: 'tooLong', line: 2, endLine: 4 }]
    },
    {
      code: '// a\n/* b */\n// c\nconst a = 1',
      options: max2,
      errors: [{ messageId: 'tooLong', line: 1, endLine: 3 }]
    },
    {
      code: '/* a\n b */\n// c\n// d\nconst a = 1',
      options: max2,
      errors: [{ messageId: 'tooLong', line: 1, endLine: 4 }]
    },
    {
      code: '// a\r\n// b\r\n// c\r\nconst a = 1',
      options: max2,
      errors: [{ messageId: 'tooLong', line: 1, endLine: 3 }]
    },
    {
      code: 'function f() {\n    // a\n    // b\n    // c\n    return 1\n}',
      options: max2,
      errors: [{ messageId: 'tooLong', line: 2, endLine: 4 }]
    }
  ]
})
