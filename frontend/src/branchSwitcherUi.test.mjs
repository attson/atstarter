// 源码级断言:守住分支切换器的接线点(纯 node,不挂载组件)。
import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const read = (name) => fs.readFileSync(new URL(name, import.meta.url), 'utf8')

const switcher = read('./components/BranchSwitcher.vue')
const projectDetail = read('./components/ProjectDetail.vue')

test('分支标签变成可点的切换入口', () => {
  assert.match(switcher, /data-test="branch-pill"/)
  assert.match(switcher, /data-test="branch-menu"/)
  assert.match(switcher, /clickable/)
})

test('切换器接的是三个 git 绑定', () => {
  assert.match(switcher, /ListProjectBranches/)
  assert.match(switcher, /CheckoutProjectBranch/)
  assert.match(switcher, /CreateProjectBranch/)
})

test('分支列表每次打开都重新拉,避免展示过期数据', () => {
  assert.match(switcher, /async function toggleOpen/)
  assert.match(switcher, /await load\(\)/)
})

test('git 的错误消息原样展示,不包装成泛化文案', () => {
  assert.match(switcher, /data-test="branch-error"/)
  assert.match(switcher, /err\?\.message \|\| '切换分支失败'/)
})

test('脏工作区与运行中项目会给出提醒', () => {
  assert.match(switcher, /checkoutWarnings/)
  assert.match(switcher, /branch-warnings/)
})

test('搜索框里输入新名字可以直接建分支', () => {
  assert.match(switcher, /data-test="branch-create"/)
  assert.match(switcher, /canCreateFromQuery/)
  assert.match(switcher, /validateBranchName/)
})

test('ProjectDetail 用切换器替掉了只读的分支 pill', () => {
  assert.match(projectDetail, /<BranchSwitcher\s+:project="project"\s+:running="state === 'running'"/)
  assert.doesNotMatch(projectDetail, /GetProjectBranch/, '分支读取已经归切换器管')
  assert.doesNotMatch(projectDetail, /refreshBranch/)
})
