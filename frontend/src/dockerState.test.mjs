import { test } from 'node:test'
import assert from 'node:assert/strict'
import { groupContainers, filterContainers } from './dockerState.js'

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
