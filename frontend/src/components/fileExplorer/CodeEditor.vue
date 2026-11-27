<script lang="ts" setup>
import { useI18n } from "vue-i18n";
import { computed, onMounted, onBeforeUnmount, ref, watch } from "vue";
import { Compartment, EditorState, type Extension } from "@codemirror/state";
import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { history, defaultKeymap, historyKeymap } from "@codemirror/commands";
import { highlightSelectionMatches, search, searchKeymap } from "@codemirror/search";
import { languageForPath } from "./languageMap";
import { highlightExtensionFor } from "./highlight";
import type { FileSystemBridge } from "./fsBridge";
import { editorPhrases, reconfigureEditorPhrases } from "./codeMirrorI18n.js";

const { t } = useI18n();

const MAX_BYTES_FRONTEND = 2 * 1024 * 1024;

export interface EditorPrefs {
  lineNumbers: boolean;
  wrap: boolean;
  tabSize: number;
  fontSize: number;
}

const props = defineProps<{
  fs: FileSystemBridge;
  path: string;
  prefs: EditorPrefs;
  theme: "dimmed" | "light";
}>();

const emit = defineEmits<{
  (e: "dirty-change", dirty: boolean): void;
  (e: "cursor-change", pos: { line: number; column: number; selected: number; lines: number }): void;
}>();

const host = ref<HTMLDivElement | null>(null);
const state = ref<"loading" | "tooLarge" | "binary" | "ok" | "error">("loading");
const errorMsg = ref<string>("");
const errorFallback = ref("loadFailed");
const reloadPending = ref(false);
const loadedAt = ref<number | null>(null);
const dirty = ref(false);
const conflict = ref<{ currentModTime: number } | null>(null);
const saveError = ref<string | null>(null);

let view: EditorView | null = null;
let off: (() => void) | null = null;
let disposed = false;
let loadGeneration = 0;
let originalText = "";

// 偏好与主题走 Compartment 热切换,而不是重建编辑器:
// 以前改行号/切主题会走一遍 load(),把未保存的改动直接丢掉。
const numbersComp = new Compartment();
const wrapComp = new Compartment();
const tabComp = new Compartment();
const themeComp = new Compartment();
const phrasesComp = new Compartment();
const phrases = computed(() => editorPhrases(t));

function isCurrent(fs: FileSystemBridge, path: string, request: number): boolean {
  return !disposed && loadGeneration === request && props.fs === fs && props.path === path;
}

function cssVar(name: string, fallback: string): string {
  if (typeof window === "undefined" || !host.value) return fallback;
  return getComputedStyle(host.value).getPropertyValue(name).trim() || fallback;
}

function decodeFileBytes(data: unknown): string {
  let bytes: Uint8Array;
  if (typeof data === "string") {
    const bin = atob(data);
    bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  } else if (data instanceof Uint8Array) {
    bytes = data;
  } else if (Array.isArray(data)) {
    bytes = new Uint8Array(data as number[]);
  } else {
    throw Object.assign(new Error(), { code: "unexpectedContentType" });
  }
  return new TextDecoder().decode(bytes);
}

function makeThemeExt(theme: "dimmed" | "light", fontSize: number): Extension {
  return EditorView.theme(
    {
      "&": {
        backgroundColor: cssVar("--ed-editor-bg", "#22272e"),
        color: cssVar("--ed-editor-fg", "#adbac7"),
        height: "100%",
      },
      ".cm-gutters": {
        backgroundColor: cssVar("--ed-gutter-bg", "#22272e"),
        color: cssVar("--ed-gutter-fg", "#545d68"),
        border: "none",
      },
      ".cm-activeLine": { backgroundColor: "transparent" },
      ".cm-activeLineGutter": { backgroundColor: "transparent" },
      ".cm-content": {
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
        fontSize: `${fontSize}px`,
        padding: "8px 0",
      },
      ".cm-lineNumbers .cm-gutterElement": { padding: "0 8px 0 14px" },
    },
    { dark: theme !== "light" },
  );
}

function themeExtensions(theme: "dimmed" | "light", fontSize: number): Extension {
  return [makeThemeExt(theme, fontSize), highlightExtensionFor(theme)];
}

function setDirty(next: boolean) {
  if (dirty.value === next) return;
  dirty.value = next;
  emit("dirty-change", next);
}

function emitCursor(v: EditorView) {
  const head = v.state.selection.main;
  const line = v.state.doc.lineAt(head.head);
  emit("cursor-change", {
    line: line.number,
    column: head.head - line.from + 1,
    selected: Math.abs(head.to - head.from),
    lines: v.state.doc.lines,
  });
}

