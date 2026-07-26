import baseConfig from '@escalite/config/eslint'

/** @type {import('eslint').Linter.Config[]} */
export default [
  ...baseConfig,
  {
    ignores: ['dist', '.expo', 'babel.config.js', 'metro.config.js'],
  },
  {
    files: ['**/context.tsx'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
]
