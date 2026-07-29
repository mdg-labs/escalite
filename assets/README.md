# Escalite branding assets

Canonical logo and app icon sources. Regenerate app/public PNGs after editing SVGs:

```bash
node scripts/generate-branding-assets.mjs
```

| File | Use |
| ---- | --- |
| `escalite_app_icon_w_dark_blue_bg.svg` | App icon (Apple-style squircle, dark blue `#000010` background) — mobile launcher, `apple-touch-icon` |
| `escalite_app_icon_wo_bg.svg` | Logo mark without background — favicon, sidebar, in-app chrome |
| `escalite_app_icon_wo_bg_big.svg` | Larger mark — auth/login hero |
| `escalite_app_icon_wo_bg_big.png` | Raster fallback (2000×2000) |

Web and status-page serve SVGs from `public/branding/`. Mobile uses generated PNGs in `apps/mobile/assets/`.