async function load() {
  const fs = props.fs;
  const path = props.path;
  const request = ++loadGeneration;
  if (disposed) return;
  state.value = "loading";
  view?.destroy();
  view = null;
  try {
    const meta = (await fs.fileMeta(path)) as { size: number; modTime: number; isBinary: boolean };
    if (!isCurrent(fs, path, request)) return;
    loadedAt.value = meta.modTime;
    reloadPending.value = false;
    if (meta.isBinary) { state.value = "binary"; setDirty(false); return; }
    if (meta.size > MAX_BYTES_FRONTEND) { state.value = "tooLarge"; setDirty(false); return; }
    const result = await fs.readBytes(path, MAX_BYTES_FRONTEND);
    if (!isCurrent(fs, path, request)) return;
    const text = decodeFileBytes(result.data);

    originalText = text;
    setDirty(false);
    conflict.value = null;
    saveError.value = null;

    const updateListener = EditorView.updateListener.of((v) => {
      if (v.docChanged) setDirty(v.state.doc.toString() !== originalText);
      if (v.docChanged || v.selectionSet) emitCursor(v.view);
    });

    const saveKey = keymap.of([
      {
        key: "Mod-s",
        preventDefault: true,
        run: () => {
          void save();
          return true;
        },
      },
    ]);

    const exts: Extension[] = [
      themeComp.of(themeExtensions(props.theme, props.prefs.fontSize)),
      numbersComp.of(props.prefs.lineNumbers ? lineNumbers() : []),
      wrapComp.of(props.prefs.wrap ? EditorView.lineWrapping : []),
      tabComp.of(EditorState.tabSize.of(props.prefs.tabSize)),
      search(),
      highlightSelectionMatches(),
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap]),
      saveKey,
      updateListener,
    ];
    const langExt = await languageForPath(path);
    if (langExt) exts.push(langExt);
    exts.push(phrasesComp.of(EditorState.phrases.of(phrases.value)));

    if (!isCurrent(fs, path, request) || !host.value) return;
    const nextView = new EditorView({
      state: EditorState.create({ doc: text, extensions: exts }),
      parent: host.value,
    });
    if (!isCurrent(fs, path, request)) {
      nextView.destroy();
      return;
    }
    view = nextView;
    state.value = "ok";
    emitCursor(nextView);
  } catch (err) {
    if (!isCurrent(fs, path, request)) return;
    state.value = "error";
    errorMsg.value = (typeof err === "string" ? err : (err as Error)?.message) || "";
    errorFallback.value = (err as { code?: string })?.code === "unexpectedContentType" ? "unexpectedContentType" : "loadFailed";
  }
}

async function save(): Promise<boolean> {
  if (!view || state.value !== "ok") return false;
  const text = view.state.doc.toString();
  const bytes = new TextEncoder().encode(text);
  saveError.value = null;
  try {
    const newModTime = await props.fs.writeBytes(props.path, bytes, loadedAt.value ?? 0, false);
    originalText = text;
    loadedAt.value = newModTime;
    reloadPending.value = false;
    conflict.value = null;
    setDirty(false);
    return true;
  } catch (err) {
    const msg = (typeof err === "string" ? err : (err as Error)?.message) ?? "";
    const m = /stale_modtime: current=(\d+)/.exec(msg);
    if (m) {
      conflict.value = { currentModTime: Number(m[1]) };
    } else {
      saveError.value = msg;
    }
    return false;
  }
}

// revert 把文档退回上次加载/保存时的内容,不碰磁盘。撤销栈保留,可以再撤回来。
function revert() {
  if (!view) return;
  view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: originalText } });
  setDirty(false);
}

function gotoLine(line: number) {
  if (!view) return;
  const target = Math.min(Math.max(1, Math.round(line)), view.state.doc.lines);
  const pos = view.state.doc.line(target).from;
  view.dispatch({ selection: { anchor: pos }, scrollIntoView: true });
  view.focus();
}

async function overwriteWithServerModTime() {
  if (!conflict.value) return;
  loadedAt.value = conflict.value.currentModTime;
  conflict.value = null;
  await save();
}

function reloadDiscardChanges() {
  conflict.value = null;
  void load();
}

async function checkDirChanged(fs: FileSystemBridge, dir: string) {
  const path = props.path;
  const request = loadGeneration;
  if (!path.startsWith(dir + "/") && path !== dir) return;
  try {
    const meta = (await fs.fileMeta(path)) as { modTime: number };
    if (disposed || props.fs !== fs || props.path !== path || loadGeneration !== request) return;
    if (loadedAt.value && meta.modTime > loadedAt.value) reloadPending.value = true;
  } catch { /* ignore */ }
}

function subscribeToDirChanges(fs: FileSystemBridge) {
  off?.();
  off = fs.onDirChanged((dir) => {
    void checkDirChanged(fs, dir);
  });
}

onMounted(() => {
  disposed = false;
  void load();
  subscribeToDirChanges(props.fs);
});

// 只有换文件/换项目才重新加载。主题与偏好走 Compartment,不能丢文档。
watch(() => [props.path, props.fs], () => { void load(); });

