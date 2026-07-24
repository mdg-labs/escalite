export type Theme = 'dark' | 'light'

/** Dark mode is the product default for on-call tooling. */
export const DEFAULT_THEME: Theme = 'dark'

export const THEME_STORAGE_KEY = 'escalite-theme'

export function resolveTheme(stored: string | null | undefined): Theme {
  if (stored === 'light' || stored === 'dark') {
    return stored
  }
  return DEFAULT_THEME
}

/** Apply theme by toggling the `dark` class on the html element. */
export function applyThemeToHtml(html: HTMLElement, theme: Theme): void {
  if (theme === 'dark') {
    html.classList.add('dark')
  } else {
    html.classList.remove('dark')
  }
  html.dataset.theme = theme
}

export function getThemeFromHtml(html: HTMLElement): Theme {
  return html.classList.contains('dark') ? 'dark' : 'light'
}

export function toggleTheme(current: Theme): Theme {
  return current === 'dark' ? 'light' : 'dark'
}

function readStoredTheme(): Theme | null {
  if (typeof window === 'undefined') {
    return null
  }
  try {
    return resolveTheme(window.localStorage.getItem(THEME_STORAGE_KEY))
  } catch {
    return null
  }
}

/** Initialize theme on the document root; defaults to dark when unset. */
export function initTheme(): Theme {
  if (typeof document === 'undefined') {
    return DEFAULT_THEME
  }

  const theme = readStoredTheme() ?? DEFAULT_THEME
  applyThemeToHtml(document.documentElement, theme)
  return theme
}

export function setTheme(theme: Theme): void {
  if (typeof document === 'undefined') {
    return
  }

  applyThemeToHtml(document.documentElement, theme)

  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // Ignore storage failures (private mode, blocked storage, etc.).
  }
}

export function getTheme(): Theme {
  if (typeof document === 'undefined') {
    return DEFAULT_THEME
  }

  return getThemeFromHtml(document.documentElement)
}
