// Shared status helpers for group member rows.
// Run status lives in App.vue `statuses`, keyed by `projectId:commandId`.

export function stateClass(state) {
  if (state === 'running') return 'running'
  if (state === 'error' || state === 'exited') return 'bad'
  return 'stopped'
}

export function memberStatusState(statuses, projectId, commandId) {
  const runId = `${projectId}:${commandId || 'default'}`
  const status = statuses && statuses[runId]
  return (status && status.State) || 'stopped'
}
