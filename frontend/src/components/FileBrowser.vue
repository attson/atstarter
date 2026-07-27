<script setup>
import { ref, watch, onMounted, onBeforeUnmount, nextTick, shallowRef } from 'vue'
import { FileTree, prepareFileTreeInput } from '@pierre/trees'
import { VirtualizedFile, Virtualizer, DIFFS_TAG_NAME } from '@pierre/diffs'
import { Editor } from '@pierre/diffs/edit'
import { WalkProjectPaths, ReadProjectFile, WriteProjectFile } from '../../wailsjs/go/main/App'
import { useTheme } from '../composables/useTheme'

const props = defineProps({ projectId: { type: String, required: true } })

const { resolvedTheme } = useTheme()

// DOM 挂载点:左树 / 右预览。
const treeMount = ref(null)
const previewMount = ref(null)

const treeError = ref('')
const truncated = ref(false)
const selectedPath = ref('')
const previewError = ref('')
const previewNote = ref('') // 二进制 / 大文件截断提示
const loadingTree = ref(false)
const loadingPreview = ref(false)

// 编辑态。
const canEdit = ref(false)   // 当前文件是否「可完整读取的文本」(可编辑)
const isEditing = ref(false)
const dirty = ref(false)
const saving = ref(false)
const saveError = ref('')

// pierre 实例(非响应式,用 shallowRef 存句柄)。
const tree = shallowRef(null)
const diffFile = shallowRef(null)     // 当前 VirtualizedFile 实例
const virtualizer = shallowRef(null)  // 虚拟化器(按可视区渲染,避免大文件全量 DOM 卡死)
const editor = shallowRef(null)

// @pierre/diffs 主题:light/dark 各挑一个 Shiki 内置主题。
const DIFF_THEMES = { light: 'github-light', dark: 'github-dark' }

function destroyTree() {
  if (tree.value) {
    try { tree.value.cleanUp() } catch {}
    tree.value = null
  }
  if (treeMount.value) treeMount.value.innerHTML = ''
}

function destroyEditor() {
  if (editor.value) {
    try { editor.value.cleanUp() } catch {}
    editor.value = null
  }
  isEditing.value = false
  dirty.value = false
  saving.value = false
  saveError.value = ''
}

function destroyPreview() {
  destroyEditor()
  if (diffFile.value) {
    try { diffFile.value.cleanUp?.() } catch {}
    diffFile.value = null
  }
  if (virtualizer.value) {
    try { virtualizer.value.cleanUp?.() } catch {}
    virtualizer.value = null
  }
  if (previewMount.value) previewMount.value.innerHTML = ''
  canEdit.value = false
}

async function loadTree() {
  destroyTree()
  destroyPreview()
  treeError.value = ''
  truncated.value = false
  selectedPath.value = ''
  previewError.value = ''
  previewNote.value = ''
  if (!props.projectId || !treeMount.value) return

  loadingTree.value = true
  let result
  try {
    result = await WalkProjectPaths(props.projectId)
  } catch (e) {
    treeError.value = String(e)
    return
  } finally {
    loadingTree.value = false
  }
  // 切项目竞态:响应回来时 projectId 已变则丢弃。
  if (!treeMount.value) return

  truncated.value = !!result.truncated
  const paths = result.paths || []
  // 后端 WalkDir 深度优先产出交错顺序,交给 pierre 默认排序(目录优先 + 自然排序)。
  const preparedInput = prepareFileTreeInput(paths)

  const instance = new FileTree({
    preparedInput,
    search: true,
    initialExpansion: 'closed',
    onSelectionChange: (selectedPaths) => onSelect(selectedPaths),
  })
  instance.render({ containerWrapper: treeMount.value })
  tree.value = instance
}

