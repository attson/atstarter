import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  groupContainers,
  filterContainers,
  findComposeProject,
  normalizeComposeProjectName,
  summarizeContainers,
} from './dockerState.js'

test('groupContainers splits compose vs standalone', () => {
  const list = [
    { id: '1', name: 'myapp-web-1', compose: 'myapp', service: 'web', state: 'running' },
    { id: '2', name: 'redis', compose: '', service: '', state: 'running' },
  ]
  const groups = groupContainers(list)
  assert.equal(groups.compose.myapp.length, 1)
  assert.equal(groups.standalone.length, 1)
  assert.equal(groups.standalone[0].name, 'redis')
})

test('filterContainers matches name substring case-insensitive', () => {
  const list = [{ name: 'Redis' }, { name: 'postgres' }]
  assert.equal(filterContainers(list, 'red').length, 1)
  assert.equal(filterContainers(list, '').length, 2)
})

test('summarizeContainers counts states and unique images', () => {
  const summary = summarizeContainers([
    { state: 'running', image: 'mysql:8', composeWorkingDir: '/srv/app' },
    { state: 'exited', image: 'redis:7' },
    { state: 'running', image: 'mysql:8' },
  ])

  assert.equal(summary.total, 3)
  assert.equal(summary.running, 2)
  assert.equal(summary.exited, 1)
  assert.equal(summary.workingDir, '/srv/app')
  assert.deepEqual(summary.images, ['mysql:8', 'redis:7'])
})

test('normalizeComposeProjectName mirrors backend compose project naming', () => {
  assert.equal(normalizeComposeProjectName('Team Manage'), 'teammanage')
  assert.equal(normalizeComposeProjectName('project_base'), 'project_base')
  assert.equal(normalizeComposeProjectName('a.b-c'), 'ab-c')
})

test('findComposeProject matches compose containers to registered compose projects', () => {
  const projects = [
    { id: 'plain', path: '/tmp/plain', detectedType: 'go' },
    { id: 'team', path: '/workspace/Team Manage', detectedType: 'compose' },
  ]

  assert.equal(findComposeProject(projects, 'teammanage').id, 'team')
  assert.equal(findComposeProject(projects, 'missing'), null)
})
