import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  UPDATE_CHECK_INTERVAL_MS,
  isUpdateBannerVisible,
  startUpdateCheckTimer,
} from './updateSchedule.js'

test('startUpdateCheckTimer schedules periodic checks with the default interval', () => {
  let scheduledFn
  let scheduledMs
  const cleanup = startUpdateCheckTimer(() => {}, {
    setIntervalFn(fn, ms) {
      scheduledFn = fn
      scheduledMs = ms
      return 42
    },
    clearIntervalFn() {},
  })

  assert.equal(typeof scheduledFn, 'function')
  assert.equal(scheduledMs, UPDATE_CHECK_INTERVAL_MS)
  assert.equal(typeof cleanup, 'function')
})

test('startUpdateCheckTimer cleanup clears the interval handle', () => {
  let cleared
  const cleanup = startUpdateCheckTimer(() => {}, {
    setIntervalFn() { return 'timer-id' },
    clearIntervalFn(id) { cleared = id },
  })

  cleanup()

  assert.equal(cleared, 'timer-id')
})

test('isUpdateBannerVisible keeps silent checks quiet until an update is available', () => {
  assert.equal(isUpdateBannerVisible({ checking: true }, { manualNotice: false, dismissed: false }), false)
  assert.equal(isUpdateBannerVisible({ error: 'network failed' }, { manualNotice: false, dismissed: false }), false)
  assert.equal(isUpdateBannerVisible({ available: true }, { manualNotice: false, dismissed: false }), true)
})

test('isUpdateBannerVisible shows feedback for manual checks', () => {
  assert.equal(isUpdateBannerVisible({ checking: true }, { manualNotice: true, dismissed: false }), true)
  assert.equal(isUpdateBannerVisible({ lastCheckAt: 1, current: 'v0.4.0', latest: 'v0.4.0' }, { manualNotice: true, dismissed: false }), true)
  assert.equal(isUpdateBannerVisible({ error: 'network failed' }, { manualNotice: true, dismissed: false }), true)
})
