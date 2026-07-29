import assert from 'node:assert/strict'
import test from 'node:test'

import {
  GetLogs,
  GetProjectBranch,
  GetStatus,
  ListGroups,
  ListProjects,
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

test('mock Wails app returns branch names for project headers', async () => {
  assert.equal(await GetProjectBranch('/Users/demo/workspaces/atlas-api'), 'master')
})
