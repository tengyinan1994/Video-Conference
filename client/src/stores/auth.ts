const AUTH_KEY = 'vc.auth'

export interface AuthUser {
  id: number
  username: string
  realName?: string
  token: string
  /** 绝对过期时间（unix 秒） */
  expiresAt: number
}

function read(): AuthUser | null {
  try {
    const raw = localStorage.getItem(AUTH_KEY)
    if (!raw) return null
    const u = JSON.parse(raw) as AuthUser & { expires?: number }
    if (!u?.token) return null
    const expiresAt = u.expiresAt || (u.expires && u.expires > 1e10 ? u.expires : 0)
    if (expiresAt > 0 && expiresAt * 1000 < Date.now()) {
      localStorage.removeItem(AUTH_KEY)
      return null
    }
    return { ...u, expiresAt }
  } catch {
    return null
  }
}

let current = read()
const listeners = new Set<() => void>()

function emit() {
  listeners.forEach((fn) => fn())
}

export function getAuth(): AuthUser | null {
  return current
}

export function getToken(): string | null {
  return current?.token ?? null
}

export function isLoggedIn(): boolean {
  return !!current?.token
}

export function setAuth(user: AuthUser) {
  current = user
  localStorage.setItem(AUTH_KEY, JSON.stringify(user))
  emit()
}

export function clearAuth() {
  current = null
  localStorage.removeItem(AUTH_KEY)
  emit()
}

export function displayName(): string {
  if (!current) return ''
  return current.realName?.trim() || current.username
}

export function subscribeAuth(fn: () => void) {
  listeners.add(fn)
  return () => listeners.delete(fn)
}
