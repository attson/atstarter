#!/usr/bin/env node
// rootyard-mcp: a thin launcher that locates the locally installed rootyard
// binary and execs `rootyard mcp`, forwarding stdio transparently. It downloads
// nothing — the desktop app (which ships the binary) is the prerequisite.

import { spawn } from 'node:child_process'

import { locateRootyardBinary } from '../lib/locate.mjs'

const binary = locateRootyardBinary()

if (!binary) {
  // Never write to stdout: it carries the MCP protocol stream.
  process.stderr.write(
    [
      'rootyard-mcp: could not find the installed rootyard binary.',
      '',
      'This launcher only wraps an already-installed Rootyard desktop app;',
      'it does not download anything. Install Rootyard first:',
      '  https://github.com/attson/rootyard/releases/latest',
      '',
      'If rootyard is installed in a non-standard location, point ROOTYARD_BIN',
      'at the executable, e.g. ROOTYARD_BIN=/path/to/rootyard.',
      '',
    ].join('\n'),
  )
  process.exit(1)
}

// Pass through any extra args (e.g. a future `rootyard mcp --flag`).
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
  process.stderr.write(`rootyard-mcp: failed to start ${binary}: ${err.message}\n`)
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
