/**
 * 会议进会凭证（sessionStorage）封装：按房间隔离。
 *
 * 统一用 `vc.session.<room>` 作为 key，而不是单个 `vc.session`：
 * 这样即便有异常路径残留了某个会议的旧数据，也天然按房间隔离，
 * 进入 / 刷新另一个会议时绝不会复用上一场会议的开关状态。
 */

export interface MeetingSession {
  serverUrl: string
  token: string
  room: string
  title?: string
  identity: string
  nickname: string
  expiresAt: number
  isHost: boolean
  enableMic?: boolean
  enableCamera?: boolean
  fromShare?: boolean
  shareCode?: string
  recordEnabled?: boolean
  recordingActive?: boolean
  /** 会议预定开始时间；房间「已进行」时长以此累计，与是否重新进房无关 */
  startAt?: string
  /** 实际开始时间（首个参会者提前入会时记录）；为空则计时以 startAt 为起点 */
  actualStartAt?: string
}

const SESSION_PREFIX = 'vc.session'

/** session key 按房间隔离：`vc.session.<room>`，不同会议互不覆盖 */
export function meetingSessionKey(room: string): string {
  return `${SESSION_PREFIX}.${room}`
}

export function readMeetingSession(room: string | string[]): MeetingSession | null {
  const key = meetingSessionKey(String(room))
  const raw = sessionStorage.getItem(key)
  if (!raw) return null
  try {
    return JSON.parse(raw) as MeetingSession
  } catch {
    return null
  }
}

export function writeMeetingSession(payload: MeetingSession): void {
  if (!payload?.room) return
  sessionStorage.setItem(meetingSessionKey(payload.room), JSON.stringify(payload))
}

export function removeMeetingSession(room: string | string[]): void {
  if (!room) return
  sessionStorage.removeItem(meetingSessionKey(String(room)))
}
