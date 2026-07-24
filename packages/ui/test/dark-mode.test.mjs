import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import test from 'node:test'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

test('dark mode defaults to dark and toggles via html class', async () => {
  const mod = await import(pathToFileURL(join(root, 'src/theme/dark-mode.ts')).href)

  assert.equal(mod.DEFAULT_THEME, 'dark')
  assert.equal(mod.resolveTheme(null), 'dark')
  assert.equal(mod.resolveTheme('light'), 'light')
  assert.equal(mod.toggleTheme('dark'), 'light')
  assert.equal(mod.toggleTheme('light'), 'dark')

  const html = { classList: new Set(), dataset: {} }
  const classList = {
    add: (name) => html.classList.add(name),
    remove: (name) => html.classList.delete(name),
    contains: (name) => html.classList.has(name),
  }
  const element = { classList, dataset: html.dataset }

  mod.applyThemeToHtml(element, 'dark')
  assert.equal(element.classList.contains('dark'), true)
  assert.equal(mod.getThemeFromHtml(element), 'dark')

  mod.applyThemeToHtml(element, 'light')
  assert.equal(element.classList.contains('dark'), false)
  assert.equal(mod.getThemeFromHtml(element), 'light')
})

test('dark mode source documents html class contract', () => {
  const source = readFileSync(join(root, 'src/theme/dark-mode.ts'), 'utf8')
  assert.match(source, /classList\.add\('dark'\)/)
  assert.match(source, /classList\.remove\('dark'\)/)
})
