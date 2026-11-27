import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { createI18n, useI18n } from 'vue-i18n'
import * as icons from 'lucide-vue-next'
import { DEFAULT_PROJECT_CARD_FIELDS } from '../projectNavigation.js'
import { buildProjectTree } from '../projectTree.js'
import { typeLabel } from '../typeLabel.js'
import { compileComponent, mountVueComponent } from '../testUtils/vueComponentHarness.mjs'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')

for (const name of ['ProjectTreeNode', 'GroupTreeNode']) {
  test(`${name} translates the root directory fallback while preserving real directory names`, async () => {
    const node = buildProjectTree([{ id: 'root-project', name: '原始 @ {name}|', path: '/app' }])[0]
    const component = compileComponent(read(`../components/${name}.vue`), {
      vue, 'vue-i18n': { useI18n }, 'lucide-vue-next': icons, '../typeLabel.js': { typeLabel },
      '../projectNavigation.js': { DEFAULT_PROJECT_CARD_FIELDS },
      './ui/AppIcon.vue': compileComponent(read('../components/ui/AppIcon.vue'), { vue }),
    })
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
    const host = mountVueComponent(component, {
      props: { node, level: 0, expandedDirs: { [node.id]: false }, checked: {} }, i18n,
      beforeMount: ({ app }) => app.component(name, component),
    })
    try {
      assert.equal(host.find((element) => element.props.class === 'dir-name').text, 'Projects')
      i18n.global.locale.value = 'zh-CN'
      await vue.nextTick()
      assert.equal(host.find((element) => element.props.class === 'dir-name').text, '项目')
      const raw = buildProjectTree([{ id: 'nested', name: 'app', path: '/Projects/app' }])[0]
      host.props.node = raw
      host.props.expandedDirs = { [raw.id]: false }
      await vue.nextTick()
      assert.equal(host.find((element) => element.props.class === 'dir-name').text, 'Projects')
      assert.equal(node.id, 'dir:Projects', 'localization must preserve existing expansion/group IDs')
    } finally { host.unmount() }
  })
}

for (const name of ['ProjectTreeNode', 'GroupTreeNode']) {
  test(`${name} translates the detector unknown sentinel without rewriting detected types`, async () => {
    const component = compileComponent(read(`../components/${name}.vue`), {
      vue, 'vue-i18n': { useI18n }, 'lucide-vue-next': icons, '../typeLabel.js': { typeLabel },
      '../projectNavigation.js': { DEFAULT_PROJECT_CARD_FIELDS },
      './ui/AppIcon.vue': compileComponent(read('../components/ui/AppIcon.vue'), { vue }),
    })
    const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
    const host = mountVueComponent(component, {
      props: { node: { type: 'project', project: { id: 'raw-id', name: 'raw name', detectedType: 'unknown' } }, level: 0, expandedDirs: {}, checked: {} }, i18n,
      beforeMount: ({ app }) => app.component(name, component),
    })
    try {
      for (const [locale, fallback] of [['en', 'Unknown'], ['zh-CN', '未知']]) {
        i18n.global.locale.value = locale
        for (const [detectedType, expected] of [
          ['unknown', fallback], ['', fallback], [undefined, fallback],
          ['node-pnpm', 'pnpm'], ['java-maven', 'maven'], ['future-tool @ {name}|原始', 'future-tool @ {name}|原始'],
        ]) {
          host.props.node.project.detectedType = detectedType
          await vue.nextTick()
          assert.equal(host.find((element) => element.props.class === 'type-pill').text, expected)
          assert.equal(host.props.node.project.detectedType, detectedType)
        }
      }
    } finally { host.unmount() }
  })
}
