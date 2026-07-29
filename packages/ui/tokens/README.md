# Escalite UI tokens

## Dark mode

Theme is controlled by adding or removing the `dark` class on the `<html>` element. `@escalite/ui/theme/dark-mode` exports helpers (`initTheme`, `setTheme`, `getTheme`) that default to **dark** when no preference is stored.

Import global styles in the host app:

```ts
import '@escalite/ui/tokens/globals.css'
```

## Severity colors (colorblind-safe)

Alert and incident severity must **never** rely on color alone. Pair every severity indicator with an icon, label, and/or shape (per doc 05).

The palette in `severity.css` is based on the [Okabe–Ito colorblind-safe set](https://jfly.uni-koeln.de/color/):

| Token | Hue role | Rationale |
| ----- | -------- | --------- |
| `--severity-critical` | Vermillion (`#D55E00`) | High urgency; distinguishable from green success states |
| `--severity-high` | Orange (`#E69F00`) | Elevated; separable from critical vermillion and info blue |
| `--severity-medium` | Blue (`#0072B2`) | Neutral attention; avoids red/green confusion |
| `--severity-low` | Sky blue (`#56B4E9`) | De-emphasized; still readable on dark backgrounds |
| `--severity-resolved` | Bluish green (`#009E73`) | Positive closure without pure green (problematic for deuteranopia) |
| `--severity-info` | Blue (`#0072B2`) | Informational; shared with medium for consistency |

These hues remain distinguishable under deuteranopia and protanopia when combined with icons (triangle, circle, dash patterns) and text labels.

## Brand amber (`--brand`)

Logo amber (`#F5A623`) is a **separate** semantic from `--accent` (neutral hover surface), `--primary` (actions), and `--warning` (caution). Use it sparingly — at most a few distinct placements per surface.

| Token | Role |
| ----- | ---- |
| `--brand` | Logo amber fill, borders, and small highlights |
| `--brand-foreground` | Text/icons on `--brand` fills (dark neutral for contrast) |
| `--brand-muted` | Subdued brand-tinted background (`bg-brand-muted`) |

Tailwind utilities: `bg-brand`, `text-brand`, `text-brand-foreground`, `bg-brand-muted`, `border-brand`, `ring-brand`.

Mobile mirrors these in `apps/mobile/src/theme/tokens/index.ts` (`brand`, `brandForeground`, `brandMuted`).

### Allowed placements

- App header / sidebar chrome — logo mark, active nav indicator, subtle top border
- Focus rings and keyboard focus on non-destructive controls (`ring-brand`)
- Status page hero or marketing shell accents (`bg-brand-muted` backgrounds)
- Loading spinners and non-severity progress indicators
- Empty-state illustration accents and onboarding callouts

### Forbidden placements

- Primary, secondary, or destructive buttons — keep `primary` / `destructive` semantics
- Alert, incident, or on-call severity badges — use `severity-*` variants only
- Warning banners, caution toasts, or “at risk” states — use `--warning`
- Replacing `--accent` hover/selected surfaces in lists, menus, or tables
- Chart series, alert priority coloring, or any data encoding that overlaps severity hues

## Badge variants

Alert status and priority badges use the `severity-*` Badge variants (`severity-critical`, `severity-high`, `severity-low`, `severity-resolved`, `severity-info`) so foreground/background hues always come from this file — never generic `warning` / `success` / `error` theme colors.
