<script lang="ts" setup>
import { X } from "lucide-vue-next";
import { baseName } from "./pathOps.js";

interface Tab {
  path: string;
  dirty: boolean;
  viewMode: "code" | "render";
}

defineProps<{
  tabs: Tab[];
  activePath: string;
}>();

const emit = defineEmits<{
  (e: "activate", path: string): void;
  (e: "close", path: string): void;
}>();

// 中键关闭是标签栏的通用习惯,不给它加可见按钮。
function onMouseUp(ev: MouseEvent, path: string) {
  if (ev.button === 1) {
    ev.preventDefault();
    emit("close", path);
  }
}
</script>

<template>
  <div class="tabs" data-test="editor-tabs" role="tablist">
    <div
      v-for="tab in tabs"
      :key="tab.path"
      class="tab"
      :class="{ active: tab.path === activePath, dirty: tab.dirty }"
      :data-test="`editor-tab-${tab.path}`"
      :title="tab.path"
      role="tab"
      :aria-selected="tab.path === activePath"
      @click="emit('activate', tab.path)"
      @mouseup="onMouseUp($event, tab.path)"
    >
      <span class="name">{{ baseName(tab.path) }}</span>
      <span v-if="tab.dirty" class="dot" title="有未保存改动">●</span>
      <button
        v-else
        class="close"
        title="关闭"
        data-test="editor-tab-close"
        @click.stop="emit('close', tab.path)"
      >
        <X :size="12" :stroke-width="2" />
      </button>
      <!-- 脏标签的关闭按钮在 hover 时替换掉圆点,避免误关。 -->
      <button
        v-if="tab.dirty"
        class="close close-dirty"
        title="关闭"
        data-test="editor-tab-close"
        @click.stop="emit('close', tab.path)"
      >
        <X :size="12" :stroke-width="2" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  align-items: stretch;
  overflow-x: auto;
  overflow-y: hidden;
  min-height: 28px;
  background: var(--ed-shell-bg, #22272e);
  border-bottom: 1px solid var(--ed-border, #444c56);
  scrollbar-width: thin;
}
.tab {
  position: relative;
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
  max-width: 180px;
  padding: 0 6px 0 10px;
  border-right: 1px solid var(--ed-border, #444c56);
  background: var(--ed-tab-bg, #2d333b);
  color: var(--ed-muted, rgba(173, 186, 199, 0.7));
  font-size: 12px;
  cursor: pointer;
  user-select: none;
}
.tab:hover { color: var(--ed-row-fg, #adbac7); }
.tab.active {
  background: var(--ed-editor-bg, #22272e);
  color: var(--ed-row-fg, #adbac7);
  box-shadow: inset 0 -2px 0 var(--ed-tab-active-bar, #539bf5);
}
.name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dot { color: var(--ed-warning, #c69026); font-size: 10px; line-height: 1; }
.tab.dirty:hover .dot { visibility: hidden; }
.close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  border-radius: 3px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  opacity: 0;
}
.tab:hover .close { opacity: 1; }
.close:hover { background: var(--ed-row-hover, rgba(255, 255, 255, 0.1)); }
/* 脏标签的关闭按钮盖在圆点上,布局不跳。 */
.close-dirty { position: absolute; right: 6px; }
</style>
