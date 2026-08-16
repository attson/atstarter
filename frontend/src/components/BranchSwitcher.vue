<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Check, ChevronDown, GitBranch, Loader2, Plus } from 'lucide-vue-next'
import AppPill from './ui/AppPill.vue'
import AppIcon from './ui/AppIcon.vue'
import {
  allBranchNames,
  branchGroups,
  branchPillLabel,
  canCreateFromQuery,
  checkoutWarnings,
  validateBranchName,
} from '../gitBranches.js'
import {
  CheckoutProjectBranch,
  CreateProjectBranch,
  ListProjectBranches,
} from '../../wailsjs/go/main/App'

const props = defineProps({
  project: { type: Object, default: null },
  running: { type: Boolean, default: false },
})
const emit = defineEmits(['changed'])

const data = ref(null)
const open = ref(false)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const query = ref('')
const searchInput = ref(null)
const rootRef = ref(null)

// 分支数据每次打开都重新拉:用户可能刚在终端里 fetch 过。
// 关着的时候只保留上一次的 status,用来渲染标签文字。
let loadToken = 0

const label = computed(() => branchPillLabel(data.value))
const groups = computed(() => branchGroups(data.value, query.value))
const warnings = computed(() => checkoutWarnings(data.value, props.running))
const trimmedQuery = computed(() => query.value.trim())
const showCreate = computed(() => canCreateFromQuery(data.value, trimmedQuery.value))
const createError = computed(() =>
  trimmedQuery.value ? validateBranchName(trimmedQuery.value, allBranchNames(data.value)) : '',
)
const empty = computed(() => !loading.value && !groups.value.length && !showCreate.value)

async function load() {
  const projectId = props.project?.id
  if (!projectId) {
    data.value = null
    return
  }
  const token = ++loadToken
  loading.value = true
  error.value = ''
  try {
    const result = await ListProjectBranches(projectId)
    if (token !== loadToken) return
    data.value = result
  } catch (err) {
    if (token !== loadToken) return
    data.value = null
    error.value = err?.message || '读取分支失败'
  } finally {
    if (token === loadToken) loading.value = false
  }
}

async function toggleOpen() {
  if (open.value) {
    close()
    return
  }
  open.value = true
  query.value = ''
  error.value = ''
  await load()
  await nextTick()
  searchInput.value?.focus()
}

function close() {
  open.value = false
  query.value = ''
}

async function runCheckout(action) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    data.value = await action()
    close()
    emit('changed', data.value)
  } catch (err) {
    // git 自己的错误消息最准确("Your local changes ... would be overwritten"),
    // 原样展示,不要包装成"切换失败"。
    error.value = err?.message || '切换分支失败'
  } finally {
    busy.value = false
  }
}

function checkout(name) {
  if (name === data.value?.branch) {
    close()
    return
  }
  void runCheckout(() => CheckoutProjectBranch(props.project.id, name))
}

function createBranch() {
  const name = trimmedQuery.value
  if (!showCreate.value) return
  // 起点留空 = 从当前 HEAD 开新分支,和 `git checkout -b` 的默认一致。
  void runCheckout(() => CreateProjectBranch(props.project.id, name, ''))
}

function onSearchKey(e) {
  if (e.key === 'Escape') {
    close()
    return
  }
  if (e.key !== 'Enter') return
  const first = groups.value[0]?.items?.[0]
  if (first) checkout(first.name)
  else if (showCreate.value) createBranch()
}

function onDocumentClick(e) {
  if (!open.value) return
  if (rootRef.value?.contains(e.target)) return
  close()
}

watch(open, (isOpen) => {
  if (isOpen) document.addEventListener('mousedown', onDocumentClick)
  else document.removeEventListener('mousedown', onDocumentClick)
})

// 换项目就重新读一次当前分支(标签要立刻更新),但不展开面板。
watch(
  () => props.project?.id,
  () => {
    close()
    data.value = null
    void load()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  loadToken++
  document.removeEventListener('mousedown', onDocumentClick)
})
</script>

