import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { AUDITED_UI_FILES, findHardcodedPresentationCopy } from './sourceAudit.js'
import { SOURCE_AUDIT_ALLOWLIST } from './sourceAuditAllowlist.js'

test('finds presentation copy but ignores domain codes and comments', () => {
  const source = `
    <!-- 中文注释 -->
    <button title="设置">Save project</button>
    <script setup>
    const state = 'running'
    const item = { label: '删除项目' }
    error.value = String(err)
    </script>`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'Example.vue'), [
    { line: 3, text: '设置', kind: 'attribute' },
    { line: 3, text: 'Save project', kind: 'template-text' },
    { line: 6, text: '删除项目', kind: 'presentation-field' },
  ])
})

test('finds accessible attributes and visible text across nested template boundaries', () => {
  const source = `<template>
  <template v-if="count > 0">
    <!-- <span>Ignored 中文</span> -->
    <button aria-label="Open project" title="Open > details"> Open &amp; view </button>
    <img alt="Project icon" /><input placeholder="Search projects" />
  </template>
  <span>After nested template</span>
  <div data-testid="中文测试" @click="emit('中文事件')" class="中文类" />
</template>
<style scoped>.x::after { content: '中文 CSS'; }</style>`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'Nested.vue'), [
    { line: 4, text: 'Open project', kind: 'attribute' },
    { line: 4, text: 'Open > details', kind: 'attribute' },
    { line: 4, text: 'Open & view', kind: 'template-text' },
    { line: 5, text: 'Project icon', kind: 'attribute' },
    { line: 5, text: 'Search projects', kind: 'attribute' },
    { line: 7, text: 'After nested template', kind: 'template-text' },
  ])
})

test('finds English presentation fields, assignments, and Han script literals', () => {
  const source = `<script setup>
const actions = [{ label: 'Delete project', text: 'Open', title: 'Details',
  message: 'Failed', description: 'Try again', confirmText: 'Confirm' }]
dialog.title = 'Project settings'
message.value = 'Saved'
const description = ref('Changes take effect later')
const fallback = '无法加载项目'
const item = { 'label': 'Quoted field' }
const text = \`Found \${count} projects\`
</script>`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'Fields.vue').map(({ text }) => text), [
    'Delete project', 'Open', 'Details', 'Failed', 'Try again', 'Confirm',
    'Project settings', 'Saved', 'Changes take effect later', '无法加载项目',
    'Quoted field', 'Found ${count} projects',
  ])
})

test('ignores comments, imports, translation calls, events, domain comparisons, and raw errors', () => {
  const source = `<script setup>
import thing from './中文.js'
export { other } from './另一个.js'
// const title = '注释'
/* const label = '块注释' */
const state = 'running'
const isRunning = state === '运行中'
const isStopped = '已停止' !== state
const event = defineEmits(['中文事件'])
emit('中文事件')
EventsOn('中文事件', handler)
const id = { 'data-testid': '中文测试' }
const title = t('中文键', { default: '中文默认值' })
const message = $t('another.key')
error.value = String(err)
const url = 'https://example.test/中文'
const path = 'C:\\\\中文\\\\file.txt'
const hint = 'https://example.test' // trailing 中文 comment
</script>
<template>
  {{ t('中文键') }} {{ $t('common.save') }} {{ error }}
  <input :title="t('settings.title')" :placeholder="$t('settings.hint')" />
</template>`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'Ignored.vue'), [
    { line: 16, text: 'https://example.test/中文', kind: 'script-string' },
    { line: 17, text: 'C:\\中文\\file.txt', kind: 'script-string' },
  ])
})

test('finds literal presentation expressions without flagging comparison codes', () => {
  const source = `<template>
  <span>{{ state === 'running' ? 'Running' : 'Stopped' }}</span>
  <input :title="ready ? 'Ready' : t('common.wait')" />
  <span>{{ t('common.save') }} project</span>
</template>`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'Expressions.vue'), [
    { line: 2, text: 'Running', kind: 'template-expression' },
    { line: 2, text: 'Stopped', kind: 'template-expression' },
    { line: 3, text: 'Ready', kind: 'attribute' },
    { line: 4, text: 'project', kind: 'template-text' },
  ])
})

test('audits JavaScript helpers but excludes message catalogs', () => {
  const source = "export const item = { label: 'Open project' }"
  assert.deepEqual(findHardcodedPresentationCopy(source, 'projectUi.js'), [
    { line: 1, text: 'Open project', kind: 'presentation-field' },
  ])
  assert.deepEqual(findHardcodedPresentationCopy(source, 'i18n/messages/en.js'), [])
  assert.deepEqual(findHardcodedPresentationCopy(source, 'C:\\repo\\i18n\\messages\\zh-CN.js'), [])
})

test('comparison operands retain nested presentation fields while direct domain literals stay ignored', () => {
  const source = `const same = renderDialog({ title: 'Project settings' }) === previous
const reversed = previous !== renderDialog({ message: 'Try again' })
const active = state === '运行中'
const stopped = '已停止' !== state`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'Comparisons.js'), [
    { line: 1, text: 'Project settings', kind: 'presentation-field' },
    { line: 2, text: 'Try again', kind: 'presentation-field' },
  ])
})

