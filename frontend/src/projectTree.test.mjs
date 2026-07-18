import assert from 'node:assert/strict'
import { buildProjectTree } from './projectTree.js'

const projects = [
  {
    id: 'toolkit',
    name: 'ad-ai-toolkit',
    path: '/home/attson/GolandProjects/ad-ai-toolkit',
    command: 'go',
    args: ['run', 'main.go', 'serve'],
    detectedType: 'go',
  },
  {
    id: 'starter',
    name: 'atstarter',
    path: '/home/attson/GolandProjects/atstarter',
    command: 'wails',
    args: ['dev', '-tags', 'webkit2_41'],
    detectedType: 'wails',
  },
  {
    id: 'admin',
    name: 'admin-ui',
    path: '/home/attson/WebstormProjects/admin-ui',
    command: 'pnpm',
    args: ['dev'],
    detectedType: 'node',
  },
]

const statuses = {
  toolkit: { State: 'running', PID: 18422 },
  starter: { State: 'stopped' },
  admin: { State: 'exited', ExitCode: 1 },
}

const tree = buildProjectTree(projects, statuses, '')
assert.equal(tree.length, 2)
assert.equal(tree[0].label, 'GolandProjects')
assert.equal(tree[0].count, 2)
assert.equal(tree[0].children[0].type, 'project')
assert.equal(tree[0].children[0].project.name, 'ad-ai-toolkit')
assert.equal(tree[0].children[0].status.State, 'running')
assert.equal(tree[1].label, 'WebstormProjects')
assert.equal(tree[1].count, 1)

const filtered = buildProjectTree(projects, statuses, 'serve')
assert.equal(filtered.length, 1)
assert.equal(filtered[0].label, 'GolandProjects')
assert.equal(filtered[0].children.length, 1)
assert.equal(filtered[0].children[0].project.id, 'toolkit')

const nested = buildProjectTree([
  {
    id: 'svc',
    name: 'billing-api',
    path: '/home/attson/GolandProjects/company/payments/billing-api',
    command: 'go',
    args: ['run', 'main.go'],
    detectedType: 'go',
  },
], {}, '')
// Deep paths collapse to their immediate parent only — no grandparent chain.
assert.equal(nested.length, 1)
assert.equal(nested[0].label, 'payments')
assert.equal(nested[0].children[0].type, 'project')
assert.equal(nested[0].children[0].project.name, 'billing-api')

// Two projects with the same immediate parent name but different full paths
// render as separate groups (distinguished by full path key).
const siblingParents = buildProjectTree([
  { id: 'a1', name: 'app', path: '/home/attson/team-a/services/app' },
  { id: 'b1', name: 'app', path: '/home/attson/team-b/services/app' },
], {}, '')
assert.equal(siblingParents.length, 2)
assert.deepEqual(siblingParents.map((n) => n.label), ['services', 'services'])
assert.notEqual(siblingParents[0].path, siblingParents[1].path)

const withWorktrees = buildProjectTree([
  {
    id: 'platform',
    name: 'ad-ai-platform',
    path: '/home/attson/GolandProjects/ad-ai-platform',
    command: 'go',
    args: ['run', 'main.go'],
    detectedType: 'go',
  },
  {
    id: 'budget',
    name: 'budget-usage-proxy',
    path: '/home/attson/GolandProjects/ad-ai-platform/.claude/worktrees/budget-usage-proxy',
    command: 'go',
    args: ['run', 'main.go'],
    detectedType: 'go',
  },
  {
    id: 'material',
    name: 'material-tag-proxy',
    path: '/home/attson/GolandProjects/ad-ai-platform/.worktrees/material-tag-proxy',
    command: 'go',
    args: ['run', 'main.go'],
    detectedType: 'go',
  },
], {}, '')
assert.equal(withWorktrees[0].label, 'GolandProjects')
const platform = withWorktrees[0].children.find((node) => node.type === 'project' && node.project.id === 'platform')
assert.ok(platform)
assert.equal(platform.count, 2)
assert.equal(platform.children.length, 2)
assert.deepEqual(platform.children.map((node) => node.project.name), ['budget-usage-proxy', 'material-tag-proxy'])
assert.equal(withWorktrees[0].children.filter((node) => node.type === 'directory' && node.label === 'ad-ai-platform').length, 0)

const filteredWorktree = buildProjectTree(withWorktrees[0].children
  .filter((node) => node.type === 'project')
  .flatMap((node) => [node.project, ...(node.children || []).map((child) => child.project)]), {}, 'budget')
const filteredPlatform = filteredWorktree[0].children.find((node) => node.type === 'project' && node.project.name === 'ad-ai-platform')
assert.ok(filteredPlatform)
assert.equal(filteredPlatform.children.length, 1)
assert.equal(filteredPlatform.children[0].project.name, 'budget-usage-proxy')

console.log('projectTree tests passed')
