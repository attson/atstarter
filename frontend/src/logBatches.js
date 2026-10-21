export function logLinesForRun(batch, runId) {
  if (!runId || !batch || !Array.isArray(batch.lines)) return []
  return batch.lines.filter((line) => line && line.id === runId)
}

export function terminalChunk(lines) {
  if (!Array.isArray(lines) || lines.length === 0) return ''
  return lines.map((line) => `${line.text ?? ''}\r\n`).join('')
}
