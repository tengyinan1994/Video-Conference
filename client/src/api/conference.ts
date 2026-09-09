import { request, ApiError } from '@/utils/request'
import { getApiBaseUrl } from '@/utils/apiBase'
import { getToken } from '@/stores/auth'

export interface TokenCreateResult {
  serverUrl: string
  room: string
  title: string
  identity: string
  nickname: string
  token: string
  expiresAt: number
  isHost: boolean
  recordEnabled?: boolean
  recordingActive?: boolean
  /** 会议预定开始时间；房间内「已进行」时长以此为起点累计 */
  startAt?: string
  /** 实际开始时间（首个参会者提前入会时记录）；为空则计时以 startAt 为起点 */
  actualStartAt?: string
}

export interface MuteAllResult {
  mutedCount: number
}

export interface UnmuteAllResult {
  unmutedCount: number
}

export interface MeetingItem {
  id: number
  title: string
  roomName: string
  hostId: number
  hostName: string
  startAt: string
  actualStartAt?: string
  endAt: string
  status: string
  shareCode: string
  shareUrl: string
  isHost: boolean
  tab: string
  attendees?: string[]
  recordEnabled?: boolean
  /** 会议类型ID，0=未分类 */
  typeId?: number
  /** 会议类型名称，未分类为空 */
  typeName?: string
  recordings?: RecordingSegment[]
}

export interface MeetingTypeOption {
  id: number
  name: string
}

export interface RecordingSegment {
  id: number
  meetingId: number
  roomName: string
  egressId: string
  seq: number
  status: string
  objectKey: string
  fileSize: number
  startedAt?: string
  endedAt?: string
  errorMsg?: string
  playUrl?: string
  downloadUrl?: string
}

export interface RecordingStatus {
  active: boolean
  segments: RecordingSegment[]
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

export function unmuteAllParticipants(room: string, requesterIdentity: string) {
  return request<UnmuteAllResult>('/api/conference/room/unmuteAll', {
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

/** 管理端维护的会议类型选项（会议列表筛选与新建会议下拉） */
export function listMeetingTypes() {
  return request<{ list: MeetingTypeOption[] }>('/api/conference/meetingType/options', {
    method: 'GET',
  })
}

export function createMeeting(payload: {
  title: string
  hostId?: number
  hostName?: string
  startAt: string
  endAt: string
  recordEnabled?: boolean
  /** 会议类型ID，非必选，0=未分类 */
  typeId?: number
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
  /** 会议类型ID，0=未分类；不传表示不改 */
  typeId?: number
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

export function startRecording(room: string) {
  return request<RecordingSegment>('/api/conference/recording/start', {
    method: 'POST',
    body: JSON.stringify({ room }),
  })
}

export function stopRecording(room: string) {
  return request<RecordingSegment>('/api/conference/recording/stop', {
    method: 'POST',
    body: JSON.stringify({ room }),
  })
}

export function recordingStatus(params: { room?: string; meetingId?: number }) {
  const q = new URLSearchParams()
  if (params.room) q.set('room', params.room)
  if (params.meetingId) q.set('meetingId', String(params.meetingId))
  return request<RecordingStatus>(`/api/conference/recording/status?${q.toString()}`, {
    method: 'GET',
  })
}

/** <video> 不能带 Authorization 头，走 HotGo 回放代理并把登录态放 query */
export function recordingPlaySrc(id: number): string {
  const base = getApiBaseUrl()
  const token = getToken() || ''
  const q = new URLSearchParams()
  q.set('id', String(id))
  if (token) q.set('authorization', token)
  return `${base}/api/conference/recording/play?${q.toString()}`
}

export async function downloadRecordingFile(id: number, filename: string) {
  const base = getApiBaseUrl()
  const token = getToken()
  const headers: Record<string, string> = {}
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  let res: Response
  try {
    res = await fetch(`${base}/api/conference/recording/download?id=${id}`, { headers })
  } catch {
    throw new ApiError(-1, '无法连接业务服务，下载失败')
  }
  const contentType = res.headers.get('content-type') || ''
  if (!res.ok || contentType.includes('application/json')) {
    let message = `下载失败（HTTP ${res.status}）`
    try {
      const body = (await res.json()) as { message?: string }
      if (body.message) message = body.message
    } catch {
      // ignore parse error
    }
    throw new ApiError(res.status, message)
  }
  const blob = await res.blob()
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objectUrl)
}
