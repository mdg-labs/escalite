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
