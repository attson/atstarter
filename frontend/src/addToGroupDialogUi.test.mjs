import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { createI18n, useI18n } from 'vue-i18n'
import en from './i18n/messages/en.js'
import zhCN from './i18n/messages/zh-CN.js'
import { compileComponent, mountVueComponent } from './testUtils/vueComponentHarness.mjs'

const read = (name) => fs.readFileSync(new URL(name, import.meta.url), 'utf8')

function mountDialog() {
  // BaseTransition retains Vue's show/hide lifecycle while leaving CSS animation
  // to the browser. The dialog, AppButton, v-model, and i18n remain real.
  const component = compileComponent(read('./components/AddToGroupDialog.vue'), {
    vue: { ...vue, Transition: vue.BaseTransition },
    'vue-i18n': { useI18n },
    './ui/AppButton.vue': compileComponent(read('./components/ui/AppButton.vue'), { vue }),
  })
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const saves = []
  const host = mountVueComponent(component, {
    i18n,
    props: {
      show: false,
      groups: [],
      project: { id: 'raw-project', name: 'Raw项目' },
      command: { id: 'default', name: 'Default', command: 'pnpm', args: ['run', 'dev'] },
      onSave: (value) => saves.push(value),
    },
  })
  return {
    ...host, i18n, saves,
    input: () => host.find((element) => element.type === 'input'),
    save: () => {
      const submit = host.find((element) => element.type === 'button' && element.props.class?.includes('variant-primary'))
      assert.ok(submit, 'the Add button must be visible')
      submit.props.onClick(new Event('click'))
    },
  }
}

test('untouched group suggestion follows locale and saves the visible generated name', async () => {
  const { props, i18n, input, save, saves, unmount } = mountDialog()
  try {
    props.show = true
    await vue.nextTick()
    assert.equal(input().value, 'Raw项目 group')
    i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.equal(input().value, 'Raw项目 分组')
    save()
    assert.deepEqual(saves.at(-1), { mode: 'new', groupId: '', groupName: 'Raw项目 分组' })
    i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.equal(input().value, 'Raw项目 group')
    save()
    assert.equal(saves.at(-1).groupName, 'Raw项目 group')
  } finally { unmount() }
})

test('edited names stay verbatim across locale changes and save without trimming', async () => {
  const { props, i18n, input, save, saves, unmount } = mountDialog()
  try {
    props.show = true
    await vue.nextTick()
    for (const raw of ['  自定组\t group  ', 'Raw项目 group']) {
      i18n.global.locale.value = 'en'
      await vue.nextTick()
      input().value = raw
      input().dispatchEvent(new Event('input'))
      await vue.nextTick()
      i18n.global.locale.value = 'zh-CN'
      await vue.nextTick()
      assert.equal(input().value, raw)
      save()
      assert.deepEqual(saves.at(-1), { mode: 'new', groupId: '', groupName: raw })
    }
    props.show = false
    await vue.nextTick()
    props.show = true
    await vue.nextTick()
    assert.equal(input().value, 'Raw项目 分组', 'reopening restores a fresh translated suggestion')
  } finally { unmount() }
})
