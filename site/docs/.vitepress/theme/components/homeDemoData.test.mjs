import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createHomeDemoState,
  selectHomeDemoProject,
  setHomeDemoCommand,
  simulateHomeDemoEdit,
  simulateHomeDemoScan,
  toggleHomeDemoGroup,
  toggleHomeDemoRun,
} from './homeDemoData.mjs'

test('selecting a project updates the active project and command', () => {
  const state = createHomeDemoState()
  const next = selectHomeDemoProject(state, 'api')

  assert.equal(next.activeProjectId, 'api')
  assert.equal(next.activeCommandId, 'dev')
  assert.equal(next.projects.find((project) => project.id === 'api').name, 'api-server')
})

test('changing the command keeps the selected project and updates the log preview', () => {
  const state = selectHomeDemoProject(createHomeDemoState(), 'web')
  const next = setHomeDemoCommand(state, 'test')

  assert.equal(next.activeProjectId, 'web')
  assert.equal(next.activeCommandId, 'test')
  assert.match(next.logLines.at(-1), /vitest run/)
})

test('toggling a stopped project starts it and appends a status log line', () => {
  const state = selectHomeDemoProject(createHomeDemoState(), 'docs')
  const next = toggleHomeDemoRun(state)
  const project = next.projects.find((item) => item.id === 'docs')

  assert.equal(project.status, 'running')
  assert.match(next.logLines.at(-1), /started docs-site/)
})

test('toggling a running project stops it and appends a status log line', () => {
  const state = selectHomeDemoProject(createHomeDemoState(), 'web')
  const next = toggleHomeDemoRun(state)
  const project = next.projects.find((item) => item.id === 'web')

  assert.equal(project.status, 'stopped')
  assert.match(next.logLines.at(-1), /stopped web-admin/)
})

test('toggling the group stops running frontend and api projects', () => {
  const state = createHomeDemoState()
  const next = toggleHomeDemoGroup(state)
  const grouped = next.projects.filter((project) => project.id === 'web' || project.id === 'api')

  assert.deepEqual(grouped.map((project) => project.status), ['stopped', 'stopped'])
  assert.match(next.logLines.at(-1), /stopped frontend \+ api/)
})

test('toggling the group starts frontend and api projects when one is stopped', () => {
  const state = toggleHomeDemoGroup(createHomeDemoState())
  const next = toggleHomeDemoGroup(state)
  const grouped = next.projects.filter((project) => project.id === 'web' || project.id === 'api')

  assert.deepEqual(grouped.map((project) => project.status), ['running', 'running'])
  assert.match(next.logLines.at(-1), /started frontend \+ api/)
})

test('mock scan and edit actions append visible log feedback', () => {
  const scanned = simulateHomeDemoScan(createHomeDemoState())
  const edited = simulateHomeDemoEdit(scanned)

  assert.match(scanned.logLines.at(-1), /workspace scan finished/)
  assert.match(edited.logLines.at(-1), /command editor opened/)
})
