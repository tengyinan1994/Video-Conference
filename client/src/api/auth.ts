import { request } from '@/utils/request'

export interface LoginResult {
  id: number
  username: string
  token: string
  expires: number
}

export interface MeResult {
  id: number
  username: string
  realName: string
}

export interface CaptchaResult {
  cid: string
  base64: string
}

export interface LoginConfigResult {
  captchaSwitch: number
  projectName: string
}

export function fetchCaptcha() {
  return request<CaptchaResult>('/api/conference/auth/captcha', { method: 'GET' })
}

export function fetchLoginConfig() {
  return request<LoginConfigResult>('/api/conference/auth/loginConfig', { method: 'GET' })
}

export function login(payload: {
  username: string
  password: string
  cid?: string
  code?: string
}) {
  return request<LoginResult>('/api/conference/auth/login', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function logout() {
  return request<Record<string, never>>('/api/conference/auth/logout', { method: 'POST' })
}

export function fetchMe() {
  return request<MeResult>('/api/conference/auth/me', { method: 'GET' })
}
