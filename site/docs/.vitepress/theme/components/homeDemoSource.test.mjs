import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const source = fs.readFileSync(new URL('./HomeDemo.vue', import.meta.url), 'utf8')
const configSource = fs.readFileSync(new URL('../../config.mjs', import.meta.url), 'utf8')
const customCss = fs.readFileSync(new URL('../custom.css', import.meta.url), 'utf8')
const fileEditorSource = fs.readFileSync(new URL('../../../../../frontend/src/components/fileExplorer/FileEditor.vue', import.meta.url), 'utf8')
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

test('vitepress uses the product icon for the published site tab', () => {
  assert.match(configSource, /head:\s*\[/)
  assert.match(configSource, /rel:\s*['"]icon/)
  assert.match(configSource, /href:\s*['"]\/atstarter\/atstarter-icon\.svg/)
})

test('hero action button text remains visible over the custom brand gradient', () => {
  assert.match(customCss, /\.VPHome\s+\.VPHero\s+\.actions\s+\.VPButton\.brand\s+\.text/)
  assert.match(customCss, /-webkit-text-fill-color:\s*#fff/)
  assert.match(customCss, /\.VPHome\s+\.VPHero\s+\.actions\s+\.VPButton\.brand\s*\{[\s\S]*?-webkit-text-fill-color:\s*#fff/)
})

test('vitepress dedupes CodeMirror packages used by the embedded frontend file editor', () => {
  assert.match(configSource, /dedupe:\s*\[[\s\S]*?@codemirror\/state/)
  assert.match(configSource, /find:\s*\/\^@codemirror\\\//)
  assert.match(configSource, /frontend\/node_modules\/@codemirror/)
})

test('embedded file editor loads CodeMirror through an async component chunk', () => {
  assert.match(fileEditorSource, /defineAsyncComponent/)
  assert.match(fileEditorSource, /import\(["']\.\/CodeEditor\.vue["']\)/)
  assert.doesNotMatch(fileEditorSource, /import\s+CodeEditor\s+from\s+["']\.\/CodeEditor\.vue["']/)
})

test('pages build installs frontend dependencies used by the real app demo', () => {
  assert.match(pagesWorkflowSource, /frontend\/src\/\*\*/)
  assert.match(pagesWorkflowSource, /cache-dependency-path:\s*\|/)
  assert.match(pagesWorkflowSource, /frontend\/package-lock\.json/)
  assert.match(pagesWorkflowSource, /working-directory:\s*frontend[\s\S]*?run:\s*npm ci/)
  assert.match(pagesWorkflowSource, /working-directory:\s*site[\s\S]*?run:\s*npm ci/)
  assert.match(pagesWorkflowSource, /npm run docs:build[\s\S]*?homeDemoBundle\.test\.mjs/)
})
