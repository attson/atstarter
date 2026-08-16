<script lang="ts">
import { defineComponent } from "vue";
// Self-name for recursive template usage.
export default defineComponent({ name: "FileTreeNode" });
</script>

<script lang="ts" setup>
import { computed } from "vue";
import {
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  File,
  FileCode2,
  FileText,
  Braces,
  Image,
} from "lucide-vue-next";
import InlineEditRow from "./InlineEditRow.vue";

interface TreeNode {
  path: string;
  name: string;
  isDir: boolean;
  expanded: boolean;
  children: TreeNode[] | null;
}

type InlineIntent =
  | { kind: "newFile"; parentPath: string; parentLevel: number }
  | { kind: "newFolder"; parentPath: string; parentLevel: number }
  | { kind: "rename"; node: TreeNode; level: number };

const props = defineProps<{
  node: TreeNode;
  level: number;
  selectedPaths: Set<string>;
  focusedPath: string;
  /** 剪切中的路径:粘贴前显示为半透明,和资源管理器一致。 */
  cutPaths: Set<string>;
  /** 当前拖拽悬停的落点目录,用来高亮。 */
  dropTarget: string;
  inlineIntent?: InlineIntent | null;
}>();

const emit = defineEmits<{
  (e: "row-click", ev: MouseEvent, n: TreeNode): void;
  (e: "row-dblclick", ev: MouseEvent, n: TreeNode): void;
  (e: "context", ev: MouseEvent, n: TreeNode, level: number): void;
  (e: "drag-start", ev: DragEvent, n: TreeNode): void;
  (e: "drag-over", ev: DragEvent, n: TreeNode): void;
  (e: "drag-leave", ev: DragEvent, n: TreeNode): void;
  (e: "drop-on", ev: DragEvent, n: TreeNode): void;
  (e: "inline-submit", value: string): void;
  (e: "inline-cancel"): void;
}>();

const renamingHere = computed(() =>
  props.inlineIntent?.kind === "rename" && props.inlineIntent.node.path === props.node.path,
);

const insertNewHere = computed(() => {
  const intent = props.inlineIntent;
  if (!intent) return null;
  if (intent.kind !== "newFile" && intent.kind !== "newFolder") return null;
  return intent.parentPath === props.node.path ? intent : null;
});

const INDENT_PX = 8;
const ROW_HEIGHT = 22;

const fileIconMap: Record<string, { comp: any; color: string }> = {
  ts: { comp: FileCode2, color: "#3178c6" },
  tsx: { comp: FileCode2, color: "#3178c6" },
  js: { comp: FileCode2, color: "#f7df1e" },
  jsx: { comp: FileCode2, color: "#f7df1e" },
  go: { comp: FileCode2, color: "#00add8" },
  py: { comp: FileCode2, color: "#3776ab" },
  rs: { comp: FileCode2, color: "#dea584" },
  sh: { comp: FileCode2, color: "#89e051" },
  vue: { comp: FileCode2, color: "#41b883" },
  json: { comp: Braces, color: "#cbcb41" },
  md: { comp: FileText, color: "#519aba" },
  markdown: { comp: FileText, color: "#519aba" },
  txt: { comp: FileText, color: "#cccccc" },
  yaml: { comp: FileCode2, color: "#cb171e" },
  yml: { comp: FileCode2, color: "#cb171e" },
  toml: { comp: FileCode2, color: "#9c4221" },
  html: { comp: FileCode2, color: "#e44d26" },
  htm: { comp: FileCode2, color: "#e44d26" },
  css: { comp: FileCode2, color: "#264de4" },
  scss: { comp: FileCode2, color: "#c6538c" },
  sass: { comp: FileCode2, color: "#c6538c" },
  png: { comp: Image, color: "#a074c4" },
  jpg: { comp: Image, color: "#a074c4" },
  jpeg: { comp: Image, color: "#a074c4" },
  gif: { comp: Image, color: "#a074c4" },
  svg: { comp: Image, color: "#ffb13b" },
  ico: { comp: Image, color: "#a074c4" },
};

function fileIconFor(name: string) {
  const m = /\.([A-Za-z0-9]+)$/.exec(name);
  if (m) {
    const entry = fileIconMap[m[1].toLowerCase()];
    if (entry) return entry;
  }
  return { comp: File, color: "currentColor" };
}

function cssVar(name: string, fallback: string): string {
  if (typeof window === "undefined") return fallback;
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}

const folderColor = computed(() => cssVar("--ed-folder", "#d4a14a"));
const chevronColor = computed(() => cssVar("--ed-chevron", "rgba(173,186,199,0.7)"));

const isSelected = computed(() => props.selectedPaths.has(props.node.path));
const isFocused = computed(() => props.focusedPath === props.node.path);
const isCut = computed(() => props.cutPaths.has(props.node.path));
const isDropTarget = computed(() => props.dropTarget !== "" && props.dropTarget === props.node.path);
const dirChevron = computed(() => (props.node.expanded ? ChevronDown : ChevronRight));
const dirIcon = computed(() => (props.node.expanded ? FolderOpen : Folder));
const fileIcon = computed(() => fileIconFor(props.node.name));
</script>

