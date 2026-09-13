import { invoke } from '@tauri-apps/api/core'
import { getCurrentWindow } from '@tauri-apps/api/window'
import { isTauri } from '@/utils/platform'

const THEME_KEY = 'vc.theme'

export type ThemeMode = 'light' | 'dark'

/** COLORREF `0x00BBGGRR`，对齐应用 chrome（`--vc-bg0` / `--vc-panel-solid`） */
const TITLEBAR_COLORREF: Record<ThemeMode, { caption: number; border: number }> = {
  // light #F3F6FB → RGB(243, 246, 251)
  light: { caption: 0x00fbf6f3, border: 0x00fbf6f3 },
  // dark #111827 → RGB(17, 24, 39)
  dark: { caption: 0x00271811, border: 0x00271811 },
}

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
    const colors = TITLEBAR_COLORREF[mode]
    void invoke('set_titlebar_colors', {
      caption: colors.caption,
      border: colors.border,
    }).catch(() => {
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
