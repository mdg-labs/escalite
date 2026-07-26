#!/usr/bin/env node

import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '../..')
const packageJsonPath = join(root, 'apps/mobile/package.json')
const appJsonPath = join(root, 'apps/mobile/app.json')

const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf8'))
const appJson = JSON.parse(readFileSync(appJsonPath, 'utf8'))

const [marketingVersion] = String(packageJson.version).split('-')
appJson.expo.version = marketingVersion

writeFileSync(appJsonPath, `${JSON.stringify(appJson, null, 2)}\n`)