async function onSelect(selectedPaths) {
  const path = selectedPaths && selectedPaths.length ? selectedPaths[0] : ''
  // 目录路径带尾 /,不预览。
  if (!path || path.endsWith('/')) return
  selectedPath.value = path
  previewError.value = ''
  previewNote.value = ''
  loadingPreview.value = true
  destroyPreview()

  let content
  try {
    content = await ReadProjectFile(props.projectId, path)
  } catch (e) {
    if (selectedPath.value !== path) return
    previewError.value = String(e)
    loadingPreview.value = false
    return
  }
  if (selectedPath.value !== path) return // 已切到别的文件
  loadingPreview.value = false

  if (content.binary) {
    previewNote.value = '二进制文件,不预览。'
    return
  }
  if (content.truncated) {
    previewNote.value = '文件较大,仅显示前 1MB。'
  }
  // 仅「可完整读取的文本」可编辑:非二进制且未截断。
  canEdit.value = !content.binary && !content.truncated
  // 等 previewMount 的 DOM 随状态更新挂载出来(它在 v-else 分支里,ref 需下一 tick 才绑定)。
  await nextTick()
  if (selectedPath.value !== path) return
  renderPreview(path, content.content || '')
}

function renderPreview(path, text) {
  if (!previewMount.value) return
  const filename = path.split('/').pop() || path
  const themeType = resolvedTheme.value === 'dark' ? 'dark' : 'light'

  // 用 VirtualizedFile 按可视区渲染,避免大文件(如 3 万行 openapi.json)全量 DOM
  // 导致主线程冻结数秒。Virtualizer 的滚动 root 必须是实际滚动容器 previewMount。
  const vz = new Virtualizer()
  vz.setup(previewMount.value)
  const fileContainer = document.createElement(DIFFS_TAG_NAME)
  previewMount.value.appendChild(fileContainer)

  const instance = new VirtualizedFile({
    theme: DIFF_THEMES,
    themeType,
    lineNumbers: true,
    disableFileHeader: true,
  }, vz)
  instance.render({
    // diffs 运行时按 file.name 推断语言(其 .d.ts 写的是 filename,beta 版名实不符),两者都给。
    file: { name: filename, filename, contents: text },
    fileContainer,
  })
  virtualizer.value = vz
  diffFile.value = instance
}

// 进入编辑:对当前预览的 File 实例套 pierre Editor(编辑时实时高亮 + 撤销栈)。
function startEdit() {
  if (!canEdit.value || !diffFile.value || isEditing.value) return
  saveError.value = ''
  dirty.value = false
  const instance = new Editor({
    onChange: () => { dirty.value = true },
    // Wails 桌面环境:优先用浏览器原生剪贴板,失败静默(pierre 有内部兜底)。
    clipboard: {
      readText: async () => {
        try { return await navigator.clipboard.readText() } catch { return '' }
      },
    },
  })
  try {
    instance.edit(diffFile.value)
  } catch (e) {
    saveError.value = '无法进入编辑:' + String(e)
    try { instance.cleanUp() } catch {}
    return
  }
  editor.value = instance
  isEditing.value = true
}

// 退出编辑回到只读预览:editor.edit() 接管的是当前 diffFile 实例,退出时
// 必须把 editor + 旧 diffFile + virtualizer + mount DOM 全部清干净再重建,
// 否则残留 <diffs-container>。
async function restorePreview(path, text) {
  destroyEditor()
  if (diffFile.value) {
    try { diffFile.value.cleanUp?.() } catch {}
    diffFile.value = null
  }
  if (virtualizer.value) {
    try { virtualizer.value.cleanUp?.() } catch {}
    virtualizer.value = null
  }
  if (previewMount.value) previewMount.value.innerHTML = ''
  await nextTick()
  if (selectedPath.value !== path || !previewMount.value) return
  renderPreview(path, text)
}

// 保存:取编辑器当前文本写回磁盘,成功后回到只读预览并刷新。
async function saveEdit() {
  if (!editor.value || saving.value) return
  const path = selectedPath.value
  const text = editor.value.getText()
  saving.value = true
  saveError.value = ''
  try {
    await WriteProjectFile(props.projectId, path, text)
  } catch (e) {
    saving.value = false
    saveError.value = '保存失败:' + String(e)
    return
  }
  // 切文件竞态:保存返回时已切走则丢弃编辑器,不再刷新预览(切文件已重建预览)。
  if (selectedPath.value !== path) { destroyEditor(); return }
  await restorePreview(path, text)
}

// 取消:丢弃改动回到只读预览(重新渲染原始内容)。
function cancelEdit() {
  if (!editor.value) return
  const path = selectedPath.value
  // 编辑器当前文本可能已改;取消要显示磁盘原文,重读一次(轻量,文件已知可完整读取)。
  ReadProjectFile(props.projectId, path)
    .then((c) => { if (selectedPath.value === path) restorePreview(path, (c && c.content) || '') })
    .catch(() => { destroyEditor() })
}

