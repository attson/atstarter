import { test } from 'node:test'
import assert from 'node:assert/strict'
import { stateClass, memberStatusState } from './groupMemberStatus.js'

test('stateClass maps running to running', () => {
  assert.equal(stateClass('running'), 'running')
})

test('stateClass maps error and exited to bad', () => {
  assert.equal(stateClass('error'), 'bad')
  assert.equal(stateClass('exited'), 'bad')
})

test('stateClass maps everything else to stopped', () => {
  assert.equal(stateClass('stopped'), 'stopped')
  assert.equal(stateClass(undefined), 'stopped')
  assert.equal(stateClass(''), 'stopped')
})

test('memberStatusState reads the State for the exact project:command runId', () => {
  const statuses = {
    'proj-a:default': { State: 'running' },
    'proj-a:build': { State: 'exited' },
  }
  assert.equal(memberStatusState(statuses, 'proj-a', 'default'), 'running')
  assert.equal(memberStatusState(statuses, 'proj-a', 'build'), 'exited')
})

test('memberStatusState defaults commandId to default', () => {
  const statuses = { 'proj-a:default': { State: 'running' } }
  assert.equal(memberStatusState(statuses, 'proj-a', undefined), 'running')
})

test('memberStatusState returns stopped when no status recorded', () => {
  assert.equal(memberStatusState({}, 'proj-a', 'default'), 'stopped')
  assert.equal(memberStatusState(undefined, 'proj-a', 'default'), 'stopped')
})
