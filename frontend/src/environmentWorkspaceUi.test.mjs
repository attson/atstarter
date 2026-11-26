import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('repository apply UI requires explicit confirmation and blocks unsafe rows', async () => {
  const repository = await readFile(new URL('./components/environments/EnvironmentRepositoryManager.vue', import.meta.url), 'utf8')
  const diff = await readFile(new URL('./components/environments/EnvironmentDiffPanel.vue', import.meta.url), 'utf8')
  assert.match(repository, /:disabled="repositoryStatus\(repository\) === 'blocked'"/)
  assert.match(repository, /from 'lucide-vue-next'/)
  assert.match(diff, /I reviewed these exact native Git changes/)
  assert.match(diff, /:disabled="!confirmed/)
  assert.match(diff, /from 'lucide-vue-next'/)
})

test('repository preview stacks its panels on narrow desktop layouts', async () => {
  const workspace = await readFile(new URL('./components/environments/EnvironmentWorkspace.vue', import.meta.url), 'utf8')
  const diff = await readFile(new URL('./components/environments/EnvironmentDiffPanel.vue', import.meta.url), 'utf8')
  assert.match(workspace, /@media \(max-width: 1100px\)[\s\S]*\.repository-flow\.previewing\s*\{[^}]*grid-template-columns:\s*minmax\(0, 1fr\)/)
  assert.match(diff, /@media \(max-width: 1100px\)[\s\S]*\.diff-panel\s*\{[^}]*border-left:\s*0/)
})

test('top-level navigation stays ahead of page-specific summary content', async () => {
  const app = await readFile(new URL('./App.vue', import.meta.url), 'utf8')
  const header = app.match(/<header class="topbar">([\s\S]*?)<\/header>/)?.[1] || ''
  const brandIndex = header.indexOf('<div class="brand">')
  const tabsIndex = header.indexOf('<div class="tabs">')
  const summaryIndex = header.indexOf('class="summary"')

  assert.ok(brandIndex >= 0 && brandIndex < tabsIndex, 'navigation should follow the brand')
  assert.ok(tabsIndex < summaryIndex, 'page-specific summary must not move the navigation')
  assert.match(app, /\.tabs\s*\{[^}]*flex-shrink:\s*0/)
})
