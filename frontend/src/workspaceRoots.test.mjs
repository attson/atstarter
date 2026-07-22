import { test } from 'node:test'
import assert from 'node:assert/strict'
import { inferWorkspaceRoots, parseWorkspaceRoots } from './workspaceRoots.js'

test('parseWorkspaceRoots trims blank lines and deduplicates roots', () => {
  assert.deepEqual(parseWorkspaceRoots(' ~/GolandProjects \n\n~/WebstormProjects\n~/GolandProjects'), [
    '~/GolandProjects',
    '~/WebstormProjects',
  ])
})

test('inferWorkspaceRoots uses common immediate parent directories', () => {
  const roots = inferWorkspaceRoots([
    { path: '/home/attson/GolandProjects/api-a' },
    { path: '/home/attson/GolandProjects/api-b' },
    { path: '/home/attson/WebstormProjects/web-a' },
    { path: '/home/attson/WebstormProjects/web-b' },
    { path: '/home/attson/tmp/one-off' },
  ])

  assert.deepEqual(roots, [
    '/home/attson/GolandProjects',
    '/home/attson/WebstormProjects',
  ])
})

test('inferWorkspaceRoots falls back to single project parent when no common roots exist', () => {
  assert.deepEqual(inferWorkspaceRoots([{ path: '/home/attson/solo/app' }]), ['/home/attson/solo'])
})
