import assert from 'node:assert/strict'
import test from 'node:test'
import * as gitBranches from './gitBranches.js'
import {
  allBranchNames,
  branchGroups,
  branchPillLabel,
  canCreateFromQuery,
  checkoutWarnings,
  validateBranchName,
} from './gitBranches.js'

const data = {
  repo: true,
  branch: 'main',
  detached: false,
  dirty: false,
  head: 'abc1234',
  local: ['main', 'feature/editor'],
  remote: ['feature/docs', 'hotfix'],
}

test('branchGroups 分本地与远端两组并标出当前分支', () => {
  const groups = branchGroups(data)
  assert.deepEqual(groups.map((g) => g.id), ['local', 'remote'])
  assert.equal(Object.hasOwn(groups[0], 'label'), false)
  assert.equal(Object.hasOwn(groups[1], 'label'), false)
  const main = groups[0].items.find((i) => i.name === 'main')
  assert.equal(main.current, true)
  assert.equal(groups[0].items.find((i) => i.name === 'feature/editor').current, false)
  assert.equal(groups[1].items[0].kind, 'remote')
})

test('branchGroups 按查询词过滤,空组不出现', () => {
  const groups = branchGroups(data, 'FEATURE')
  assert.deepEqual(groups.map((g) => g.id), ['local', 'remote'])
  assert.deepEqual(groups[0].items.map((i) => i.name), ['feature/editor'])
  assert.deepEqual(groups[1].items.map((i) => i.name), ['feature/docs'])

  const onlyRemote = branchGroups(data, 'hotfix')
  assert.deepEqual(onlyRemote.map((g) => g.id), ['remote'])

  assert.deepEqual(branchGroups(data, 'nothing-matches'), [])
})

test('branchGroups 容忍空数据', () => {
  assert.deepEqual(branchGroups(null), [])
  assert.deepEqual(branchGroups({}), [])
})

test('branchPillLabel 区分普通分支、detached 与非仓库', () => {
  assert.equal(branchPillLabel(data), 'main')
  assert.equal(branchPillLabel({ repo: true, detached: true, head: 'abc1234' }), 'detached @ abc1234')
  assert.equal(branchPillLabel({ repo: true, detached: true }), 'detached')
  assert.equal(branchPillLabel({ repo: false }), '')
  assert.equal(branchPillLabel(null), '')
})

test('validateBranchName 挡住 git 会拒绝的名字', () => {
  assert.equal(validateBranchName('feature/ok'), '')
  for (const [name, code] of [
    ['', 'empty'], ['   ', 'empty'], ['-f', 'leadingDash'],
    ['has space', 'invalidCharacters'], ['sta*r', 'invalidCharacters'],
    ['/lead', 'slash'], ['trail/', 'slash'], ['a//b', 'slash'],
    ['.hidden', 'dot'], ['a..b', 'dot'], ['x.lock', 'dot'], ['a@{0}', 'reflog'],
  ]) {
    assert.deepEqual(validateBranchName(name), { code, values: {} })
  }
})

test('validateBranchName 拒绝重名', () => {
  assert.deepEqual(validateBranchName('main', ['main']), { code: 'exists', values: { name: 'main' } })
  assert.deepEqual(validateBranchName(' feature/原始 ', ['feature/原始']), {
    code: 'exists', values: { name: 'feature/原始' },
  })
  assert.equal(validateBranchName('main', ['other']), '')
})

test('allBranchNames 只把本地分支算作已占用', () => {
  assert.deepEqual(allBranchNames(data), ['main', 'feature/editor'])
})

test('canCreateFromQuery 只在输入了合法新名字时才成立', () => {
  assert.equal(canCreateFromQuery(data, ''), false)
  assert.equal(canCreateFromQuery(data, 'main'), false, '本地已有同名分支')
  assert.equal(canCreateFromQuery(data, 'bad name'), false)
  assert.equal(canCreateFromQuery(data, 'feature/new'), true)
  assert.equal(canCreateFromQuery(data, 'feature/docs'), true, '远端同名不占用本地新建')
})

test('checkoutWarnings 提醒脏工作区与运行中的项目', () => {
  assert.deepEqual(checkoutWarnings(data, false), [])
  assert.deepEqual(checkoutWarnings({ ...data, dirty: true }, false), ['dirty'])
  assert.deepEqual(checkoutWarnings(data, true), ['running'])
  assert.deepEqual(checkoutWarnings({ ...data, dirty: true }, true), ['dirty', 'running'])
})

test('gitErrorDetail preserves string rejections and error messages byte for byte', () => {
  const { gitErrorDetail } = gitBranches
  assert.equal(typeof gitErrorDetail, 'function')
  const raw = '  fatal: 分支 feature/原始\nYour local changes would be overwritten.\n'
  assert.equal(gitErrorDetail(raw), raw)
  assert.equal(gitErrorDetail(new Error(raw)), raw)
  assert.equal(gitErrorDetail({ message: raw }), raw)
  for (const error of [null, undefined, '', new Error(''), {}]) {
    assert.equal(gitErrorDetail(error), '')
  }
})
