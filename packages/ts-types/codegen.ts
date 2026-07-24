import type { CodegenConfig } from '@graphql-codegen/cli'

const config: CodegenConfig = {
  schema: '../schema/graphql/*.graphql',
  documents: 'graphql/**/*.graphql',
  generates: {
    'src/generated/graphql.ts': {
      plugins: ['typescript', 'typescript-operations', 'typescript-urql'],
      config: {
        withHooks: true,
        gqlImport: 'urql#gql',
        scalars: {
          DateTime: 'string',
        },
      },
    },
  },
}

export default config
