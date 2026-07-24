<script setup>
import { ref } from 'vue'
import { ListProjectDir } from '../../wailsjs/go/main/App'

const props = defineProps({
  projectId: { type: String, required: true },
  entry: { type: Object, required: true }, // { name, isDir, size }
  path: { type: String, required: true },  // 该节点相对项目根的路径
  depth: { type: Number, default: 0 },
  selectedPath: { type: String, default: '' },
})
const emit = defineEmits(['select'])

const expanded = ref(false)
const loaded = ref(false)
const children = ref([])
const error = ref('')

async function toggle() {
  if (!props.entry.isDir) {
    emit('select', props.path)
    return
  }
  expanded.value = !expanded.value
  if (expanded.value && !loaded.value) {
    try {
      children.value = await ListProjectDir(props.projectId, props.path)
      loaded.value = true
    } catch (e) {
      error.value = String(e)
    }
  }
}
</script>

<template>
  <div>
    <div
      class="node"
      :class="{ selected: !entry.isDir && selectedPath === path }"
      :style="{ paddingLeft: depth * 14 + 8 + 'px' }"
      @click="toggle"
    >
      <span class="twisty">{{ entry.isDir ? (expanded ? '▾' : '▸') : '' }}</span>
      <span class="name">{{ entry.name }}</span>
    </div>
    <div v-if="error" class="node-error" :style="{ paddingLeft: (depth + 1) * 14 + 8 + 'px' }">{{ error }}</div>
    <template v-if="entry.isDir && expanded">
      <FileTreeNode
        v-for="child in children"
        :key="child.name"
        :projectId="projectId"
        :entry="child"
        :path="path ? path + '/' + child.name : child.name"
        :depth="depth + 1"
        :selectedPath="selectedPath"
        @select="(p) => emit('select', p)"
      />
    </template>
  </div>
</template>

<style scoped>
.node { display: flex; align-items: center; gap: var(--space-1); height: 24px; cursor: pointer; font-size: var(--fs-sm); color: var(--text); border-radius: var(--radius-sm); }
.node:hover { background: var(--elevated); }
.node.selected { background: var(--elevated); color: var(--text); font-weight: var(--fw-medium); }
.twisty { width: 12px; display: inline-block; text-align: center; color: var(--text-muted); }
.name { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.node-error { color: var(--danger, #e55); font-size: var(--fs-sm); }
</style>
