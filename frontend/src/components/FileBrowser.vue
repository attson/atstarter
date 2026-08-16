<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { File, FilePlus2, Folder, FolderPlus, RefreshCw, Search, X } from 'lucide-vue-next'
import { useTheme } from '../composables/useTheme'
import FileTree from './fileExplorer/FileTree.vue'
import FileEditor from './fileExplorer/FileEditor.vue'
import EditorTabs from './fileExplorer/EditorTabs.vue'
import EditorToolbar from './fileExplorer/EditorToolbar.vue'
import EditorStatusBar from './fileExplorer/EditorStatusBar.vue'
import EditorPrefsPanel from './fileExplorer/EditorPrefsPanel.vue'
import FileConfirmDialog from './fileExplorer/FileConfirmDialog.vue'
import { createProjectFSBridge } from './fileExplorer/fsBridge'
import { readPrefs, writePrefs } from './fileExplorer/editorPrefs.js'
import {
  activateTab,
  closeTab,
  createTabsState,
  findTab,
  neighborTab,
  openTab,
  removePath,
  renamePath,
  setTabDirty,
  setTabViewMode,
} from './fileExplorer/editorTabs.js'
import { baseName, rewritePrefix } from './fileExplorer/pathOps.js'
import './fileExplorer/theme-bridge.css'

const props = defineProps({ projectId: { type: String, required: true } })

const { resolvedTheme } = useTheme()
const activeTheme = ref(resolvedTheme.value)
let themeObserver = null

// 懒加载文件树:每个项目一个 fsBridge(相对项目根的 relPath)。
const fs = computed(() => createProjectFSBridge(props.projectId))

// FileEditor 主题:App 的 dark → CodeMirror 的 dimmed。
const editorTheme = computed(() => (activeTheme.value === 'dark' ? 'dimmed' : 'light'))

const storage = typeof localStorage === 'undefined' ? null : localStorage
const prefs = reactive(readPrefs(storage))
const prefsOpen = ref(false)
// CodeEditor 只认这四项;showHidden 是文件树的事,不传下去。
const editorPrefs = computed(() => ({
  lineNumbers: prefs.lineNumbers,
  wrap: prefs.wrap,
  tabSize: prefs.tabSize,
  fontSize: prefs.fontSize,
}))

function updatePrefs(patch) {
  const next = writePrefs(storage, { ...prefs, ...patch })
  Object.assign(prefs, next)
}

// 标签页:切走的标签留在内存里(每个标签一个常驻 FileEditor,用 v-show 隐藏),
// 只有关闭标签时才拦截未保存改动。
const tabs = ref(createTabsState())
const activePath = computed(() => tabs.value.activePath)
const activeTab = computed(() => findTab(tabs.value, activePath.value))
// 每个标签自己的光标位置与预览类型,切回来时状态栏/工具栏不闪。
const cursors = reactive({})
const kinds = reactive({})

const treeRef = ref(null)
const statusBarRef = ref(null)
const editorRefs = new Map()
const closeGuard = ref(null)

const searchQuery = ref('')
const searchResults = ref([])
const searchLoading = ref(false)
const searchError = ref('')
const searchTruncated = ref(false)
let searchTimer = null
let searchGeneration = 0

const trimmedSearchQuery = computed(() => searchQuery.value.trim())
const searching = computed(() => trimmedSearchQuery.value.length > 0)

// 左侧文件树面板收起(腾空间给预览);偏好持久化。
const treeCollapsed = ref(storage?.getItem('fileBrowser.treeCollapsed') === '1')
function toggleTree() {
  treeCollapsed.value = !treeCollapsed.value
  storage?.setItem('fileBrowser.treeCollapsed', treeCollapsed.value ? '1' : '0')
}

/* ------------------------------------------------------------ 标签页操作 */

function setEditorRef(path, el) {
  if (el) editorRefs.set(path, el)
  else editorRefs.delete(path)
}

