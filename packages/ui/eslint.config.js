import baseConfig from '@escalite/config/eslint'

/** @type {import('eslint').Linter.Config[]} */
export default [...baseConfig, { ignores: ['dist'] }]