test('computed member lookup keys are structural when forwarding errors and labels', () => {
  const source = `const message = err['message']
const label = item['name']
const title = item?.['title']
const description = item['中文键']
const text = renderDialog({ label: 'Open project' })['name']`
  assert.deepEqual(findHardcodedPresentationCopy(source, 'LookupKeys.js'), [
    { line: 5, text: 'Open project', kind: 'presentation-field' },
  ])
})

test('fails visibly on invalid source instead of silently skipping coverage', () => {
  assert.throws(() => findHardcodedPresentationCopy('<template><div></template>', 'Broken.vue'), /Broken.vue/)
  assert.throws(() => findHardcodedPresentationCopy('const label =', 'Broken.js'), /Broken.js/)
})

test('allowlist entries require exact file/text strings and a concrete reason', () => {
  const seen = new Set()
  for (const entry of SOURCE_AUDIT_ALLOWLIST) {
    assert.deepEqual(Object.keys(entry).sort(), ['file', 'reason', 'text'])
    for (const key of ['file', 'text', 'reason']) {
      assert.equal(typeof entry[key], 'string', `${key} must be an exact string`)
      assert.ok(entry[key].trim(), `${key} must not be empty`)
    }
    assert.ok(!/[*?]/.test(entry.file), `${entry.file}: file globs are forbidden`)
    assert.ok(entry.reason.trim().length >= 10, `${entry.file}: explain the concrete exception`)
    const key = JSON.stringify([entry.file, entry.text])
    assert.ok(!seen.has(key), `${entry.file}: duplicate exception for ${entry.text}`)
    seen.add(key)
  }
})

test('audited UI files have no hardcoded presentation copy without an exact exception', async () => {
  for (const file of [
    'components/environments/CodexEnvironmentFieldEditor.vue',
    'components/environments/EnvironmentDiffPanel.vue',
    'components/environments/EnvironmentFieldEditor.vue',
    'components/environments/EnvironmentOverview.vue',
    'components/environments/EnvironmentPresetManager.vue',
    'components/environments/EnvironmentRepositoryManager.vue',
    'components/environments/EnvironmentRuleManager.vue',
    'components/environments/EnvironmentWorkspace.vue',
    'App.vue', 'components/ConfirmDialog.vue', 'components/DetectionSwitch.vue',
    'components/SettingsDialog.vue', 'components/ui/AppButton.vue',
    'components/ui/AppIcon.vue', 'components/ui/AppPill.vue',
    'components/ui/ThemeToggle.vue', 'components/ui/ContextMenu.vue',
    'components/AddProjectDialog.vue', 'components/EditProjectDialog.vue',
    'components/ProjectList.vue', 'components/ProjectDetail.vue',
    'components/ProjectTreeNode.vue', 'components/GroupDialog.vue',
    'components/GroupDetail.vue', 'components/GroupTreeItem.vue',
    'components/GroupTreeNode.vue', 'components/AddToGroupDialog.vue',
    'components/CommandRow.vue', 'components/BranchSwitcher.vue',
    'components/ScanDialog.vue',
    'components/ComposeDetail.vue', 'components/ContainerPanel.vue',
    'components/LogPanel.vue', 'components/UpdateBanner.vue',
    'components/FileBrowser.vue', 'components/fileExplorer/BinaryBanner.vue',
    'components/fileExplorer/CodeEditor.vue', 'components/fileExplorer/EditorPrefsPanel.vue',
    'components/fileExplorer/EditorStatusBar.vue', 'components/fileExplorer/EditorTabs.vue',
    'components/fileExplorer/EditorToolbar.vue', 'components/fileExplorer/FileConfirmDialog.vue',
    'components/fileExplorer/FileEditor.vue', 'components/fileExplorer/FileTree.vue',
    'components/fileExplorer/FileTreeNode.vue', 'components/fileExplorer/ImagePreview.vue',
    'components/fileExplorer/InlineEditRow.vue', 'components/fileExplorer/MarkdownPreview.vue',
    'components/fileExplorer/MediaPreview.vue', 'components/fileExplorer/PdfPreview.vue',
    'components/fileExplorer/fileClipboard.js', 'components/fileExplorer/previewKind.ts',
    'components/fileExplorer/codeMirrorI18n.js',
  ]) assert.ok(AUDITED_UI_FILES.includes(file), `Missing audit coverage: ${file}`)
  assert.equal(new Set(AUDITED_UI_FILES).size, AUDITED_UI_FILES.length)
  const unmatched = []
  for (const file of AUDITED_UI_FILES) {
    const source = await readFile(new URL(`../${file}`, import.meta.url), 'utf8')
    for (const finding of findHardcodedPresentationCopy(source, file)) {
      if (!SOURCE_AUDIT_ALLOWLIST.some((entry) => entry.file === file && entry.text === finding.text)) {
        unmatched.push(`${file}:${finding.line} [${finding.kind}] ${JSON.stringify(finding.text)}`)
      }
    }
  }
  assert.deepEqual(unmatched, [], `Localize this copy or add a specific, reasoned exception:\n${unmatched.join('\n')}`)
})
