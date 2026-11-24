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

test('card view icon opens a persistent field menu on right click', () => {
  const fieldMenu = read('./components/ProjectCardFieldMenu.vue')

  assert.match(projectList, /readProjectCardFields/)
  assert.match(projectList, /writeProjectCardFields/)
  assert.match(projectList, /data-test="project-view-card"[\s\S]*@contextmenu\.prevent\.stop="openCardFieldMenu"/)
  assert.match(projectList, /title="卡片视图；右键配置字段"/)
  assert.match(projectList, /aria-label="卡片视图；右键配置字段"/)
  assert.match(projectList, /<ProjectCardFieldMenu[\s\S]*:fields="cardFields"[\s\S]*@toggle="toggleCardField"/)
  assert.match(fieldMenu, /data-test="project-card-field-menu"/)
  assert.match(fieldMenu, /type="checkbox"/)
  assert.match(fieldMenu, /type="checkbox" checked disabled[\s\S]*项目名与运行状态/)
  assert.match(fieldMenu, /document\.addEventListener\('mousedown'/)
  assert.match(fieldMenu, /event\.key === 'Escape'/)
  assert.match(fieldMenu, /@change="emit\('toggle', option\.key\)"/)
  assert.doesNotMatch(fieldMenu, /@change="[^"]*close/)
})

test('card view loads branch metadata without changing list view branch work', () => {
  assert.match(projectList, /GetProjectBranch/)
  assert.match(projectList, /loadVisibleProjectBranches/)
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

test('one card field preference controls every project card metadata row', () => {
  const card = read('./components/ProjectCardContent.vue')

  assert.match(projectList, /:card-fields="cardFields"/)
  assert.match(projectNode, /cardFields: \{ type: Object/)
  assert.match(projectNode, /:card-fields="cardFields"/)
  assert.match(groupItem, /cardFields: \{ type: Object/)
  assert.match(groupItem, /:fields="cardFields"/)
  assert.match(card, /fields: \{ type: Object/)
  assert.match(card, /v-else-if="fields\.type"/)
  assert.match(card, /v-if="fields\.branch"[\s\S]*data-test="project-card-branch"/)
  assert.match(card, /v-if="commandName && fields\.command"/)
  assert.match(card, /v-if="fields\.path"[\s\S]*data-test="project-card-path"/)
  assert.match(card, /class="project-name"/)
  assert.match(card, /\['status-dot', statusClass\]/)
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
