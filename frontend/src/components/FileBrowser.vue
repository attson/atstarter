<script setup>
import { ref, watch } from 'vue'
import { ListProjectDir, ReadProjectFile } from '../../wailsjs/go/main/App'
import FileTreeNode from './FileTreeNode.vue'

const props = defineProps({ projectId: { type: String, required: true } })

const rootEntries = ref([])
const treeError = ref('')
const selectedPath = ref('')
const preview = ref(null) // { content, size, truncated, binary }
const previewError = ref('')
const loadingPreview = ref(false)

async function loadRoot() {
  treeError.value = ''
  rootEntries.value = []
  selectedPath.value = ''
  preview.value = null
  previewError.value = ''
  loadingPreview.value = false
  if (!props.projectId) return
  try {
    rootEntries.value = await ListProjectDir(props.projectId, '')
  } catch (e) {
    treeError.value = String(e)
  }
}

async function onSelect(path) {
  selectedPath.value = path
  previewError.value = ''
  preview.value = null
  loadingPreview.value = true
  try {
    const result = await ReadProjectFile(props.projectId, path)
    if (selectedPath.value !== path) return // 已切到别的文件,丢弃过期响应
    preview.value = result
  } catch (e) {
    if (selectedPath.value !== path) return
    previewError.value = String(e)
  } finally {
    if (selectedPath.value === path) loadingPreview.value = false
  }
}

watch(() => props.projectId, loadRoot, { immediate: true })
</script>

<template>
  <div class="file-browser">
    <div class="tree">
      <div v-if="treeError" class="msg error">{{ treeError }}</div>
      <FileTreeNode
        v-for="entry in rootEntries"
        :key="entry.name"
        :projectId="projectId"
        :entry="entry"
        :path="entry.name"
        :depth="0"
        :selectedPath="selectedPath"
        @select="onSelect"
      />
    </div>
    <div class="preview">
      <div v-if="loadingPreview" class="msg">加载中…</div>
      <div v-else-if="previewError" class="msg error">{{ previewError }}</div>
      <div v-else-if="!selectedPath" class="msg">从左侧选择文件查看内容。</div>
      <div v-else-if="preview && preview.binary" class="msg">二进制文件,不预览。</div>
      <template v-else-if="preview">
        <div v-if="preview.truncated" class="msg warn">文件较大,仅显示前 1MB。</div>
        <pre class="content">{{ preview.content }}</pre>
      </template>
    </div>
  </div>
</template>

<style scoped>
.file-browser { display: grid; grid-template-columns: minmax(180px, 260px) 1fr; gap: var(--space-2); height: 100%; min-height: 0; }
.tree { overflow: auto; border-right: 1px solid var(--border); padding-right: var(--space-1); }
.preview { overflow: auto; min-width: 0; }
.content { margin: 0; padding: var(--space-2); font-family: var(--font-mono, monospace); font-size: var(--fs-sm); white-space: pre; color: var(--text); }
.msg { padding: var(--space-2); color: var(--text-muted); font-size: var(--fs-sm); }
.msg.error { color: var(--danger, #e55); }
.msg.warn { color: var(--warning, #c80); }
</style>
