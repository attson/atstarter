import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { createI18n, useI18n } from 'vue-i18n'
import * as icons from 'lucide-vue-next'
import { DEFAULT_PROJECT_CARD_FIELDS } from '../projectNavigation.js'
import { displayPath } from '../projectTree.js'
import { typeLabel } from '../typeLabel.js'
import { compileComponent, mountVueComponent } from '../testUtils/vueComponentHarness.mjs'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')

function mountProjectComponent(name, props) {
  const dependencies = {
    vue,
    'vue-i18n': { useI18n },
    'lucide-vue-next': icons,
    './ui/AppIcon.vue': compileComponent(read('../components/ui/AppIcon.vue'), { vue }),
    '../projectNavigation.js': { DEFAULT_PROJECT_CARD_FIELDS },
    '../projectTree.js': { displayPath },
    '../typeLabel.js': { typeLabel },
  }
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const component = compileComponent(read(`../components/${name}.vue`), dependencies)
  return { ...mountVueComponent(component, { props, i18n }), i18n }
}

test('project card presentation follows locale while project data stays raw', async () => {
  const project = {
    id: 'raw-id',
    name: '原始 project @ {name}|',
    path: '/workspace/原始 project',
    detectedType: 'unknown',
  }
  const host = mountProjectComponent('ProjectCardContent', {
    project,
    status: { State: 'running' },
    branchEntry: { state: 'loading', branch: '' },
    missing: true,
    fields: { ...DEFAULT_PROJECT_CARD_FIELDS },
  })
  try {
    assert.match(host.visibleText(), /Reading branch/)
    assert.match(host.visibleText(), /Missing/)
    host.props.missing = false
    host.props.branchEntry = { state: 'error', branch: '' }
    await vue.nextTick()
    assert.match(host.visibleText(), /Failed to read branch/)
    assert.match(host.visibleText(), /Unknown/)
    host.props.branchEntry = { state: 'ready', branch: '' }
    await vue.nextTick()
    assert.match(host.visibleText(), /No branch information/)

    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /无分支信息/)
    assert.match(host.visibleText(), /未知/)

    const rawBranch = 'feature/原始 @ {branch}|'
    host.props.branchEntry = { state: 'ready', branch: rawBranch }
    await vue.nextTick()
    assert.ok(host.visibleText().includes(project.name))
    assert.ok(host.visibleText().includes(rawBranch))
    assert.equal(host.props.project.path, project.path)
    assert.equal(host.props.project.detectedType, 'unknown')
  } finally { host.unmount() }
})

test('project card field menu labels and accessible name follow locale', async () => {
  const host = mountProjectComponent('ProjectCardFieldMenu', {
    x: 10,
    y: 20,
    fields: { ...DEFAULT_PROJECT_CARD_FIELDS },
  })
  try {
    assert.match(host.visibleText(), /Display fields/)
    assert.match(host.visibleText(), /Project name and status/)
    assert.match(host.visibleText(), /Project type/)
    assert.match(host.visibleText(), /Launch command/)
    assert.ok(host.find((element) => element.props['aria-label'] === 'Card display fields' && element.props.role === 'group'))

    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /显示字段/)
    assert.match(host.visibleText(), /项目名与运行状态/)
    assert.match(host.visibleText(), /项目类型/)
    assert.match(host.visibleText(), /启动命令/)
    assert.ok(host.find((element) => element.props['aria-label'] === '卡片显示字段' && element.props.role === 'group'))
  } finally { host.unmount() }
})

test('group member cards preserve the missing-project state in both locales', async () => {
  const dependencies = {
    vue,
    'vue-i18n': { useI18n },
    'lucide-vue-next': icons,
    './ui/AppIcon.vue': compileComponent(read('../components/ui/AppIcon.vue'), { vue }),
    '../projectNavigation.js': { DEFAULT_PROJECT_CARD_FIELDS },
    '../projectTree.js': { displayPath },
    '../typeLabel.js': { typeLabel },
  }
  const card = compileComponent(read('../components/ProjectCardContent.vue'), dependencies)
  const group = compileComponent(read('../components/GroupTreeItem.vue'), {
    ...dependencies,
    './ProjectCardContent.vue': card,
  })
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const project = {
    id: 'missing-project',
    name: 'raw project',
    path: '/workspace/raw-project',
    detectedType: 'go',
    commands: [{ id: 'dev', name: 'raw command' }],
  }
  const host = mountVueComponent(group, {
    props: {
      group: { id: 'group', name: 'raw group', items: [{ projectId: project.id, commandId: 'dev' }] },
      projects: [project],
      statuses: {},
      expanded: true,
      viewMode: 'card',
      missingIds: new Set([project.id]),
    },
    i18n,
  })
  try {
    assert.match(host.visibleText(), /Missing/)
    i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /缺失/)
    assert.ok(host.visibleText().includes(project.name))
    assert.ok(host.visibleText().includes(project.commands[0].name))
  } finally { host.unmount() }
})
