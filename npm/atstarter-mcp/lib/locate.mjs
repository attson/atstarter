import { existsSync as nodeExistsSync } from 'node:fs'
import { delimiter as nodeDelimiter, posix, win32 } from 'node:path'

// binaryName returns the platform-specific atstarter executable name.
function binaryName(platform) {
  return platform === 'win32' ? 'atstarter.exe' : 'atstarter'
}

// joinFor picks the join flavour matching the target platform, so path
// separators stay correct even when the test runner OS differs from `platform`.
function joinFor(platform) {
  return platform === 'win32' ? win32.join : posix.join
}

// knownLocations returns absolute candidate paths per platform, matching the
// formal-installer layout documented in docs/tasks/cross-platform-cli-command/spec.md.
function knownLocations(platform, env) {
  const name = binaryName(platform)
  const join = joinFor(platform)
  if (platform === 'darwin') {
    return [
      '/usr/local/bin/atstarter',
      '/Applications/AT Starter.app/Contents/MacOS/atstarter',
    ]
  }
  if (platform === 'win32') {
    const dirs = [env.ProgramFiles, env['ProgramFiles(x86)'], env.LOCALAPPDATA]
    const candidates = []
    for (const dir of dirs) {
      if (!dir) continue
      candidates.push(join(dir, 'AT Starter', name))
    }
    return candidates
  }
  // linux and other unix
  const candidates = ['/usr/bin/atstarter', '/usr/local/bin/atstarter']
  if (env.HOME) {
    candidates.push(join(env.HOME, '.local', 'bin', 'atstarter'))
  }
  return candidates
}

// locateAtstarterBinary finds the installed atstarter binary without downloading
// anything. Resolution order: ATSTARTER_BIN override, then PATH, then the known
// per-platform install locations (fallback for portable packages or terminals
// whose PATH has not refreshed). Returns an absolute path or null.
//
// existsSync and delimiter are injectable so the resolution logic stays a pure,
// testable function.
export function locateAtstarterBinary({
  platform = process.platform,
  env = process.env,
  existsSync = nodeExistsSync,
  delimiter = nodeDelimiter,
} = {}) {
  const name = binaryName(platform)
  const join = joinFor(platform)

  const override = env.ATSTARTER_BIN
  if (override && existsSync(override)) {
    return override
  }

  const pathValue = env.PATH || env.Path || ''
  for (const dir of pathValue.split(delimiter)) {
    if (!dir) continue
    const candidate = join(dir, name)
    if (existsSync(candidate)) {
      return candidate
    }
  }

  for (const candidate of knownLocations(platform, env)) {
    if (existsSync(candidate)) {
      return candidate
    }
  }

  return null
}
