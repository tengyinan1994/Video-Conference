import { getApiBaseUrl } from '@/utils/apiBase'

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
  let res: Response
  try {
    res = await fetch(`${base}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        ...(init?.headers ?? {}),
      },
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
    throw new ApiError(body.code, body.message || '请求失败')
  }
  return body.data
}