watch(
  () => [props.theme, props.prefs.fontSize],
  () => {
    view?.dispatch({
      effects: themeComp.reconfigure(themeExtensions(props.theme, props.prefs.fontSize)),
    });
  },
);
watch(() => props.prefs.lineNumbers, (on) => {
  view?.dispatch({ effects: numbersComp.reconfigure(on ? lineNumbers() : []) });
});
watch(() => props.prefs.wrap, (on) => {
  view?.dispatch({ effects: wrapComp.reconfigure(on ? EditorView.lineWrapping : []) });
});
watch(() => props.prefs.tabSize, (size) => {
  view?.dispatch({ effects: tabComp.reconfigure(EditorState.tabSize.of(size)) });
});
watch(phrases, (value) => {
  if (!view) return;
  reconfigureEditorPhrases(view, phrasesComp.reconfigure(EditorState.phrases.of(value)));
});

watch(() => props.fs, (fs) => subscribeToDirChanges(fs));

onBeforeUnmount(() => {
  disposed = true;
  loadGeneration++;
  view?.destroy();
  view = null;
  if (off) off();
});

// testAppend is used by unit tests to simulate a doc edit without a live
// DOM keyboard event. It's a thin no-op wrapper in production.
defineExpose({
  save,
  revert,
  gotoLine,
  focus: () => view?.focus(),
  testAppend: (text: string) => {
    if (!view) return;
    view.dispatch({ changes: { from: view.state.doc.length, insert: text } });
  },
});
</script>

<template>
  <div class="file-editor">
    <div v-if="conflict" class="banner err conflict" data-test="conflict-banner">
      <span class="msg">{{ t("files.staleModTime") }}</span>
      <button data-test="conflict-overwrite" @click="overwriteWithServerModTime">
        {{ t("files.overwrite") }}
      </button>
      <button data-test="conflict-reload" @click="reloadDiscardChanges">
        {{ t("files.reloadDiscard") }}
      </button>
      <button data-test="conflict-cancel" @click="conflict = null">
        {{ t("files.cancel") }}
      </button>
    </div>
    <div v-if="saveError !== null" class="banner err" data-test="save-error">
      {{ t("files.saveFailed", { message: saveError || t("files.errors.saveFailed") }) }}
    </div>
    <div v-if="reloadPending" class="reload-badge">
      {{ t("files.fileChanged") }}
      <button @click="load">
        {{ dirty ? t("files.reloadDiscard") : t("files.reload") }}
      </button>
    </div>
    <div v-if="state === 'tooLarge'" class="banner muted">{{ t("files.tooLarge") }}</div>
    <div v-else-if="state === 'binary'" class="banner muted">{{ t("files.binary") }}</div>
    <div v-else-if="state === 'error'" class="banner err">{{ t("files.errorPrefix", { message: errorMsg || t(`files.errors.${errorFallback}`) }) }}</div>
    <div v-else-if="state === 'loading'" class="banner muted">{{ t("files.loading") }}</div>
    <div v-show="state === 'ok'" ref="host" class="cm-host" />
  </div>
</template>

<style scoped>
.file-editor {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--ed-editor-bg, #22272e);
}
.cm-host { flex: 1 1 auto; overflow: auto; }
.cm-host::-webkit-scrollbar,
.cm-host :deep(.cm-scroller)::-webkit-scrollbar {
  width: 12px;
  height: 12px;
}
.cm-host::-webkit-scrollbar-track,
.cm-host :deep(.cm-scroller)::-webkit-scrollbar-track {
  background: var(--ed-editor-bg, #22272e);
}
.cm-host::-webkit-scrollbar-thumb,
.cm-host :deep(.cm-scroller)::-webkit-scrollbar-thumb {
  background: var(--ed-row-hover, rgba(173, 186, 199, 0.2));
  border-radius: 6px;
  border: 3px solid var(--ed-editor-bg, #22272e);
}
.cm-host::-webkit-scrollbar-thumb:hover,
.cm-host :deep(.cm-scroller)::-webkit-scrollbar-thumb:hover {
  background: var(--ed-chevron, rgba(173, 186, 199, 0.4));
}
.cm-host::-webkit-scrollbar-corner,
.cm-host :deep(.cm-scroller)::-webkit-scrollbar-corner {
  background: var(--ed-editor-bg, #22272e);
}
.banner { padding: 8px 12px; font-size: 13px; }
.muted { color: var(--ed-muted, rgba(173, 186, 199, 0.5)); }
.err { color: var(--ed-error, #f47067); }
.conflict {
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--ed-border, #444c56);
}
.conflict .msg { flex: 1 1 auto; }
.conflict button {
  background: var(--ed-shell-bg, #22272e);
  border: 1px solid var(--ed-border, #444c56);
  color: var(--ed-row-fg, #adbac7);
  padding: 1px 8px;
  border-radius: 3px;
  cursor: pointer;
}
.conflict button:hover { background: var(--ed-row-hover, rgba(173, 186, 199, 0.1)); }
.reload-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  background: var(--ed-tab-bg, #2d333b);
  border-bottom: 1px solid var(--ed-border, #444c56);
  font-size: 11px;
  color: var(--ed-muted, rgba(173, 186, 199, 0.7));
}
.reload-badge button {
  background: var(--ed-shell-bg, #22272e);
  border: 1px solid var(--ed-border, #444c56);
  color: var(--ed-row-fg, #adbac7);
  padding: 1px 8px;
  border-radius: 3px;
  cursor: pointer;
}
</style>
