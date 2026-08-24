#!/usr/bin/env node
// atstarter-mcp: a thin launcher that locates the locally installed atstarter
// binary and execs `atstarter mcp`, forwarding stdio transparently. It downloads
// nothing — the desktop app (which ships the binary) is the prerequisite.

import { spawn } from 'node:child_process'

import { locateAtstarterBinary } from '../lib/locate.mjs'

const binary = locateAtstarterBinary()

if (!binary) {
  // Never write to stdout: it carries the MCP protocol stream.
  process.stderr.write(
    [
      'atstarter-mcp: could not find the installed atstarter binary.',
      '',
      'This launcher only wraps an already-installed AT Starter desktop app;',
      'it does not download anything. Install AT Starter first:',
      '  https://github.com/attson/atstarter/releases/latest',
      '',
      'If atstarter is installed in a non-standard location, point ATSTARTER_BIN',
      'at the executable, e.g. ATSTARTER_BIN=/path/to/atstarter.',
      '',
    ].join('\n'),
  )
  process.exit(1)
}

// Pass through any extra args (e.g. a future `atstarter mcp --flag`).
const args = ['mcp', ...process.argv.slice(2)]
const child = spawn(binary, args, { stdio: 'inherit' })

// Forward termination signals so MCP clients can cleanly stop the server.
const signals = ['SIGINT', 'SIGTERM', 'SIGHUP']
for (const signal of signals) {
  process.on(signal, () => {
    if (!child.killed) {
      child.kill(signal)
    }
  })
}

child.on('error', (err) => {
  process.stderr.write(`atstarter-mcp: failed to start ${binary}: ${err.message}\n`)
  process.exit(1)
})

child.on('exit', (code, signal) => {
  if (signal) {
    // Re-raise the signal so our exit status mirrors the child's.
    process.kill(process.pid, signal)
    return
  }
  process.exit(code ?? 0)
})
