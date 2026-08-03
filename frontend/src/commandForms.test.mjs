import test from 'node:test'
import assert from 'node:assert/strict'
import { commandFormsForProject, blankCommandForm } from './commandForms.js'

test('commandFormsForProject uses project path as cwd value when command cwd is empty', () => {
  const forms = commandFormsForProject({
    name: 'api',
    path: '/home/attson/GolandProjects/api',
    commands: [{
      id: 'default',
      name: 'Default',
      command: 'go',
      args: ['run', 'main.go'],
      cwd: '',
      env: { PORT: '8080' },
      isDefault: true,
    }],
  })

  assert.equal(forms[0].cwd, '/home/attson/GolandProjects/api')
})

test('commandFormsForProject preserves explicit command cwd', () => {
  const forms = commandFormsForProject({
    path: '/repo',
    commands: [{
      id: 'debug',
      name: 'Debug',
      command: 'pnpm',
      args: ['dev'],
      cwd: '/repo/frontend',
      isDefault: true,
    }],
  })

  assert.equal(forms[0].cwd, '/repo/frontend')
})

test('commandFormsForProject uses legacy project env and path for fallback command', () => {
  const forms = commandFormsForProject({
    path: '/repo',
    command: 'go',
    args: ['run', '.'],
    env: { NODE_ENV: 'dev' },
  })

  assert.equal(forms[0].cwd, '/repo')
  assert.equal(forms[0].envText, 'NODE_ENV=dev')
})

test('blankCommandForm seeds cwd with project path as an editable value', () => {
  const form = blankCommandForm({ path: '/repo' }, 1)

  assert.equal(form.cwd, '/repo')
  assert.equal(form.line, '')
  assert.equal(form.envText, '')
  assert.equal(form.name, 'Command 2')
  assert.equal(form.isDefault, false)
})

test('blankCommandForm makes the first command default', () => {
  const form = blankCommandForm({ path: '/repo' }, 0)

  assert.equal(form.isDefault, true)
  assert.equal(form.name, 'Command 1')
})

test('blankCommandForm tolerates a project without a path', () => {
  const form = blankCommandForm(null, 0)

  assert.equal(form.cwd, '')
})
