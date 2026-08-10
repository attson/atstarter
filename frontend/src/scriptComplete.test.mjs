import test from 'node:test'
import assert from 'node:assert/strict'
import { scriptCompleteContext, filterScripts, applyScript } from './scriptComplete.js'

test('scriptCompleteContext 识别四种包管理器的 run 子命令', () => {
  for (const pm of ['npm', 'pnpm', 'yarn', 'bun']) {
    assert.deepEqual(scriptCompleteContext(`${pm} run `), { head: `${pm} run `, prefix: '' })
    assert.deepEqual(scriptCompleteContext(`${pm} run de`), { head: `${pm} run `, prefix: 'de' })
  }
})

test('scriptCompleteContext 保留原始空白,只把尾部 token 当前缀', () => {
  assert.deepEqual(
    scriptCompleteContext('  pnpm   run   dev'),
    { head: '  pnpm   run   ', prefix: 'dev' },
  )
})

test('scriptCompleteContext 对非补全场景返回 null', () => {
  const cases = [
    '',
    '   ',
    'npm',
    'npm ',
    'npm run dev --watch', // 脚本名后已有第二个 token,认为写完了
    'npm run dev ',
    'npm install',
    'npm i',
    'pnpm dev', // 省略 run 的写法不在本次范围内
    'go run main.go',
    'yarnpkg run dev', // 前缀必须是完整 token
    'npx run dev',
  ]
  for (const line of cases) {
    assert.equal(scriptCompleteContext(line), null, `期望 ${JSON.stringify(line)} 不触发补全`)
  }
})

test('scriptCompleteContext 大小写不敏感地匹配包管理器', () => {
  assert.deepEqual(scriptCompleteContext('NPM RUN de'), { head: 'NPM RUN ', prefix: 'de' })
})

const SCRIPTS = [
  { name: 'build', script: 'vite build' },
  { name: 'dev', script: 'vite' },
  { name: 'install:pods', script: 'cd ios && pod install' },
  { name: 'ios:open', script: 'react-native run-ios' },
]

test('filterScripts 空前缀返回全部,顺序不变', () => {
  assert.deepEqual(filterScripts(SCRIPTS, ''), SCRIPTS)
})

test('filterScripts 前缀匹配排在子串匹配之前,组内保持原顺序', () => {
  // build 里的 "i" 属于子串命中,排在两个前缀命中之后。
  assert.deepEqual(
    filterScripts(SCRIPTS, 'i').map((s) => s.name),
    ['install:pods', 'ios:open', 'build'],
  )
  assert.deepEqual(
    filterScripts(SCRIPTS, 'os').map((s) => s.name),
    ['ios:open'],
  )
  assert.deepEqual(
    filterScripts(SCRIPTS, 'in').map((s) => s.name),
    ['install:pods'],
  )
})

test('filterScripts 前缀命中优先于同样匹配的子串命中', () => {
  const scripts = [
    { name: 'test:unit', script: 'vitest' },
    { name: 'e2e-test', script: 'playwright' },
  ]
  assert.deepEqual(
    filterScripts(scripts, 'test').map((s) => s.name),
    ['test:unit', 'e2e-test'],
  )
})

test('filterScripts 大小写不敏感', () => {
  assert.deepEqual(filterScripts(SCRIPTS, 'IOS').map((s) => s.name), ['ios:open'])
})

test('filterScripts 无命中返回空数组', () => {
  assert.deepEqual(filterScripts(SCRIPTS, 'zzz'), [])
  assert.deepEqual(filterScripts([], 'a'), [])
  assert.deepEqual(filterScripts(null, 'a'), [])
})

test('applyScript 只替换尾部脚本名,保留原始空白', () => {
  assert.equal(applyScript('npm run i', 'ios:open'), 'npm run ios:open')
  assert.equal(applyScript('npm run ', 'dev'), 'npm run dev')
  assert.equal(applyScript('  pnpm   run   de', 'dev'), '  pnpm   run   dev')
})

test('applyScript 对非补全场景原样返回', () => {
  assert.equal(applyScript('go run main.go', 'dev'), 'go run main.go')
})
