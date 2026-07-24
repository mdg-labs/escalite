import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const generatedPath = join(root, 'src/generated/graphql.ts')

test('codegen output exists with urql hooks', () => {
  assert.equal(existsSync(generatedPath), true, 'expected generated graphql.ts')

  const source = readFileSync(generatedPath, 'utf8')
  assert.match(source, /export type MeQuery/)
  assert.match(source, /export function useMeQuery/)
  assert.match(source, /export function useLoginMutation/)
})
