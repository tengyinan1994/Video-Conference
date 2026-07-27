import { request } from '@/utils/request'

export interface TokenCreateResult {
  serverUrl: string
  room: string
  identity: string
  nickname: string
  token: string
  expiresAt: number
}

export function createToken(room: string, nickname: string) {
  return request<TokenCreateResult>('/api/conference/token/create', {
    method: 'POST',
    body: JSON.stringify({ room, nickname }),
  })
}
