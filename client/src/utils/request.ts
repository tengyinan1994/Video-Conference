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
  const base = import.meta.env.VITE_API_BASE_URL ?? ''
  const res = await fetch(`${base}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })

  if (!res.ok) {
    throw new ApiError(res.status, `HTTP ${res.status}`)
  }

  const body = (await res.json()) as ApiResponse<T>
  if (body.code !== 0) {
    throw new ApiError(body.code, body.message || '请求失败')
  }
  return body.data
}
