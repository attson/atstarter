import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const source = fs.readFileSync(new URL('./HomeDemo.vue', import.meta.url), 'utf8')
const themeSource = fs.readFileSync(new URL('../index.js', import.meta.url), 'utf8')

test('home demo does not render explanatory heading copy', () => {
  assert.equal(source.includes('LIVE MOCK WORKSPACE'), false)
  assert.equal(source.includes('首页直接体验项目启动器'), false)
  assert.equal(source.includes('demo-heading'), false)
})

test('home demo uses app workspace tokens and shared UI primitives', () => {
  assert.match(source, /from ['"]\.\.\/\.\.\/\.\.\/\.\.\/\.\.\/frontend\/src\/components\/ui\/AppButton\.vue['"]/)
  assert.match(source, /from ['"]\.\.\/\.\.\/\.\.\/\.\.\/\.\.\/frontend\/src\/components\/ui\/AppPill\.vue['"]/)
  assert.match(source, /class="app-shell/)
  assert.match(source, /class="project-list/)
  assert.match(source, /class="project-header/)
})

test('site theme does not import app global html styles', () => {
  assert.equal(themeSource.includes('frontend/src/styles/tokens.css'), false)
})
