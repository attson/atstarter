import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const read = (name) => fs.readFileSync(new URL(name, import.meta.url), 'utf8')

const projectList = read('./components/ProjectList.vue')
const projectNode = read('./components/ProjectTreeNode.vue')
const groupItem = read('./components/GroupTreeItem.vue')
const projectDetail = read('./components/ProjectDetail.vue')
const branchSwitcher = read('./components/BranchSwitcher.vue')
const app = read('./App.vue')

test('project navigation exposes persistent list and card view controls', () => {
  assert.match(projectList, /data-test="project-view-list"/)
  assert.match(projectList, /data-test="project-view-card"/)
  assert.match(projectList, /readProjectListView/)
  assert.match(projectList, /writeProjectListView/)
})

test('card view loads branch metadata without changing list view branch work', () => {
  assert.match(projectList, /GetProjectBranch/)
  assert.match(projectList, /loadProjectBranches/)
  assert.match(projectList, /viewMode\.value !== 'card'/)
})

test('tree and group project cards share the full metadata component', () => {
  assert.match(projectNode, /<ProjectCardContent/)
  assert.match(projectNode, /:branch-entry="branches\[node\.project\.id\]"/)
  assert.match(groupItem, /<ProjectCardContent/)
  assert.match(groupItem, /:command-name="member\.command\.name"/)
  assert.match(groupItem, /\.member-row \{\s+box-sizing: border-box;/)

  const card = read('./components/ProjectCardContent.vue')
  assert.match(card, /data-test="project-card-content"/)
  assert.match(card, /data-test="project-card-branch"/)
  assert.match(card, /if \(props\.branchEntry\.state === 'error'\) return '分支读取失败'/)
  assert.match(card, /data-test="project-card-path"/)
  assert.match(card, /:title="project\.name"/)
  assert.match(card, /:title="project\.path"/)
})

test('successful branch switches flow back to the matching navigation card', () => {
  assert.match(branchSwitcher, /const projectId = props\.project\?\.id/)
  assert.match(branchSwitcher, /emit\('changed', \{ \.\.\.result, projectId \}\)/)
  assert.match(branchSwitcher, /catch \(err\) \{[\s\S]*if \(props\.project\?\.id === projectId\)/)
  assert.match(projectDetail, /emit\('branch-changed'/)
  assert.match(projectDetail, /projectId: data\.projectId/)
  assert.match(projectDetail, /@changed="onBranchChanged"/)
  assert.match(app, /:branch-update="branchUpdate"/)
  assert.match(app, /@branch-changed="onBranchChanged"/)
})
