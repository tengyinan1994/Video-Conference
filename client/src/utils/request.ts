import { getApiBaseUrl } from '@/utils/apiBase'
import { clearAuth, getToken } from '@/stores/auth'

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
  timestamp?: number
  traceID?: string
}

export class ApiError extends Error {
  code: number

  constructor(code: number, message: string) {
    super(message)
    this.code = code
    this.name = 'ApiError'
  }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const base = getApiBaseUrl()
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string> | undefined),
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  let res: Response
  try {
    res = await fetch(`${base}${path}`, {
      ...init,
      headers,
    })
  } catch {
    throw new ApiError(
      -1,
      base
        ? `无法连接业务服务（${base}）。请确认 HotGo 已启动且本机网络可达`
        : '无法连接业务服务。请确认开发代理或 HotGo 已启动',
    )
  }

  if (!res.ok) {
    throw new ApiError(res.status, `HTTP ${res.status}`)
  }

  const body = (await res.json()) as ApiResponse<T>
  if (body.code !== 0) {
    // 未授权时清登录态，便于路由回登录页
    if (body.code === 401 || body.code === 61) {
      clearAuth()
    }
    throw new ApiError(body.code, body.message || '请求失败')
  }
  return body.data
}
