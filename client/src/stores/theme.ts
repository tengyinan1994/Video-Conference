import { getCurrentWindow } from '@tauri-apps/api/window'
import { isTauri } from '@/utils/platform'

const THEME_KEY = 'vc.theme'

export type ThemeMode = 'light' | 'dark'

function read(): ThemeMode {
  try {
    const v = localStorage.getItem(THEME_KEY)
    if (v === 'light' || v === 'dark') return v
  } catch {
    // ignore
  }
  return 'light'
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

function syncNativeTitleBar(mode: ThemeMode) {
  if (!isTauri()) return
  try {
    void getCurrentWindow().setTheme(mode).catch(() => {
      // ignore
    })
  } catch {
    // ignore
  }
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
  syncNativeTitleBar(mode)
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
