import assert from 'node:assert/strict'
import test from 'node:test'

import {
  GetLogs,
  GetProjectBranch,
  GetStatus,
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
