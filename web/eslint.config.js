import js from '@eslint/js'
import tsParser from '@typescript-eslint/parser'
import tsPlugin from '@typescript-eslint/eslint-plugin'
import prettier from 'eslint-config-prettier'
import sonarjs from 'eslint-plugin-sonarjs'

export default [
  js.configs.recommended,
  ...tsPlugin.configs['flat/recommended-type-checked'],
  ...tsPlugin.configs['flat/stylistic-type-checked'],
  {
    files: ['src/**/*.ts'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: './tsconfig.json',
        tsconfigRootDir: import.meta.dirname
      }
    },
    plugins: {
      '@typescript-eslint': tsPlugin,
      sonarjs
    },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/consistent-type-imports': 'error',
      '@typescript-eslint/no-floating-promises': 'warn',
      '@typescript-eslint/no-non-null-assertion': 'error',
      '@typescript-eslint/restrict-template-expressions': ['error', { allowNumber: true }],
      '@typescript-eslint/switch-exhaustiveness-check': 'error',
      'sonarjs/cognitive-complexity': ['error', 15]
    }
  },
  {
    files: ['src/**/*.test.ts', 'src/test/**/*.ts'],
    rules: {
      '@typescript-eslint/no-non-null-assertion': 'off'
    }
  },
  {
    files: ['src/api/**/*.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['**/rendering/*', '**/ui/*', '**/scenes/*'],
              message: 'api sits below the rendering layers'
            }
          ]
        }
      ]
    }
  },
  prettier,
  {
    files: ['src/**/*.ts'],
    rules: {
      curly: ['error', 'all']
    }
  }
]