// 编辑态下的保存快捷键:Ctrl/Cmd+S。
function onKeydown(e) {
  if (isEditing.value && (e.ctrlKey || e.metaKey) && (e.key === 's' || e.key === 'S')) {
    e.preventDefault()
    saveEdit()
  }
}

// 主题切换:通知 diffs 组件切明暗(trees 用 CSS var 联动 App 的 data-theme,见 style)。
watch(resolvedTheme, (val) => {
  const themeType = val === 'dark' ? 'dark' : 'light'
  if (diffFile.value && diffFile.value.setThemeType) {
    try { diffFile.value.setThemeType(themeType) } catch {}
  }
})

// 首次加载放到 onMounted:此时 treeMount 的 DOM 才存在(watch immediate 会早于挂载)。
onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  await nextTick()
  loadTree()
})

// projectId 变化时重载(切换项目)。不用 immediate,首次由 onMounted 负责。
watch(() => props.projectId, async () => {
  await nextTick()
  loadTree()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  destroyTree()
  destroyPreview()
})
</script>

<template>
  <div class="file-browser">
    <div class="tree-col">
      <div v-if="treeError" class="msg error">{{ treeError }}</div>
      <div v-else-if="loadingTree" class="msg">加载文件树…</div>
      <div v-if="truncated" class="msg warn">文件过多,仅显示前 50000 项。</div>
      <div ref="treeMount" class="tree-mount"></div>
    </div>
    <div class="preview-col">
      <div v-if="loadingPreview" class="msg">加载中…</div>
      <div v-else-if="previewError" class="msg error">{{ previewError }}</div>
      <div v-else-if="!selectedPath" class="msg">从左侧选择文件查看内容。</div>
      <div v-else-if="previewNote && !previewNote.startsWith('文件较大')" class="msg">{{ previewNote }}</div>
      <template v-else>
        <div class="preview-toolbar">
          <span class="path">{{ selectedPath }}</span>
          <span class="spacer"></span>
          <span v-if="saveError" class="msg error inline">{{ saveError }}</span>
          <template v-if="isEditing">
            <span v-if="dirty" class="dirty" title="有未保存改动">●</span>
            <button class="btn" :disabled="saving" @click="cancelEdit">取消</button>
            <button class="btn primary" :disabled="saving" @click="saveEdit">
              {{ saving ? '保存中…' : '保存' }}
            </button>
          </template>
          <button v-else-if="canEdit" class="btn" @click="startEdit">编辑</button>
        </div>
        <div v-if="previewNote" class="msg warn">{{ previewNote }}</div>
        <div ref="previewMount" class="preview-mount"></div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.file-browser { display: grid; grid-template-columns: minmax(200px, 300px) 1fr; gap: var(--space-2); height: 100%; min-height: 0; }
.tree-col { overflow: hidden; border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 0; }
.tree-mount { flex: 1; min-height: 0; overflow: auto; }
.preview-col { display: flex; flex-direction: column; overflow: hidden; min-width: 0; min-height: 0; }
.preview-mount { flex: 1; min-width: 0; min-height: 0; overflow: auto; }
.msg { padding: var(--space-2); color: var(--text-muted); font-size: var(--fs-sm); }
.msg.error { color: var(--danger, #e55); }
.msg.warn { color: var(--warning, #c80); }
.msg.inline { padding: 0; }
.preview-toolbar { display: flex; align-items: center; gap: var(--space-2); padding: var(--space-1) var(--space-2); border-bottom: 1px solid var(--border); font-size: var(--fs-sm); }
.preview-toolbar .path { color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.preview-toolbar .spacer { flex: 1; }
.preview-toolbar .dirty { color: var(--warning, #c80); }
.btn { padding: 2px 10px; border: 1px solid var(--border); border-radius: var(--radius-sm, 4px); background: var(--surface, transparent); color: var(--text); cursor: pointer; font-size: var(--fs-sm); }
.btn:hover:not(:disabled) { border-color: var(--text-muted); }
.btn:disabled { opacity: 0.5; cursor: default; }
.btn.primary { background: var(--accent, #37f); border-color: var(--accent, #37f); color: #fff; }
</style>
