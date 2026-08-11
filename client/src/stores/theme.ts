const THEME_KEY = 'vc.theme'

export type ThemeMode = 'light' | 'dark'

function read(): ThemeMode {
  try {
    const v = localStorage.getItem(THEME_KEY)
    if (v === 'light' || v === 'dark') return v
  } catch {
    // ignore
  }
  if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: light)').matches) {
    return 'light'
  }
  return 'dark'
}

let current: ThemeMode = read()
const listeners = new Set<() => void>()

function emit() {
  listeners.forEach((fn) => fn())
}

export function getTheme(): ThemeMode {
  return current
}

export function isDark(): boolean {
  return current === 'dark'
}

export function applyTheme(mode: ThemeMode = current) {
  current = mode
  document.documentElement.setAttribute('data-theme', mode)
  document.documentElement.style.colorScheme = mode
  try {
    localStorage.setItem(THEME_KEY, mode)
  } catch {
    // ignore
  }
  emit()
}

export function setTheme(mode: ThemeMode) {
  applyTheme(mode)
}

export function toggleTheme() {
  applyTheme(current === 'dark' ? 'light' : 'dark')
}

export function subscribeTheme(fn: () => void) {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

// 启动时立即应用，避免闪白/闪黑
if (typeof document !== 'undefined') {
  applyTheme(current)
}
