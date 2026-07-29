/**
 * Regenerate raster branding assets from canonical SVGs in assets/.
 *
 * Usage: node scripts/generate-branding-assets.mjs
 *
 * Requires: npx @resvg/resvg-js-cli (no install — fetched on demand).
 */
import { spawnSync } from 'node:child_process'
import { copyFileSync, mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..')
const assetsDir = join(repoRoot, 'assets')

const svgSources = {
  squircle: join(assetsDir, 'escalite_app_icon_w_dark_blue_bg.svg'),
  mark: join(assetsDir, 'escalite_app_icon_wo_bg.svg'),
  markBig: join(assetsDir, 'escalite_app_icon_wo_bg_big.svg'),
}

const outputs = [
  {
    svg: svgSources.squircle,
    out: join(repoRoot, 'apps/mobile/assets/icon.png'),
    width: 1024,
    height: 1024,
  },
  {
    svg: svgSources.mark,
    out: join(repoRoot, 'apps/mobile/assets/adaptive-icon.png'),
    width: 1024,
    height: 1024,
  },
  {
    svg: svgSources.squircle,
    out: join(repoRoot, 'apps/web/public/apple-touch-icon.png'),
    width: 180,
    height: 180,
  },
  {
    svg: svgSources.squircle,
    out: join(repoRoot, 'apps/status-page/public/apple-touch-icon.png'),
    width: 180,
    height: 180,
  },
  {
    svg: svgSources.mark,
    out: join(repoRoot, 'apps/web/public/favicon.png'),
    width: 32,
    height: 32,
  },
  {
    svg: svgSources.mark,
    out: join(repoRoot, 'apps/status-page/public/favicon.png'),
    width: 32,
    height: 32,
  },
  {
    svg: svgSources.markBig,
    out: join(repoRoot, 'apps/mobile/assets/logo.png'),
    width: 128,
    height: 128,
  },
]

function resvg(svgPath, outPath, width, height) {
  mkdirSync(dirname(outPath), { recursive: true })
  const result = spawnSync(
    'npx',
    [
      '--yes',
      '@resvg/resvg-js-cli',
      '--fit-width',
      String(width),
      '--fit-height',
      String(height),
      svgPath,
      outPath,
    ],
    { cwd: repoRoot, stdio: 'inherit' },
  )
  if (result.status !== 0) {
    throw new Error(`resvg failed for ${outPath}`)
  }
}

function copySvgBranding(targetPublicDir) {
  mkdirSync(join(targetPublicDir, 'branding'), { recursive: true })
  const brandingDir = join(targetPublicDir, 'branding')
  copyFileSync(svgSources.mark, join(brandingDir, 'logo.svg'))
  copyFileSync(svgSources.markBig, join(brandingDir, 'logo-big.svg'))
  copyFileSync(svgSources.squircle, join(brandingDir, 'app-icon.svg'))
  copyFileSync(svgSources.mark, join(brandingDir, 'favicon.svg'))
}

mkdirSync(join(repoRoot, 'apps/mobile/assets'), { recursive: true })

for (const { svg, out, width, height } of outputs) {
  resvg(svg, out, width, height)
}

copySvgBranding(join(repoRoot, 'apps/web/public'))
copySvgBranding(join(repoRoot, 'apps/status-page/public'))

console.log('Branding assets generated.')
