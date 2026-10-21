import assert from 'node:assert/strict'
import { test } from 'node:test'

import { startVisibleStatusPolling } from './statusPolling.js'

function fakeDocument(hidden = false) {
  const listeners = new Map()
  return {
    hidden,
    addEventListener(name, fn) { listeners.set(name, fn) },
    removeEventListener(name) { listeners.delete(name) },
    dispatch(name) { listeners.get(name)?.() },
  }
}

test('status polling pauses while hidden and refreshes immediately when visible', () => {
  const documentRef = fakeDocument(true)
  const refreshes = []
  const intervals = []
  const cleared = []
  const stop = startVisibleStatusPolling(() => refreshes.push('refresh'), {
    documentRef,
    setIntervalFn(fn, ms) { intervals.push({ fn, ms }); return intervals.length },
    clearIntervalFn(id) { cleared.push(id) },
  })

  assert.equal(intervals.length, 0)
  assert.equal(refreshes.length, 0)

  documentRef.hidden = false
  documentRef.dispatch('visibilitychange')
  assert.equal(refreshes.length, 1)
  assert.equal(intervals.length, 1)

  documentRef.hidden = true
  documentRef.dispatch('visibilitychange')
  assert.deepEqual(cleared, [1])

  stop()
})

test('status polling never calls refresh from an interval while hidden', () => {
  const documentRef = fakeDocument(false)
  let refreshes = 0
  let intervalCallback
  const stop = startVisibleStatusPolling(() => { refreshes += 1 }, {
    documentRef,
    setIntervalFn(fn) { intervalCallback = fn; return 7 },
    clearIntervalFn() {},
  })

  documentRef.hidden = true
  intervalCallback()
  assert.equal(refreshes, 0)
  stop()
})
