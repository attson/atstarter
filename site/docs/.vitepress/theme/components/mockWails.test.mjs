import assert from 'node:assert/strict'
import test from 'node:test'
import * as mock from './mockWailsApp.mjs'
import { EventsOn } from './mockWailsRuntime.mjs'

import {
  GetLogs,
  GetProjectBranch,
  GetStatus,
  GetStatuses,
  ListGroups,
  ListProjects,
  ProjectFileMeta,
  ReadProjectFileBytes,
  StartProjectCommand,
  StopProjectCommand,
} from './mockWailsApp.mjs'

test('mock Wails app exposes realistic project and group data', async () => {
  const projects = await ListProjects()
  const groups = await ListGroups()

  assert.ok(projects.length >= 6)
  assert.ok(groups.length >= 1)
  assert.ok(projects.some((project) => project.name === 'atlas-api'))
  assert.ok(projects.every((project) => project.id && project.path && Array.isArray(project.commands)))
})

test('mock Wails app updates command status and logs for the real app controls', async () => {
  const project = (await ListProjects()).find((item) => item.name === 'atlas-api')
  assert.ok(project)

  const commandId = project.commands[0].id
  const runId = `${project.id}:${commandId}`

  await StopProjectCommand(project.id, commandId)
  assert.equal((await GetStatus(runId)).State, 'stopped')

  await StartProjectCommand(project.id, commandId)
  assert.equal((await GetStatus(runId)).State, 'running')
  assert.match((await GetLogs(runId)).join('\n'), /go run main\.go/)
})

test('mock Wails app starts with visible running projects and seeded logs', async () => {
  assert.equal((await GetStatus('atlas-api:go')).State, 'running')
  assert.equal((await GetStatus('atlas-worker:go')).State, 'running')
  assert.match((await GetLogs('atlas-api:go')).join('\n'), /listening on http:\/\/localhost:8080/)
})

test('mock Wails app returns statuses in one RunID-keyed snapshot', async () => {
  const statuses = await GetStatuses(['atlas-api:go', 'missing:default'])
  assert.equal(statuses['atlas-api:go'].State, 'running')
  assert.equal(statuses['missing:default'].State, 'stopped')
})

test('mock Wails app returns branch names for project headers', async () => {
  assert.equal(await GetProjectBranch('/Users/demo/workspaces/atlas-api'), 'master')
})

test('mock file bridge returns editable bytes and metadata for selected files', async () => {
  const meta = await ProjectFileMeta('atlas-api', 'cmd/main.go')
  const file = await ReadProjectFileBytes('atlas-api', 'cmd/main.go', 2048)
  const text = new TextDecoder().decode(new Uint8Array(file.data))

  assert.equal(meta.isDir, false)
  assert.equal(meta.isBinary, false)
  assert.equal(meta.size, file.data.length)
  assert.match(text, /package main/)
  assert.match(text, /8080/)
})

test('mock environment reads and edits remain cloned in-memory documents', async () => {
  let document = await mock.GetEnvironmentDocument()
  assert.equal(document.config.version, 2)
  assert.deepEqual(document.config.presets, [])
  document.config.presets.push({ id: 'not-saved' })
  assert.deepEqual((await mock.GetEnvironmentDocument()).config.presets, [])
  document = await mock.SaveEnvironmentPreset({
    revision: document.revision, name: 'Demo preset', description: '', git: {}, codex: {},
  })
  const preset = document.config.presets[0]
  assert.equal(preset.name, 'Demo preset')
  document = await mock.CloneEnvironmentPreset({ revision: document.revision, id: preset.id, name: 'Demo copy' })
  assert.equal(document.config.presets[1].name, 'Demo copy')
  document = await mock.SaveEnvironmentDefaults({ revision: document.revision, presetId: preset.id, git: {}, codex: {} })
  assert.equal(document.config.defaults.presetId, preset.id)
  document = await mock.SaveEnvironmentRule({
    revision: document.revision, name: 'Demo rule', path: '/Users/demo/workspaces',
    recursive: true, autoDiscover: false, presetId: preset.id, git: {}, codex: {},
  })
  assert.equal(document.config.rules[0].path, '/Users/demo/workspaces')
  await assert.rejects(mock.SaveEnvironmentDefaults({ revision: 'outdated', git: {}, codex: {} }), /environment_stale/)
  document = await mock.DeleteEnvironmentRule({ revision: document.revision, id: document.config.rules[0].id })
  assert.deepEqual(document.config.rules, [])
  document = await mock.DeleteEnvironmentPreset({ revision: document.revision, id: preset.id })
  assert.equal(document.config.presets.length, 1)
  assert.deepEqual(await mock.PlanEnvironmentRepositories({ repositoryPaths: [], ruleIds: [], session: {} }), [])
  assert.deepEqual(await mock.ApplyEnvironmentPlans([]), [])
  assert.deepEqual(await mock.DoctorEnvironments({ repositoryPaths: [], ruleIds: [], session: {} }), [])
  const state = await mock.GetEnvironmentState()
  assert.deepEqual(state.repositories, [])
  state.repositories.push({ id: 'not-saved' })
  assert.deepEqual((await mock.GetEnvironmentState()).repositories, [])
})

test('mock environment scans complete asynchronously and support cancellation', async () => {
  let finish
  const completion = new Promise((resolve) => { finish = resolve })
  const stop = EventsOn('projectenv:scan-complete', finish)
  try {
    const scanId = await mock.StartEnvironmentScan([])
    const payload = await completion
    assert.equal(payload.scanId, scanId)
    assert.equal(payload.result.canceled, false)
    assert.equal(mock.CancelEnvironmentScan(scanId), false)
    const pending = await mock.StartEnvironmentScan([])
    assert.equal(mock.CancelEnvironmentScan(pending), true)
    assert.equal(mock.CancelEnvironmentScan(pending), false)
  } finally {
    stop()
  }
})
