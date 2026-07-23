import test from 'node:test'
import assert from 'node:assert/strict'
import {
  applyDetectionOption,
  commandLineForDetection,
  detectionOptionsFor,
  hasDetectionSwitch,
  ignoreComposeDetections,
} from './projectDetection.js'

test('detectionOptionsFor returns switchable compose fallback options', () => {
  const project = {
    detectedType: 'compose',
    detectionOptions: [
      { type: 'compose', command: '', args: [] },
      { type: 'go', command: 'go', args: ['run', 'main.go'] },
    ],
  }

  const options = detectionOptionsFor(project)

  assert.equal(hasDetectionSwitch(project), true)
  assert.deepEqual(options.map((o) => o.type), ['compose', 'go'])
  assert.equal(commandLineForDetection(options[1]), 'go run main.go')
})

test('applyDetectionOption switches compose project to normal default command', () => {
  const project = {
    id: 'p1',
    detectedType: 'compose',
    command: '',
    args: [],
    cwd: '/repo',
    env: { NODE_ENV: 'dev' },
  }
  const next = applyDetectionOption(project, { type: 'go', command: 'go', args: ['run', '.'] })

  assert.equal(next.detectedType, 'go')
  assert.equal(next.command, 'go')
  assert.deepEqual(next.args, ['run', '.'])
  assert.equal(next.autoDetected, false)
  assert.equal(next.commands.length, 1)
  assert.equal(next.commands[0].id, 'default')
  assert.equal(next.commands[0].command, 'go')
})

test('applyDetectionOption switches normal project back to compose', () => {
  const project = {
    id: 'p1',
    detectedType: 'go',
    command: 'go',
    args: ['run', '.'],
    commands: [{ id: 'default', name: 'Default', command: 'go', args: ['run', '.'], isDefault: true }],
  }
  const next = applyDetectionOption(project, { type: 'compose', command: '', args: [] })

  assert.equal(next.detectedType, 'compose')
  assert.equal(next.command, '')
  assert.deepEqual(next.args, [])
  assert.deepEqual(next.commands, [])
})

test('ignoreComposeDetections switches all compose candidates to fallback', () => {
  const input = [
    {
      id: 'a',
      detectedType: 'compose',
      command: '',
      args: [],
      detectionOptions: [
        { type: 'compose', command: '', args: [] },
        { type: 'go', command: 'go', args: ['run', '.'] },
      ],
    },
    {
      id: 'b',
      detectedType: 'compose',
      command: '',
      args: [],
      detectionOptions: [{ type: 'compose', command: '', args: [] }],
    },
    {
      id: 'c',
      detectedType: 'go',
      command: 'go',
      args: ['run', 'main.go'],
    },
  ]

  const output = ignoreComposeDetections(input)

  assert.equal(output[0].detectedType, 'go')
  assert.equal(output[0].command, 'go')
  assert.deepEqual(output[0].args, ['run', '.'])
  assert.equal(output[1], input[1])
  assert.equal(output[2], input[2])
})
