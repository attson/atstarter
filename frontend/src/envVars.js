export function envTextToMap(text) {
  const env = {}
  for (const rawLine of String(text || '').split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line) continue
    const eq = line.indexOf('=')
    if (eq < 0) continue
    const key = line.slice(0, eq).trim()
    if (!key) continue
    env[key] = line.slice(eq + 1).trim()
  }
  return env
}

export function envMapToText(env) {
  return Object.keys(env || {})
    .sort()
    .map((key) => `${key}=${env[key] ?? ''}`)
    .join('\n')
}