function openPath(path) {
  if (!path) return
  tabs.value = openTab(tabs.value, path)
}

function activate(path) {
  tabs.value = activateTab(tabs.value, path)
  void nextTick(() => editorRefs.get(path)?.focus?.())
}

// requestClose 是关闭标签的唯一入口:脏标签先弹「保存 / 不保存 / 取消」。
function requestClose(path) {
  const tab = findTab(tabs.value, path)
  if (!tab) return
  if (tab.dirty) {
    closeGuard.value = path
    return
  }
  forceClose(path)
}

function forceClose(path) {
  tabs.value = closeTab(tabs.value, path)
  delete cursors[path]
  delete kinds[path]
  editorRefs.delete(path)
}

async function resolveCloseGuard(id) {
  const path = closeGuard.value
  closeGuard.value = null
  if (!path || id === 'cancel') return
  if (id === 'save') {
    const saved = await editorRefs.get(path)?.save?.()
    if (!saved) return // 保存失败(冲突/权限)就把标签留着,别把改动吞了
  }
  forceClose(path)
}

function onDirtyChange(path, dirty) {
  tabs.value = setTabDirty(tabs.value, path, dirty)
}

function onCursorChange(path, pos) {
  cursors[path] = pos
}

function onKindChange(path, kind) {
  kinds[path] = kind
}

const activeKind = computed(() => kinds[activePath.value] ?? null)
const canRender = computed(() => activeKind.value === 'markdown' || activeKind.value === 'svg')
const isCodeView = computed(() => {
  if (activeKind.value === 'code') return true
  return canRender.value && activeTab.value?.viewMode === 'code'
})
const activeCursor = computed(() => cursors[activePath.value] ?? { line: 1, column: 1, selected: 0, lines: 1 })

const KIND_LABELS = {
  code: '文本',
  markdown: 'Markdown',
  svg: 'SVG',
  image: '图片',
  video: '视频',
  audio: '音频',
  pdf: 'PDF',
  'binary-unknown': '二进制',
}
const kindLabel = computed(() => KIND_LABELS[activeKind.value] ?? '')

function toggleViewMode() {
  const tab = activeTab.value
  if (!tab) return
  tabs.value = setTabViewMode(tabs.value, tab.path, tab.viewMode === 'code' ? 'render' : 'code')
}

function saveActive() {
  void editorRefs.get(activePath.value)?.save?.()
}

function revertActive() {
  editorRefs.get(activePath.value)?.revert?.()
}

function gotoLine(line) {
  editorRefs.get(activePath.value)?.gotoLine?.(line)
}

async function revealActive() {
  if (!activePath.value) return
  try {
    await fs.value.reveal(activePath.value)
  } catch (err) {
    treeError.value = err?.message || '无法在文件管理器中显示'
  }
}

/* ---------------------------------------------------- 文件树回调与错误 */

const treeError = ref('')

function onTreeError(message) {
  treeError.value = message
}

// 重命名/移动之后标签要跟着走,光标与类型缓存也一起搬。
function onPathRenamed(from, to) {
  tabs.value = renamePath(tabs.value, from, to)
  for (const map of [cursors, kinds]) {
    for (const key of Object.keys(map)) {
      const next = rewritePrefix(key, from, to)
      if (next === key) continue
      map[next] = map[key]
      delete map[key]
    }
  }
}

function onPathRemoved(path) {
  tabs.value = removePath(tabs.value, path)
  for (const map of [cursors, kinds]) {
    for (const key of Object.keys(map)) {
      if (key === path || key.startsWith(path + '/')) delete map[key]
    }
  }
}

/* ---------------------------------------------------------------- 搜索 */

function clearSearch() {
  searchQuery.value = ''
  searchResults.value = []
  searchError.value = ''
  searchTruncated.value = false
  searchLoading.value = false
}

function onSearchResultClick(result) {
  if (!result || result.isDir) return
  openPath(result.path)
}

