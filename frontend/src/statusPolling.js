export const STATUS_POLL_INTERVAL_MS = 1500

export function startVisibleStatusPolling(refresh, {
  documentRef = document,
  intervalMs = STATUS_POLL_INTERVAL_MS,
  setIntervalFn = setInterval,
  clearIntervalFn = clearInterval,
} = {}) {
  let timer = null

  const stopTimer = () => {
    if (timer === null) return
    clearIntervalFn(timer)
    timer = null
  }
  const refreshIfVisible = () => {
    if (!documentRef.hidden) refresh()
  }
  const startTimer = () => {
    if (documentRef.hidden || timer !== null) return
    timer = setIntervalFn(refreshIfVisible, intervalMs)
  }
  const onVisibilityChange = () => {
    if (documentRef.hidden) {
      stopTimer()
      return
    }
    refresh()
    startTimer()
  }

  documentRef.addEventListener('visibilitychange', onVisibilityChange)
  startTimer()

  return () => {
    stopTimer()
    documentRef.removeEventListener('visibilitychange', onVisibilityChange)
  }
}
