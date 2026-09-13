import { invoke } from '@tauri-apps/api/core'
import { getCurrentWindow } from '@tauri-apps/api/window'
import { isTauri } from '@/utils/platform'

const THEME_KEY = 'vc.theme'

export type ThemeMode = 'light' | 'dark'

/** COLORREF `0x00BBGGRR`，对齐大厅页顶渐变（`#f7f8fc` / `#0b1220`） */
const TITLEBAR_COLORREF: Record<ThemeMode, { caption: number; border: number }> = {
  // light #f7f8fc → RGB(247, 248, 252)
  light: { caption: 0x00fcf8f7, border: 0x00fcf8f7 },
  // dark #0b1220 → RGB(11, 18, 32)
  dark: { caption: 0x0020120b, border: 0x0020120b },
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
    const win = getCurrentWindow()
    void win.setTitle('').catch(() => {
      // ignore
    })
    void win.setTheme(mode).catch(() => {
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