function resetSearchResults() {
  searchResults.value = []
  searchError.value = ''
  searchTruncated.value = false
  searchLoading.value = false
}

function scheduleSearch() {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
  const query = trimmedSearchQuery.value
  const bridge = fs.value
  const request = ++searchGeneration
  if (!query) {
    resetSearchResults()
    return
  }
  searchLoading.value = true
  searchError.value = ''
  searchTimer = setTimeout(() => {
    void runSearch(bridge, query, request)
  }, 160)
}

async function runSearch(bridge, query, request) {
  try {
    const result = await bridge.searchPaths(query, 100)
    if (request !== searchGeneration || fs.value !== bridge || trimmedSearchQuery.value !== query) return
    searchResults.value = result.matches || []
    searchTruncated.value = !!result.truncated
    searchError.value = ''
  } catch (err) {
    if (request !== searchGeneration || fs.value !== bridge || trimmedSearchQuery.value !== query) return
    searchResults.value = []
    searchTruncated.value = false
    searchError.value = err?.message || '搜索失败'
  } finally {
    if (request === searchGeneration && fs.value === bridge && trimmedSearchQuery.value === query) {
      searchLoading.value = false
    }
  }
}

/* ------------------------------------------------------------ 快捷键 */

function onKeydown(e) {
  const mod = e.ctrlKey || e.metaKey
  if (!mod) return
  const key = e.key.toLowerCase()
  if (key === 's') {
    if (!activePath.value) return
    e.preventDefault()
    saveActive()
    return
  }
  if (key === 'w') {
    if (!activePath.value) return
    e.preventDefault()
    requestClose(activePath.value)
    return
  }
  if (key === 'g') {
    if (!activePath.value || !isCodeView.value) return
    e.preventDefault()
    statusBarRef.value?.openGoto?.()
    return
  }
  // Ctrl/Cmd + PageUp/PageDown 在标签间走。
  if (e.key === 'PageDown' || e.key === 'PageUp') {
    const next = neighborTab(tabs.value, e.key === 'PageDown' ? 1 : -1)
    if (!next) return
    e.preventDefault()
    activate(next)
  }
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
watch(() => [trimmedSearchQuery.value, fs.value], scheduleSearch)
// 换项目 = 换一套标签。内容都在各自的 CodeEditor 里,跟着卸载。
watch(() => props.projectId, () => {
  tabs.value = createTabsState()
  editorRefs.clear()
  for (const key of Object.keys(cursors)) delete cursors[key]
  for (const key of Object.keys(kinds)) delete kinds[key]
  treeError.value = ''
  clearSearch()
})

onMounted(() => {
  syncActiveTheme()
  themeObserver = new MutationObserver(syncActiveTheme)
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  searchGeneration++
  if (searchTimer) clearTimeout(searchTimer)
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
        <div class="tree-title-row">
          <span class="tree-title">文件</span>
          <span class="tree-tools">
            <!-- 最外层新建的固定入口:不依赖右键选中了哪一行。 -->
            <button class="icon-btn" data-test="tree-new-file" title="在项目根新建文件" @click="treeRef?.newFile()">
              <FilePlus2 :size="14" :stroke-width="1.8" />
            </button>
            <button class="icon-btn" data-test="tree-new-folder" title="在项目根新建文件夹" @click="treeRef?.newFolder()">
              <FolderPlus :size="14" :stroke-width="1.8" />
            </button>
            <button class="icon-btn" data-test="tree-refresh" title="刷新" @click="treeRef?.refresh()">
              <RefreshCw :size="14" :stroke-width="1.8" />
            </button>
            <button class="icon-btn" title="收起文件树" @click="toggleTree">‹</button>
          </span>
        </div>
        <label class="search-box">
          <Search :size="13" :stroke-width="1.8" />
          <input
            v-model="searchQuery"
            data-test="file-search-input"
            type="search"
            placeholder="搜索文件名"
            autocomplete="off"
            spellcheck="false"
          >
          <button v-if="searchQuery" class="search-clear" title="清空搜索" type="button" @click="clearSearch">
            <X :size="13" :stroke-width="1.8" />
          </button>
        </label>
      </div>
      <div v-if="searching" class="search-results" data-test="file-search-results">
        <div v-if="searchLoading" class="search-state">搜索中...</div>
        <div v-else-if="searchError" class="search-state search-error">{{ searchError }}</div>
        <div v-else-if="!searchResults.length" class="search-state">
          {{ searchTruncated ? '搜索达到上限,请缩小关键词。' : '没有匹配文件。' }}
        </div>
        <template v-else>
          <button
            v-for="result in searchResults"
            :key="result.path"
            class="search-result"
            :class="{ selected: activePath === result.path, disabled: result.isDir }"
            type="button"
            :title="result.path"
            @click="onSearchResultClick(result)"
          >
            <Folder v-if="result.isDir" :size="14" :stroke-width="1.6" />
            <File v-else :size="14" :stroke-width="1.6" />
            <span class="result-name">{{ result.name }}</span>
            <span class="result-path">{{ result.path }}</span>
          </button>
          <div v-if="searchTruncated" class="search-state">结果较多,已显示前 100 项。</div>
        </template>
      </div>
      <FileTree
        v-else
        ref="treeRef"
        class="tree-mount"
        :fs="fs"
        root=""
        :show-hidden="prefs.showHidden"
        @file-clicked="openPath"
        @path-renamed="onPathRenamed"
        @path-removed="onPathRemoved"
        @error="onTreeError"
      />
    </div>
    <div class="preview-col">
      <EditorTabs
        v-if="tabs.tabs.length"
        :tabs="tabs.tabs"
        :active-path="activePath"
        @activate="activate"
        @close="requestClose"
      />
      <div v-if="!activePath" class="msg">从左侧选择文件查看内容。</div>
      <template v-else>
        <EditorToolbar
          :path="activePath"
          :dirty="!!activeTab?.dirty"
          :can-render="canRender"
          :view-mode="activeTab?.viewMode ?? 'code'"
          :editable="isCodeView"
          @save="saveActive"
          @revert="revertActive"
          @toggle-view="toggleViewMode"
          @reveal="revealActive"
          @open-prefs="prefsOpen = true"
        />
        <div class="editor-stack">
          <!-- 每个标签一个常驻实例:切标签不重载,未保存的改动留在内存里。 -->
          <FileEditor
            v-for="tab in tabs.tabs"
            :key="tab.path"
            v-show="tab.path === activePath"
            :ref="(el) => setEditorRef(tab.path, el)"
            class="editor-host"
            :fs="fs"
            :path="tab.path"
            :prefs="editorPrefs"
            :theme="editorTheme"
            :view-mode="tab.viewMode"
            @dirty-change="(v) => onDirtyChange(tab.path, v)"
            @cursor-change="(p) => onCursorChange(tab.path, p)"
            @kind-change="(k) => onKindChange(tab.path, k)"
          />
        </div>
        <EditorStatusBar
          ref="statusBarRef"
          :line="activeCursor.line"
          :column="activeCursor.column"
          :selected="activeCursor.selected"
          :lines="activeCursor.lines"
          :show-cursor="isCodeView"
          :kind-label="kindLabel"
          @goto="gotoLine"
        />
      </template>
      <div v-if="treeError" class="browser-error" data-test="file-browser-error">{{ treeError }}</div>
    </div>

    <EditorPrefsPanel
      v-if="prefsOpen"
      :prefs="prefs"
      @update="updatePrefs"
      @close="prefsOpen = false"
    />

    <FileConfirmDialog
      v-if="closeGuard"
      :title="`「${baseName(closeGuard)}」有未保存的改动`"
      message="关闭标签会丢失这些改动。"
      :buttons="[
        { id: 'save', label: '保存并关闭', kind: 'primary' },
        { id: 'discard', label: '不保存', kind: 'danger' },
        { id: 'cancel', label: '取消', kind: 'secondary' },
      ]"
      @resolve="resolveCloseGuard"
    />
  </div>
</template>

<style scoped>
.file-browser { display: grid; grid-template-columns: minmax(200px, 300px) 1fr; gap: var(--space-2); height: 100%; min-height: 0; }
.file-browser.tree-collapsed { grid-template-columns: 24px 1fr; }
.tree-col { overflow: hidden; border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 0; }
.tree-mount { flex: 1; min-height: 0; overflow: auto; }
.tree-head { display: flex; flex-direction: column; gap: var(--space-2); padding: var(--space-1) var(--space-2) var(--space-2); border-bottom: 1px solid var(--border); }
.tree-title-row { display: flex; align-items: center; justify-content: space-between; min-height: 22px; gap: var(--space-2); }
.tree-title { font-size: var(--fs-sm); color: var(--text-muted); }
.tree-tools { display: inline-flex; align-items: center; gap: 2px; }
.icon-btn { display: inline-flex; align-items: center; justify-content: center; border: none; background: transparent; color: var(--text-muted); cursor: pointer; font-size: 16px; line-height: 1; padding: 2px 4px; border-radius: 3px; }
.icon-btn:hover { background: var(--surface-hover, rgba(127,127,127,.15)); color: var(--text); }
.search-box { display: flex; align-items: center; gap: var(--space-2); min-height: 26px; padding: 0 var(--space-2); border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--elevated); color: var(--text-muted); }
.search-box input { flex: 1; min-width: 0; border: 0; outline: 0; padding: 0; background: transparent; color: var(--text); font: inherit; font-size: var(--fs-sm); }
.search-box input::placeholder { color: var(--text-subtle); }
.search-clear { display: inline-flex; align-items: center; justify-content: center; width: 18px; height: 18px; padding: 0; border: 0; border-radius: 3px; background: transparent; color: var(--text-muted); cursor: pointer; }
.search-clear:hover { background: var(--surface-hover, rgba(127,127,127,.15)); color: var(--text); }
.search-results { flex: 1; min-height: 0; overflow: auto; padding: var(--space-1); }
.search-state { padding: var(--space-2); color: var(--text-muted); font-size: var(--fs-sm); }
.search-error { color: var(--danger); }
.search-result { display: grid; grid-template-columns: 18px minmax(0, 1fr); grid-template-areas: "icon name" "icon path"; align-items: center; column-gap: var(--space-2); width: 100%; min-height: 34px; padding: var(--space-1) var(--space-2); border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text); cursor: pointer; text-align: left; }
.search-result svg { grid-area: icon; color: var(--text-muted); }
.search-result:hover { background: var(--surface-hover, rgba(127,127,127,.15)); }
.search-result.selected { background: var(--success-soft); }
.search-result.disabled { cursor: default; }
.search-result.disabled:hover { background: transparent; }
.result-name { grid-area: name; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: var(--fs-sm); }
.result-path { grid-area: path; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-muted); font-size: var(--fs-xs); }
.tree-expand-strip { border: none; border-right: 1px solid var(--border); background: transparent; color: var(--text-muted); cursor: pointer; font-size: 16px; writing-mode: vertical-rl; padding: var(--space-2) 0; }
.tree-expand-strip:hover { background: var(--surface-hover, rgba(127,127,127,.15)); color: var(--text); }
.preview-col { display: flex; flex-direction: column; overflow: hidden; min-width: 0; min-height: 0; }
.editor-stack { position: relative; flex: 1; min-width: 0; min-height: 0; overflow: hidden; }
.editor-host { position: absolute; inset: 0; min-width: 0; min-height: 0; overflow: hidden; }
.msg { padding: var(--space-2); color: var(--text-muted); font-size: var(--fs-sm); }
.browser-error { padding: var(--space-1) var(--space-2); border-top: 1px solid var(--border); color: var(--danger); font-size: var(--fs-xs); }
</style>