<template>
  <div class="node-wrap">
    <InlineEditRow
      v-if="renamingHere"
      :level="level"
      :icon="node.isDir ? 'folder' : 'file'"
      :initial-value="node.name"
      @submit="(v) => emit('inline-submit', v)"
      @cancel="emit('inline-cancel')"
    />
    <div
      v-else
      class="node"
      :class="{
        selected: isSelected,
        focused: isFocused,
        cut: isCut,
        'drop-target': isDropTarget,
        'is-dir': node.isDir,
        'is-file': !node.isDir,
      }"
      :data-type="node.isDir ? 'dir' : 'file'"
      :data-path="node.path"
      draggable="true"
      :style="{
        paddingLeft: `${level * INDENT_PX}px`,
        height: `${ROW_HEIGHT}px`,
        lineHeight: `${ROW_HEIGHT}px`,
      }"
      :title="node.path"
      @click="(ev) => emit('row-click', ev, node)"
      @dblclick="(ev) => emit('row-dblclick', ev, node)"
      @contextmenu="(ev) => emit('context', ev, node, level)"
      @dragstart="(ev) => emit('drag-start', ev, node)"
      @dragover="(ev) => emit('drag-over', ev, node)"
      @dragleave="(ev) => emit('drag-leave', ev, node)"
      @drop="(ev) => emit('drop-on', ev, node)"
    >
      <span class="twisty">
        <component
          v-if="node.isDir"
          :is="dirChevron"
          :size="14"
          :color="chevronColor"
          :stroke-width="2"
        />
      </span>
      <span class="icon">
        <component
          v-if="node.isDir"
          :is="dirIcon"
          :size="16"
          :color="folderColor"
          :stroke-width="1.5"
        />
        <component
          v-else
          :is="fileIcon.comp"
          :size="16"
          :color="fileIcon.color"
          :stroke-width="1.5"
        />
      </span>
      <span class="node-name">{{ node.name }}</span>
    </div>
    <ul
      v-if="node.isDir && node.expanded && node.children"
      class="tree-list"
      :style="{ '--indent-base': `${level * INDENT_PX + 12}px` }"
    >
      <li v-if="insertNewHere">
        <InlineEditRow
          :level="level + 1"
          :icon="insertNewHere.kind === 'newFolder' ? 'folder' : 'file'"
          @submit="(v) => emit('inline-submit', v)"
          @cancel="emit('inline-cancel')"
        />
      </li>
      <li v-for="c in node.children" :key="c.path">
        <FileTreeNode
          :node="c"
          :level="level + 1"
          :selected-paths="selectedPaths"
          :focused-path="focusedPath"
          :cut-paths="cutPaths"
          :drop-target="dropTarget"
          :inline-intent="inlineIntent"
          @row-click="(ev: MouseEvent, n: TreeNode) => emit('row-click', ev, n)"
          @row-dblclick="(ev: MouseEvent, n: TreeNode) => emit('row-dblclick', ev, n)"
          @context="(ev: MouseEvent, n: TreeNode, l: number) => emit('context', ev, n, l)"
          @drag-start="(ev: DragEvent, n: TreeNode) => emit('drag-start', ev, n)"
          @drag-over="(ev: DragEvent, n: TreeNode) => emit('drag-over', ev, n)"
          @drag-leave="(ev: DragEvent, n: TreeNode) => emit('drag-leave', ev, n)"
          @drop-on="(ev: DragEvent, n: TreeNode) => emit('drop-on', ev, n)"
          @inline-submit="(v: string) => emit('inline-submit', v)"
          @inline-cancel="() => emit('inline-cancel')"
        />
      </li>
    </ul>
  </div>
</template>

<style scoped>
.node-wrap { display: block; }
.tree-list {
  list-style: none;
  margin: 0;
  padding: 0;
  position: relative;
}
.tree-list::before {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--ed-indent-guide, rgba(204, 204, 204, 0.1));
  left: var(--indent-base, 12px);
  pointer-events: none;
}
.tree-list > li { display: block; }
.node {
  display: flex;
  align-items: center;
  padding-right: 6px;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
  color: var(--ed-row-fg, #cccccc);
  user-select: none;
}
.node:hover { background: var(--ed-row-hover, rgba(255, 255, 255, 0.05)); }
.node.selected { background: var(--ed-row-selected, #37373d); }
/* 焦点行画描边而不是换底色:多选时选中底色仍要看得见。 */
.node.focused { box-shadow: inset 0 0 0 1px var(--ed-tab-active-bar, #539bf5); }
.node.cut { opacity: 0.5; }
.node.drop-target { background: var(--ed-row-selected, #37373d); outline: 1px dashed var(--ed-tab-active-bar, #539bf5); outline-offset: -1px; }
.twisty {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 14px;
  width: 14px;
  height: 100%;
  color: var(--ed-chevron, rgba(204, 204, 204, 0.7));
}
.icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 20px;
  width: 20px;
  margin-right: 4px;
}
.node-name {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
