import { request } from '@/utils/request'

export interface TokenCreateResult {
  serverUrl: string
  room: string
  identity: string
  nickname: string
  token: string
  expiresAt: number
  isHost: boolean
}

export interface MuteAllResult {
  mutedCount: number
}

export function createToken(room: string, nickname: string) {
  return request<TokenCreateResult>('/api/conference/token/create', {
    method: 'POST',
    body: JSON.stringify({ room, nickname }),
  })
}

export function kickParticipant(room: string, targetIdentity: string, requesterIdentity: string) {
  return request<Record<string, never>>('/api/conference/room/kick', {
    method: 'POST',
    body: JSON.stringify({ room, targetIdentity, requesterIdentity }),
  })
}

export function muteAllParticipants(room: string, requesterIdentity: string) {
  return request<MuteAllResult>('/api/conference/room/muteAll', {
    method: 'POST',
    body: JSON.stringify({ room, requesterIdentity }),
  })
}

export function claimHost(room: string, requesterIdentity: string) {
  return request<{ isHost: boolean }>('/api/conference/room/claimHost', {
    method: 'POST',
    body: JSON.stringify({ room, requesterIdentity }),
  })
}
