<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useTheme } from '../composables/useTheme'
import FileTree from './fileExplorer/FileTree.vue'
import FileEditor from './fileExplorer/FileEditor.vue'
import { createProjectFSBridge } from './fileExplorer/fsBridge'
import './fileExplorer/theme-bridge.css'

const props = defineProps({ projectId: { type: String, required: true } })

const { resolvedTheme } = useTheme()
const activeTheme = ref(resolvedTheme.value)
let themeObserver = null

// 懒加载文件树:每个项目一个 fsBridge(相对项目根的 relPath)。
const fs = computed(() => createProjectFSBridge(props.projectId))

// FileEditor 主题:App 的 dark → CodeMirror 的 dimmed。
const editorTheme = computed(() => (activeTheme.value === 'dark' ? 'dimmed' : 'light'))

const selectedPath = ref('')       // 当前预览/编辑的文件 relPath
const dirty = ref(false)           // 编辑器是否有未保存改动
const fileEditorRef = ref(null)

// 行号显隐:默认隐藏,右键预览区可开关;偏好持久化。
const showLineNumbers = ref(localStorage.getItem('fileBrowser.lineNumbers') === '1')
const menu = ref({ open: false, x: 0, y: 0 })

// 左侧文件树面板收起(腾空间给预览);偏好持久化。
const treeCollapsed = ref(localStorage.getItem('fileBrowser.treeCollapsed') === '1')
function toggleTree() {
  treeCollapsed.value = !treeCollapsed.value
  localStorage.setItem('fileBrowser.treeCollapsed', treeCollapsed.value ? '1' : '0')
}

// atterm FileTree 的 @file-clicked 传来相对项目根的 relPath。
function onSelect(path) {
  if (!path) return
  selectedPath.value = path
}

function onDirtyChange(v) { dirty.value = v }

// 保存快捷键 Ctrl/Cmd+S 交给 FileEditor(CodeEditor 内部也监听,这里兜底触发)。
function onKeydown(e) {
  if ((e.ctrlKey || e.metaKey) && (e.key === 's' || e.key === 'S')) {
    if (!selectedPath.value) return
    e.preventDefault()
    fileEditorRef.value?.save?.()
  }
}

// 预览区右键:切换行号。
function onContextMenu(e) {
  if (!selectedPath.value) return
  e.preventDefault()
  menu.value = { open: true, x: e.clientX, y: e.clientY }
}
function closeMenu() { menu.value.open = false }
function toggleLineNumbers() {
  showLineNumbers.value = !showLineNumbers.value
  localStorage.setItem('fileBrowser.lineNumbers', showLineNumbers.value ? '1' : '0')
  menu.value.open = false
}

function documentTheme() {
  if (typeof document === 'undefined') return resolvedTheme.value
  const value = document.documentElement.getAttribute('data-theme')
  return value === 'dark' || value === 'light' ? value : resolvedTheme.value
}

function syncActiveTheme() {
  activeTheme.value = documentTheme()
}

watch(resolvedTheme, syncActiveTheme)

onMounted(() => {
  syncActiveTheme()
  themeObserver = new MutationObserver(syncActiveTheme)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  if (themeObserver) themeObserver.disconnect()
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="file-browser fe-scope" :class="{ 'tree-collapsed': treeCollapsed }">
    <!-- 收起态:窄条,点击展开树 -->
    <button v-if="treeCollapsed" class="tree-expand-strip" title="展开文件树" @click="toggleTree">›</button>
    <div v-show="!treeCollapsed" class="tree-col">
      <div class="tree-head">
        <span class="tree-title">文件</span>
        <button class="icon-btn" title="收起文件树" @click="toggleTree">‹</button>
      </div>
      <FileTree
        class="tree-mount"
        :fs="fs"
        root=""
        :show-hidden="true"
        @file-clicked="onSelect"
      />
    </div>
    <div class="preview-col" @contextmenu="onContextMenu">
      <div v-if="!selectedPath" class="msg">从左侧选择文件查看内容。</div>
      <template v-else>
        <div class="preview-toolbar">
          <span class="path">{{ selectedPath }}</span>
          <span v-if="dirty" class="dirty" title="有未保存改动">●</span>
        </div>
        <FileEditor
          ref="fileEditorRef"
          class="editor-host"
          :fs="fs"
          :path="selectedPath"
          :show-line-numbers="showLineNumbers"
          :theme="editorTheme"
          view-mode="code"
          @dirty-change="onDirtyChange"
        />
      </template>
    </div>

    <!-- 预览区右键菜单:切换行号 -->
    <div v-if="menu.open" class="ctx-backdrop" @click="closeMenu" @contextmenu.prevent="closeMenu">
      <ul class="ctx-menu" :style="{ left: menu.x + 'px', top: menu.y + 'px' }" @click.stop>
        <li @click="toggleLineNumbers">{{ showLineNumbers ? '隐藏行号' : '显示行号' }}</li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.file-browser { display: grid; grid-template-columns: minmax(200px, 300px) 1fr; gap: var(--space-2); height: 100%; min-height: 0; }
.file-browser.tree-collapsed { grid-template-columns: 24px 1fr; }
.tree-col { overflow: hidden; border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 0; }
.tree-mount { flex: 1; min-height: 0; overflow: auto; }
.tree-head { display: flex; align-items: center; justify-content: space-between; padding: var(--space-1) var(--space-2); border-bottom: 1px solid var(--border); }
.tree-title { font-size: var(--fs-sm); color: var(--text-muted); }
.icon-btn { border: none; background: transparent; color: var(--text-muted); cursor: pointer; font-size: 16px; line-height: 1; padding: 2px 6px; border-radius: 3px; }
.icon-btn:hover { background: var(--surface-hover, rgba(127,127,127,.15)); color: var(--text); }
.tree-expand-strip { border: none; border-right: 1px solid var(--border); background: transparent; color: var(--text-muted); cursor: pointer; font-size: 16px; writing-mode: vertical-rl; padding: var(--space-2) 0; }
.tree-expand-strip:hover { background: var(--surface-hover, rgba(127,127,127,.15)); color: var(--text); }
.preview-col { display: flex; flex-direction: column; overflow: hidden; min-width: 0; min-height: 0; }
.editor-host { flex: 1; min-width: 0; min-height: 0; overflow: hidden; }
.msg { padding: var(--space-2); color: var(--text-muted); font-size: var(--fs-sm); }
.preview-toolbar { display: flex; align-items: center; gap: var(--space-2); padding: var(--space-1) var(--space-2); border-bottom: 1px solid var(--border); font-size: var(--fs-sm); }
.preview-toolbar .path { color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.preview-toolbar .dirty { color: var(--warning, #c80); }

/* 右键菜单 */
.ctx-backdrop { position: fixed; inset: 0; z-index: 1000; }
.ctx-menu { position: fixed; margin: 0; padding: 4px; list-style: none; min-width: 120px;
  background: var(--surface, #222); border: 1px solid var(--border); border-radius: var(--radius-sm, 4px);
  box-shadow: 0 4px 16px rgba(0,0,0,.3); font-size: var(--fs-sm); }
.ctx-menu li { padding: 6px 12px; border-radius: 3px; cursor: pointer; color: var(--text); white-space: nowrap; }
.ctx-menu li:hover { background: var(--accent, #37f); color: #fff; }
</style>
