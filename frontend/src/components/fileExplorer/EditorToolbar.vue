<script lang="ts" setup>
import { computed } from "vue";
import { Code2, Eye, FolderOpen, RotateCcw, Save, Settings2 } from "lucide-vue-next";

const props = defineProps<{
  path: string;
  dirty: boolean;
  /** 该文件是否有「源码 / 渲染」两种视图(markdown、svg)。 */
  canRender: boolean;
  viewMode: "code" | "render";
  /** 只有可编辑的文本类文件才给保存/还原按钮。 */
  editable: boolean;
}>();

const emit = defineEmits<{
  (e: "save"): void;
  (e: "revert"): void;
  (e: "toggle-view"): void;
  (e: "reveal"): void;
  (e: "open-prefs"): void;
}>();

// 面包屑:把 relPath 拆成段,最后一段是文件名。纯展示,不可点。
const segments = computed(() => props.path.split("/").filter(Boolean));
</script>

<template>
  <div class="toolbar" data-test="editor-toolbar">
    <div class="crumbs" :title="path">
      <template v-for="(seg, i) in segments" :key="i">
        <span v-if="i > 0" class="sep">/</span>
        <span class="crumb" :class="{ last: i === segments.length - 1 }">{{ seg }}</span>
      </template>
    </div>
    <div class="actions">
      <button
        v-if="canRender"
        class="tb-btn"
        data-test="toolbar-toggle-view"
        :title="viewMode === 'code' ? '预览渲染结果' : '查看源码'"
        @click="emit('toggle-view')"
      >
        <component :is="viewMode === 'code' ? Eye : Code2" :size="14" :stroke-width="1.8" />
      </button>
      <button
        v-if="editable"
        class="tb-btn"
        data-test="toolbar-revert"
        title="放弃改动"
        :disabled="!dirty"
        @click="emit('revert')"
      >
        <RotateCcw :size="14" :stroke-width="1.8" />
      </button>
      <button
        v-if="editable"
        class="tb-btn"
        data-test="toolbar-save"
        title="保存 (Ctrl/Cmd+S)"
        :disabled="!dirty"
        @click="emit('save')"
      >
        <Save :size="14" :stroke-width="1.8" />
      </button>
      <button class="tb-btn" data-test="toolbar-reveal" title="在文件管理器中显示" @click="emit('reveal')">
        <FolderOpen :size="14" :stroke-width="1.8" />
      </button>
      <button class="tb-btn" data-test="toolbar-prefs" title="编辑器偏好" @click="emit('open-prefs')">
        <Settings2 :size="14" :stroke-width="1.8" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-2, 8px);
  min-height: 26px;
  padding: 0 6px 0 10px;
  border-bottom: 1px solid var(--ed-border, #444c56);
  background: var(--ed-shell-bg, #22272e);
  font-size: 12px;
}
.crumbs {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--ed-muted, rgba(173, 186, 199, 0.7));
}
.crumb.last { color: var(--ed-row-fg, #adbac7); }
.sep { margin: 0 4px; opacity: 0.5; }
.actions { display: flex; align-items: center; gap: 2px; flex: 0 0 auto; }
.tb-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: 3px;
  background: transparent;
  color: var(--ed-muted, rgba(173, 186, 199, 0.7));
  cursor: pointer;
}
.tb-btn:hover:not(:disabled) { background: var(--ed-row-hover, rgba(255, 255, 255, 0.08)); color: var(--ed-row-fg, #adbac7); }
.tb-btn:disabled { opacity: 0.35; cursor: default; }
</style>