<template>
  <div v-if="label || open" ref="rootRef" class="branch-switcher" data-test="branch-switcher">
    <AppPill
      variant="neutral"
      class="branch-pill"
      clickable
      :active="open"
      data-test="branch-pill"
      :title="data?.dirty ? '工作区有未提交改动' : '切换分支'"
      @click="toggleOpen"
    >
      <AppIcon :icon="GitBranch" :size="11" />
      {{ label }}
      <span v-if="data?.dirty" class="dirty-dot" title="工作区有未提交改动">●</span>
      <AppIcon :icon="ChevronDown" :size="11" />
    </AppPill>

    <div v-if="open" class="branch-menu" data-test="branch-menu">
      <input
        ref="searchInput"
        v-model="query"
        class="branch-search"
        data-test="branch-search"
        type="search"
        placeholder="筛选或输入新分支名"
        autocomplete="off"
        spellcheck="false"
        @keydown="onSearchKey"
      >

      <div v-if="warnings.length" class="branch-warnings">
        <div v-for="w in warnings" :key="w">{{ w }}</div>
      </div>

      <div v-if="error" class="branch-error" data-test="branch-error">{{ error }}</div>

      <div class="branch-list">
        <div v-if="loading" class="branch-state">
          <AppIcon :icon="Loader2" :size="12" class="spin" /> 读取分支…
        </div>
        <template v-for="group in groups" :key="group.id">
          <div class="branch-group">{{ group.label }}</div>
          <button
            v-for="item in group.items"
            :key="group.id + ':' + item.name"
            class="branch-item"
            :class="{ current: item.current }"
            :data-test="`branch-item-${item.name}`"
            :disabled="busy"
            type="button"
            @click="checkout(item.name)"
          >
            <AppIcon v-if="item.current" :icon="Check" :size="12" />
            <span v-else class="branch-item-spacer" />
            <span class="branch-item-name">{{ item.name }}</span>
            <span v-if="item.kind === 'remote'" class="branch-item-tag">远端</span>
          </button>
        </template>
        <div v-if="empty && !error" class="branch-state">
          {{ createError || (data?.repo ? '没有匹配分支。' : '该项目不是 git 仓库。') }}
        </div>
      </div>

      <button
        v-if="showCreate"
        class="branch-create"
        data-test="branch-create"
        :disabled="busy"
        type="button"
        @click="createBranch"
      >
        <AppIcon :icon="Plus" :size="12" />
        从当前 HEAD 新建「{{ trimmedQuery }}」
      </button>
    </div>
  </div>
</template>

<style scoped>
.branch-switcher { position: relative; display: inline-flex; }
.branch-pill {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--accent-strong);
  border-color: var(--success-line);
  background: var(--success-gradient);
}
.dirty-dot { color: var(--warning); font-size: 8px; line-height: 1; }
.branch-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 40;
  display: flex;
  flex-direction: column;
  min-width: 260px;
  max-width: 340px;
  padding: var(--space-2);
  gap: var(--space-2);
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg, 0 8px 24px rgba(0, 0, 0, 0.28));
}
.branch-search {
  min-height: 26px;
  padding: 0 var(--space-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  outline: none;
}
.branch-search:focus { border-color: var(--accent-strong); }
.branch-warnings {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--warning-gradient);
  color: var(--warning);
  font-size: var(--fs-xs);
  line-height: 1.5;
}
.branch-error {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--danger-gradient);
  color: var(--danger-fg);
  font-size: var(--fs-xs);
  line-height: 1.5;
  white-space: pre-wrap;
  max-height: 140px;
  overflow: auto;
}
.branch-list { max-height: 260px; overflow: auto; display: flex; flex-direction: column; }
.branch-group {
  padding: var(--space-1) var(--space-2) 2px;
  color: var(--text-subtle);
  font-size: var(--fs-xs);
}
.branch-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  min-height: 26px;
  padding: 0 var(--space-2);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  text-align: left;
  cursor: pointer;
}
.branch-item:hover:not(:disabled) { background: var(--surface-hover, rgba(127, 127, 127, 0.15)); }
.branch-item:disabled { opacity: 0.5; cursor: default; }
.branch-item.current { color: var(--accent-strong); }
.branch-item-spacer { width: 12px; flex: 0 0 12px; }
.branch-item-name { flex: 1 1 auto; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.branch-item-tag { flex: 0 0 auto; color: var(--text-subtle); font-size: var(--fs-xs); }
.branch-state {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}
.branch-create {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 26px;
  padding: 0 var(--space-2);
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  cursor: pointer;
}
.branch-create:hover:not(:disabled) { background: var(--surface-hover, rgba(127, 127, 127, 0.15)); }
.branch-create:disabled { opacity: 0.5; cursor: default; }
.spin { animation: branch-spin 1s linear infinite; }
@keyframes branch-spin { to { transform: rotate(360deg); } }
</style>
