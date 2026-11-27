<script lang="ts" setup>
import { useI18n } from "vue-i18n";
import { FONT_SIZE_RANGE, TAB_SIZE_RANGE } from "./editorPrefs.js";

const { t } = useI18n();

interface Prefs {
  lineNumbers: boolean;
  wrap: boolean;
  tabSize: number;
  fontSize: number;
  showHidden: boolean;
}

const props = defineProps<{ prefs: Prefs }>();

const emit = defineEmits<{
  (e: "update", patch: Partial<Prefs>): void;
  (e: "close"): void;
}>();

function setNumber(key: "tabSize" | "fontSize", value: string) {
  const n = Number.parseInt(value, 10);
  if (Number.isFinite(n)) emit("update", { [key]: n } as Partial<Prefs>);
}
</script>

<template>
  <div class="prefs-scrim" @click.self="emit('close')" @contextmenu.prevent="emit('close')">
    <div class="prefs" data-test="editor-prefs-panel" role="dialog" :aria-label="t('files.preferences')">
      <label class="row">
        <input
          type="checkbox"
          data-test="prefs-line-numbers"
          :checked="props.prefs.lineNumbers"
          @change="emit('update', { lineNumbers: ($event.target as HTMLInputElement).checked })"
        >
        <span>{{ t('files.prefs.lineNumbers') }}</span>
      </label>
      <label class="row">
        <input
          type="checkbox"
          data-test="prefs-wrap"
          :checked="props.prefs.wrap"
          @change="emit('update', { wrap: ($event.target as HTMLInputElement).checked })"
        >
        <span>{{ t('files.prefs.wrap') }}</span>
      </label>
      <label class="row">
        <input
          type="checkbox"
          data-test="prefs-show-hidden"
          :checked="props.prefs.showHidden"
          @change="emit('update', { showHidden: ($event.target as HTMLInputElement).checked })"
        >
        <span>{{ t('files.prefs.showHidden') }}</span>
      </label>
      <label class="row">
        <span class="label">{{ t('files.prefs.tabSize') }}</span>
        <input
          class="num"
          type="number"
          data-test="prefs-tab-size"
          :min="TAB_SIZE_RANGE.min"
          :max="TAB_SIZE_RANGE.max"
          :value="props.prefs.tabSize"
          @change="setNumber('tabSize', ($event.target as HTMLInputElement).value)"
        >
      </label>
      <label class="row">
        <span class="label">{{ t('files.prefs.fontSize') }}</span>
        <input
          class="num"
          type="number"
          data-test="prefs-font-size"
          :min="FONT_SIZE_RANGE.min"
          :max="FONT_SIZE_RANGE.max"
          :value="props.prefs.fontSize"
          @change="setNumber('fontSize', ($event.target as HTMLInputElement).value)"
        >
      </label>
    </div>
  </div>
</template>

<style scoped>
.prefs-scrim { position: fixed; inset: 0; z-index: 60; }
.prefs {
  position: absolute;
  top: 64px;
  right: 12px;
  min-width: 200px;
  padding: 8px;
  background: var(--ed-shell-bg, #22272e);
  color: var(--ed-row-fg, #adbac7);
  border: 1px solid var(--ed-border, #444c56);
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.35);
  font-size: 12px;
}
.row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 6px;
  border-radius: 3px;
  cursor: pointer;
}
.row:hover { background: var(--ed-row-hover, rgba(255, 255, 255, 0.06)); }
.label { flex: 1 1 auto; }
.num {
  width: 56px;
  background: var(--ed-editor-bg, #22272e);
  color: inherit;
  border: 1px solid var(--ed-border, #444c56);
  border-radius: 3px;
  font: inherit;
  padding: 1px 4px;
}
</style>
