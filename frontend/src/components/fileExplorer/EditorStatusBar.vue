<script lang="ts" setup>
import { nextTick, ref } from "vue";

const props = defineProps<{
  line: number;
  column: number;
  selected: number;
  lines: number;
  /** 只有代码/文本视图才显示行列;图片、PDF 等只显示文件类型。 */
  showCursor: boolean;
  kindLabel: string;
}>();

const emit = defineEmits<{
  (e: "goto", line: number): void;
}>();

const gotoOpen = ref(false);
const gotoValue = ref("");
const gotoInput = ref<HTMLInputElement | null>(null);

async function openGoto() {
  gotoOpen.value = true;
  gotoValue.value = String(props.line);
  await nextTick();
  gotoInput.value?.focus();
  gotoInput.value?.select();
}

function submitGoto() {
  const n = Number.parseInt(gotoValue.value, 10);
  gotoOpen.value = false;
  if (Number.isFinite(n) && n > 0) emit("goto", n);
}

function onKey(e: KeyboardEvent) {
  if (e.key === "Enter") submitGoto();
  else if (e.key === "Escape") gotoOpen.value = false;
}

defineExpose({ openGoto });
</script>

<template>
  <div class="status" data-test="editor-status-bar">
    <span class="kind">{{ kindLabel }}</span>
    <span class="spacer" />
    <template v-if="showCursor">
      <span v-if="selected > 0" class="item">已选 {{ selected }} 字符</span>
      <button
        v-if="!gotoOpen"
        class="item link"
        data-test="status-goto"
        title="跳转到行"
        @click="openGoto"
      >
        行 {{ line }}:{{ column }} / {{ lines }}
      </button>
      <span v-else class="item">
        <input
          ref="gotoInput"
          v-model="gotoValue"
          class="goto-input"
          data-test="status-goto-input"
          type="text"
          inputmode="numeric"
          @keydown="onKey"
          @blur="gotoOpen = false"
        >
      </span>
      <span class="item">UTF-8</span>
    </template>
  </div>
</template>

<style scoped>
.status {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 20px;
  padding: 0 10px;
  border-top: 1px solid var(--ed-border, #444c56);
  background: var(--ed-shell-bg, #22272e);
  color: var(--ed-muted, rgba(173, 186, 199, 0.7));
  font-size: 11px;
}
.spacer { flex: 1 1 auto; }
.item { white-space: nowrap; }
.link {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  padding: 0 2px;
  border-radius: 2px;
  cursor: pointer;
}
.link:hover { background: var(--ed-row-hover, rgba(255, 255, 255, 0.08)); color: var(--ed-row-fg, #adbac7); }
.goto-input {
  width: 60px;
  background: var(--ed-editor-bg, #22272e);
  color: var(--ed-row-fg, #adbac7);
  border: 1px solid var(--ed-tab-active-bar, #539bf5);
  border-radius: 2px;
  font: inherit;
  padding: 0 4px;
  outline: none;
}
</style>
