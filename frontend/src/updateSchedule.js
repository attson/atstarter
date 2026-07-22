export const UPDATE_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000

export function startUpdateCheckTimer(check, {
  intervalMs = UPDATE_CHECK_INTERVAL_MS,
  setIntervalFn = setInterval,
  clearIntervalFn = clearInterval,
} = {}) {
  const timer = setIntervalFn(check, intervalMs)
  return () => clearIntervalFn(timer)
}

export function isUpdateBannerVisible(state, { dismissed = false, manualNotice = false } = {}) {
  if (dismissed) return false
  if (!state) return false
  if (state.available || state.downloading || state.ready) return true
  if (!manualNotice) return false
  return !!(state.checking || state.error || state.lastCheckAt)
}
