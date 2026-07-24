import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const severityTokens = [
  '--severity-critical',
  '--severity-high',
  '--severity-medium',
  '--severity-low',
  '--severity-resolved',
  '--severity-info',
]

test('severity tokens are defined for light and dark themes', () => {
  const css = readFileSync(join(root, 'tokens/severity.css'), 'utf8')

  for (const token of severityTokens) {
    assert.match(css, new RegExp(`${token}:`))
  }

  assert.match(css, /\.dark\s*\{[\s\S]*--severity-critical:/)
})

test('severity colors documented as colorblind-safe in tokens README', () => {
  const readmePath = join(root, 'tokens/README.md')
  assert.equal(existsSync(readmePath), true)

  const readme = readFileSync(readmePath, 'utf8')
  assert.match(readme, /colorblind-safe/i)
  assert.match(readme, /Okabe/i)
  assert.match(readme, /never.*color alone/i)
  assert.match(readme, /deuteranopia|protanopia/i)
})
