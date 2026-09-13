import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { resolveLiveKitUrlFromEnv } from '../src/composables/resolveLiveKitUrl.ts'

const SERVER = 'wss://livekit.example:7880'

describe('resolveLiveKitUrlFromEnv', () => {
  it('Tauri: always return serverUrl as-is', () => {
    assert.equal(
      resolveLiveKitUrlFromEnv(SERVER, {
        isTauri: true,
        protocol: 'https:',
        host: 'tauri.localhost',
        isDev: false,
      }),
      SERVER,
    )
    assert.equal(
      resolveLiveKitUrlFromEnv(SERVER, {
        isTauri: true,
        protocol: 'http:',
        host: 'localhost:5173',
        isDev: true,
      }),
      SERVER,
    )
  })

  it('browser https: same-origin wss rewrite', () => {
    assert.equal(
      resolveLiveKitUrlFromEnv(SERVER, {
        isTauri: false,
        protocol: 'https:',
        host: 'meet.example:17885',
        isDev: false,
      }),
      'wss://meet.example:17885',
    )
  })

  it('browser http: DEV 走同源 ws，生产原样返回 serverUrl', () => {
    assert.equal(
      resolveLiveKitUrlFromEnv(SERVER, {
        isTauri: false,
        protocol: 'http:',
        host: '127.0.0.1:5173',
        isDev: true,
      }),
      'ws://127.0.0.1:5173',
    )
    assert.equal(
      resolveLiveKitUrlFromEnv(SERVER, {
        isTauri: false,
        protocol: 'http:',
        host: '192.168.1.10:5173',
        isDev: false,
      }),
      SERVER,
    )
  })
})
