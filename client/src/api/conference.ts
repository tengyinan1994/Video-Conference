import { request } from '@/utils/request'

export interface TokenCreateResult {
  serverUrl: string
  room: string
  title: string
  identity: string
  nickname: string
  token: string
  expiresAt: number
  isHost: boolean
}

export interface MuteAllResult {
  mutedCount: number
}

export interface MeetingItem {
  id: number
  title: string
  roomName: string
  hostId: number
  hostName: string
  startAt: string
  endAt: string
  status: string
  shareCode: string
  shareUrl: string
  isHost: boolean
  tab: string
  attendees?: string[]
}

export interface MeetingShareView {
  title: string
  roomName: string
  hostName: string
  startAt: string
  endAt: string
  status: string
  shareCode: string
  canJoin: boolean
}

export function createToken(payload: { room?: string; nickname: string; shareCode?: string }) {
  return request<TokenCreateResult>('/api/conference/token/create', {
    method: 'POST',
    body: JSON.stringify(payload),
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

export function listMeetings(tab: 'all' | 'ongoing' | 'scheduled' | 'ended' = 'all') {
  return request<{ list: MeetingItem[] }>(`/api/conference/meeting/list?tab=${tab}`, {
    method: 'GET',
  })
}

export function createMeeting(payload: {
  title: string
  hostId?: number
  hostName?: string
  startAt: string
  endAt: string
}) {
  return request<MeetingItem>('/api/conference/meeting/create', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function endMeeting(id: number) {
  return request<Record<string, never>>('/api/conference/meeting/release', {
    method: 'POST',
    body: JSON.stringify({ id }),
  })
}

export function deleteMeeting(id: number) {
  return request<Record<string, never>>('/api/conference/meeting/delete', {
    method: 'POST',
    body: JSON.stringify({ id }),
  })
}

export function updateMeeting(payload: {
  id: number
  title: string
  startAt: string
  endAt: string
}) {
  return request<MeetingItem>('/api/conference/meeting/update', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function shareView(code: string) {
  return request<MeetingShareView>(
    `/api/conference/meeting/shareView?code=${encodeURIComponent(code)}`,
    { method: 'GET' },
  )
}
