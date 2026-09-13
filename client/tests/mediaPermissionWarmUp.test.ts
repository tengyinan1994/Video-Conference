import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  hasRealVideoInput,
  warmUpMediaPermissionsWith,
  type CreateLocalTracksOpts,
} from '../src/composables/mediaPermissionWarmUp.ts'

describe('hasRealVideoInput', () => {
  it('false when list is empty or only audio', () => {
    assert.equal(hasRealVideoInput([]), false)
    assert.equal(hasRealVideoInput([{ kind: 'audioinput', deviceId: 'mic1' }]), false)
  })

  it('false when videoinput has no real deviceId', () => {
    assert.equal(hasRealVideoInput([{ kind: 'videoinput', deviceId: '' }]), false)
    assert.equal(hasRealVideoInput([{ kind: 'videoinput' }]), false)
  })

  it('true when a videoinput has deviceId', () => {
    assert.equal(
      hasRealVideoInput([
        { kind: 'audioinput', deviceId: 'mic1' },
        { kind: 'videoinput', deviceId: 'cam1' },
      ]),
      true,
    )
  })
})

describe('warmUpMediaPermissionsWith', () => {
  it('audio-only then enumerate; never requests video without videoinput', async () => {
    const events: string[] = []
    const calls: CreateLocalTracksOpts[] = []
    await warmUpMediaPermissionsWith({
      createLocalTracks: async (opts) => {
        events.push(`create:${opts.audio}:${opts.video}`)
        calls.push(opts)
        return [{ stop: () => events.push('stop') }]
      },
      enumerateDevices: async () => {
        events.push('enumerate')
        return [{ kind: 'audioinput', deviceId: 'mic1' }]
      },
    })
    assert.deepEqual(events, ['create:true:false', 'stop', 'enumerate'])
    assert.deepEqual(calls, [{ audio: true, video: false }])
  })

  it('never requests video when videoinput has empty deviceId', async () => {
    const calls: CreateLocalTracksOpts[] = []
    await warmUpMediaPermissionsWith({
      createLocalTracks: async (opts) => {
        calls.push(opts)
        return [{ stop() {} }]
      },
      enumerateDevices: async () => [{ kind: 'videoinput', deviceId: '' }],
    })
    assert.deepEqual(calls, [{ audio: true, video: false }])
  })

  it('requests video only after a real videoinput is listed', async () => {
    const events: string[] = []
    const calls: CreateLocalTracksOpts[] = []
    await warmUpMediaPermissionsWith({
      createLocalTracks: async (opts) => {
        events.push(`create:${opts.audio}:${opts.video}`)
        calls.push(opts)
        return [{ stop: () => events.push(`stop:${opts.audio}:${opts.video}`) }]
      },
      enumerateDevices: async () => {
        events.push('enumerate')
        return [
          { kind: 'audioinput', deviceId: 'mic1' },
          { kind: 'videoinput', deviceId: 'cam1' },
        ]
      },
    })
    assert.deepEqual(events, [
      'create:true:false',
      'stop:true:false',
      'enumerate',
      'create:false:true',
      'stop:false:true',
    ])
    assert.deepEqual(calls, [
      { audio: true, video: false },
      { audio: false, video: true },
    ])
  })

  it('ignores video warm-up failure after listing a camera', async () => {
    const calls: CreateLocalTracksOpts[] = []
    await warmUpMediaPermissionsWith({
      createLocalTracks: async (opts) => {
        calls.push(opts)
        if (opts.video) throw new Error('video capture failed')
        return [{ stop() {} }]
      },
      enumerateDevices: async () => [{ kind: 'videoinput', deviceId: 'cam1' }],
    })
    assert.deepEqual(calls, [
      { audio: true, video: false },
      { audio: false, video: true },
    ])
  })
})
