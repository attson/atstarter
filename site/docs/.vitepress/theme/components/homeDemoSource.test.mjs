import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const source = fs.readFileSync(new URL('./HomeDemo.vue', import.meta.url), 'utf8')
const configSource = fs.readFileSync(new URL('../../config.mjs', import.meta.url), 'utf8')
const pagesWorkflowSource = fs.readFileSync(new URL('../../../../../.github/workflows/pages.yml', import.meta.url), 'utf8')

test('home demo does not render explanatory heading copy', () => {
  assert.equal(source.includes('LIVE MOCK WORKSPACE'), false)
  assert.equal(source.includes('首页直接体验项目启动器'), false)
  assert.equal(source.includes('demo-heading'), false)
})

test('home demo renders the real frontend app instead of a hand-built approximation', () => {
  assert.match(source, /from ['"]\.\.\/\.\.\/\.\.\/\.\.\/\.\.\/frontend\/src\/App\.vue['"]/)
  assert.match(source, /<FrontendApp\s+embedded\s*\/>/)
  assert.equal(source.includes('homeDemoData.mjs'), false)
  assert.equal(source.includes('AppButton.vue'), false)
  assert.equal(source.includes('AppPill.vue'), false)
})

test('home demo imports the real app token and global style sheets', () => {
  assert.match(source, /frontend\/src\/styles\/tokens\.css/)
  assert.match(source, /frontend\/src\/style\.css/)
})

test('vitepress resolves vue from site dependencies for imported app components', () => {
  assert.match(configSource, /alias/)
  assert.match(configSource, /find:\s*\/\^vue\$\/,/)
  assert.match(configSource, /find:\s*\/\^vue\\\/server-renderer\$\/,/)
  assert.match(configSource, /node_modules\/vue/)
})

test('vitepress aliases Wails bindings to site mock modules for the real app demo', () => {
  assert.match(configSource, /wailsjs\\\/go\\\/main\\\/App/)
  assert.match(configSource, /wailsjs\\\/runtime\\\/runtime/)
  assert.match(configSource, /mockWailsApp\.mjs/)
  assert.match(configSource, /mockWailsRuntime\.mjs/)
})

test('pages build installs frontend dependencies used by the real app demo', () => {
  assert.match(pagesWorkflowSource, /frontend\/src\/\*\*/)
  assert.match(pagesWorkflowSource, /cache-dependency-path:\s*\|/)
  assert.match(pagesWorkflowSource, /frontend\/package-lock\.json/)
  assert.match(pagesWorkflowSource, /working-directory:\s*frontend[\s\S]*?run:\s*npm ci/)
  assert.match(pagesWorkflowSource, /working-directory:\s*site[\s\S]*?run:\s*npm ci/)
})
