<script lang="ts" setup>
import { computed, nextTick, ref, watch, onMounted, onBeforeUnmount } from "vue";
import FileTreeNode from "./FileTreeNode.vue";
import ConfirmDialog from "./FileConfirmDialog.vue";
import InlineEditRow from "./InlineEditRow.vue";
import type { FileSystemBridge } from "./fsBridge";
import { baseName, isDescendant, joinPath, parentDir, targetDirFor } from "./pathOps.js";
import { fileClipboard, planDrop, planPaste } from "./fileClipboard.js";
import { placeMenu } from "./menuPlacement.js";
import {
  findRow,
  flattenVisible,
  moveFocus,
  rangePaths,
  toggleSelection,
  treeKeyAction,
} from "./treeKeyboard.js";

// 中文文案(替代 atterm 的 i18n)。
const T = {
  newFile: "新建文件",
  newFolder: "新建文件夹",
  copy: "复制",
  cut: "剪切",
  paste: "粘贴",
  duplicate: "创建副本",
  copyRelPath: "复制相对路径",
  copyAbsPath: "复制绝对路径",
  reveal: "在文件管理器中显示",
  rename: "重命名",
  delete: "删除",
  moveToTrash: "移到废纸篓",
  cancel: "取消",
  confirmHardDelete: (name: string) => `确定永久删除「${name}」?此操作不可恢复。`,
  confirmTrash: (name: string) => `确定把「${name}」移到废纸篓?`,
  confirmHardDeleteMany: (n: number) => `确定永久删除这 ${n} 项?此操作不可恢复。`,
  confirmTrashMany: (n: number) => `确定把这 ${n} 项移到废纸篓?`,
};

interface DirEntry {
  name: string;
  isDir: boolean;
  size?: number;
  modTime?: number;
}

interface TreeNode {
  path: string;
  name: string;
  isDir: boolean;
  expanded: boolean;
  children: TreeNode[] | null;
}

const props = defineProps<{
  fs: FileSystemBridge;
  root: string;
  showHidden: boolean;
}>();

const emit = defineEmits<{
  (e: "file-clicked", path: string): void;
  (e: "file-double-clicked", path: string): void;
  (e: "dir-toggled", path: string, expanded: boolean): void;
  /** 路径被重命名或同项目内移动;编辑器标签要跟着改写。 */
  (e: "path-renamed", from: string, to: string): void;
  /** 路径被删除;编辑器要关掉它和它下面的标签。 */
  (e: "path-removed", path: string): void;
  (e: "error", message: string): void;
}>();

type ContextMenuAnchor = {
  x: number;
  y: number;
  node: TreeNode | null; // null = 空白区/根
  level: number;
  shift: boolean;
};
const menu = ref<ContextMenuAnchor | null>(null);
const menuRef = ref<HTMLDivElement | null>(null);
// 菜单先以点击点为准渲染但不可见,量到真实尺寸后再定位 —— 高度取决于菜单项
// 多少(根菜单和节点菜单不一样长),只能实测,不能写死。
const menuStyle = ref<Record<string, string>>({ visibility: "hidden" });

watch(menu, async (anchor) => {
  if (!anchor) return;
  menuStyle.value = { left: `${anchor.x}px`, top: `${anchor.y}px`, visibility: "hidden" };
  await nextTick();
  const el = menuRef.value;
  if (!el || menu.value !== anchor) return;
  const rect = el.getBoundingClientRect();
  const placed = placeMenu(
    { x: anchor.x, y: anchor.y },
    { width: rect.width, height: rect.height },
    { width: window.innerWidth, height: window.innerHeight },
  );
  menuStyle.value = {
    left: `${placed.left}px`,
    top: `${placed.top}px`,
    maxHeight: `${placed.maxHeight}px`,
    visibility: "visible",
  };
});

type InlineIntent =
  | { kind: "newFile"; parentPath: string; parentLevel: number }
  | { kind: "newFolder"; parentPath: string; parentLevel: number }
  | { kind: "rename"; node: TreeNode; level: number };
const inlineIntent = ref<InlineIntent | null>(null);

type DeleteConfirmSpec = { nodes: TreeNode[]; mode: "trash" | "hard" };
const deleteConfirm = ref<DeleteConfirmSpec | null>(null);

const rootNodes = ref<TreeNode[]>([]);
const selectedPaths = ref<Set<string>>(new Set());
const focusedPath = ref<string>("");
const selectionAnchor = ref<string>("");
const cutPaths = ref<Set<string>>(new Set());
const dropTarget = ref<string>("");
const rootDropActive = ref(false);
const treeError = ref<string>("");
const clipboardEntry = ref(fileClipboard.read());
const wrapRef = ref<HTMLDivElement | null>(null);

let dragItems: Array<{ path: string; isDir: boolean }> = [];
let errorTimer: ReturnType<typeof setTimeout> | null = null;
let offClipboard: () => void = () => {};

function showError(message: string) {
  treeError.value = message;
  emit("error", message);
  if (errorTimer) clearTimeout(errorTimer);
  errorTimer = setTimeout(() => { treeError.value = ""; }, 6000);
}

// runFsOp 是所有文件操作的唯一出口:失败必须让用户看见,不能只 console.warn。
async function runFsOp<T>(action: () => Promise<T>): Promise<T | null> {
  try {
    return await action();
  } catch (err) {
    showError((err as Error)?.message || "操作失败");
    return null;
  }
}

/* ---------------------------------------------------------------- 树数据 */

async function loadDir(fs: FileSystemBridge, path: string, showHidden: boolean): Promise<TreeNode[]> {
  const entries = (await fs.listDir(path)) as DirEntry[];
  return entries
    .filter((e) => showHidden || !e.name.startsWith("."))
    .map((e) => ({
      path: joinPath(path, e.name),
      name: e.name,
      isDir: e.isDir,
      expanded: false,
      children: null,
    }))
    .sort((a, b) => (a.isDir === b.isDir ? a.name.localeCompare(b.name) : a.isDir ? -1 : 1));
}

// mergeNodes 用新列表刷新一层,但复用仍然存在的旧节点对象,以保住它们的
// expanded/children —— 否则根目录一有风吹草动整棵树就会全部收起来。
// 返回消失的路径,调用方据此释放对应的 watcher 与待办状态。
function mergeNodes(prev: TreeNode[], next: TreeNode[]): { nodes: TreeNode[]; removed: string[] } {
  const byPath = new Map(prev.map((n) => [n.path, n]));
  const nodes = next.map((n) => {
    const old = byPath.get(n.path);
    if (old && old.isDir === n.isDir) {
      byPath.delete(n.path);
      return old;
    }
    byPath.delete(n.path);
    return n;
  });
  return { nodes, removed: [...byPath.keys()] };
}

const watchHandles = new Map<string, { fs: FileSystemBridge; id: number | string }>();
const pendingExpands = new Map<string, number>();
const watchGenerations = new Map<string, number>();
const refreshGenerations = new Map<string, number>();
let disposed = false;
let generation = 0;
let offDirChanged: () => void = () => {};

function isCurrent(fs: FileSystemBridge, root: string, showHidden: boolean, request: number): boolean {
  return !disposed
    && generation === request
    && props.fs === fs
    && props.root === root
    && props.showHidden === showHidden;
}

async function unwatch(fs: FileSystemBridge, id: number | string): Promise<void> {
  try { await fs.unwatchDir(id as number); } catch { /* ignore */ }
}

function releaseWatches() {
  const handles = Array.from(watchHandles.values());
  watchHandles.clear();
  for (const { fs, id } of handles) void unwatch(fs, id);
}

// releaseSubtreeState 释放某个路径自身及其子孙的 watcher 与代际标记。
function releaseSubtreeState(path: string) {
  for (const [p, handle] of Array.from(watchHandles)) {
    if (p !== path && !isDescendant(p, path)) continue;
    watchHandles.delete(p);
    void unwatch(handle.fs, handle.id);
  }
  for (const p of Array.from(pendingExpands.keys())) {
    if (p === path || isDescendant(p, path)) pendingExpands.delete(p);
  }
  for (const p of Array.from(watchGenerations.keys())) {
    if (p === path || isDescendant(p, path)) advanceWatchGeneration(p);
  }
  for (const p of Array.from(refreshGenerations.keys())) {
    if (p === path || isDescendant(p, path)) advanceRefreshGeneration(p);
  }
}

function advanceWatchGeneration(path: string): number {
  const next = (watchGenerations.get(path) ?? 0) + 1;
  watchGenerations.set(path, next);
  return next;
}

function advanceRefreshGeneration(path: string): number {
  const next = (refreshGenerations.get(path) ?? 0) + 1;
  refreshGenerations.set(path, next);
  return next;
}

function collapseDescendants(node: TreeNode) {
  for (const child of node.children ?? []) {
    child.expanded = false;
    collapseDescendants(child);
  }
}

function stopCurrentGeneration() {
  generation++;
  offDirChanged();
  offDirChanged = () => {};
  pendingExpands.clear();
  watchGenerations.clear();
  refreshGenerations.clear();
  releaseWatches();
}

async function refreshRoot(fs: FileSystemBridge, root: string, showHidden: boolean, request: number) {
  const refreshRequest = advanceRefreshGeneration(root);
  const nodes = await loadDir(fs, root, showHidden);
  if (!isCurrent(fs, root, showHidden, request) || refreshGenerations.get(root) !== refreshRequest) return;
  const merged = mergeNodes(rootNodes.value, nodes);
  for (const gone of merged.removed) releaseSubtreeState(gone);
  rootNodes.value = merged.nodes;
}

async function refreshChanged(
  fs: FileSystemBridge,
  root: string,
  showHidden: boolean,
  request: number,
  dir: string,
) {
  if (!isCurrent(fs, root, showHidden, request)) return;
  if (dir === root) {
    await refreshRoot(fs, root, showHidden, request);
    return;
  }
  const node = findNode(rootNodes.value, dir);
  if (!node || !node.expanded) return;
  const refreshRequest = advanceRefreshGeneration(node.path);
  const children = await loadDir(fs, node.path, showHidden);
  if (
    !isCurrent(fs, root, showHidden, request)
    || !node.expanded
    || findNode(rootNodes.value, dir) !== node
    || refreshGenerations.get(node.path) !== refreshRequest
  ) return;
  const merged = mergeNodes(node.children ?? [], children);
  for (const gone of merged.removed) releaseSubtreeState(gone);
  node.children = merged.nodes;
}

// watchDirOnce 给某个目录挂 watcher(重复调用只保留一个句柄)。
async function watchDirOnce(fs: FileSystemBridge, path: string, request: number) {
  if (watchHandles.has(path)) return;
  try {
    const id = await fs.watchDir(path);
    if (disposed || generation !== request || props.fs !== fs) {
      await unwatch(fs, id);
      return;
    }
    if (watchHandles.has(path)) {
      await unwatch(fs, id);
      return;
    }
    watchHandles.set(path, { fs, id });
  } catch (err) {
    console.warn("plugin-fs: watcher unavailable or cap reached for", path, err);
  }
}

function startGeneration() {
  stopCurrentGeneration();
  const fs = props.fs;
  const root = props.root;
  const showHidden = props.showHidden;
  const request = generation;
  rootNodes.value = [];
  selectedPaths.value = new Set();
  focusedPath.value = "";
  selectionAnchor.value = "";
  dropTarget.value = "";
  offDirChanged = fs.onDirChanged((dir) => {
    void refreshChanged(fs, root, showHidden, request, dir).catch(() => {});
  });
  void refreshRoot(fs, root, showHidden, request)
    // 根目录也要 watch,否则在最外层新建/粘贴出来的文件不会自动出现。
    .then(() => watchDirOnce(fs, root, request))
    .catch(() => {});
}

watch(() => [props.root, props.fs, props.showHidden], startGeneration);

onMounted(() => {
  startGeneration();
  offClipboard = fileClipboard.subscribe((entry) => { clipboardEntry.value = entry; });
});

onBeforeUnmount(() => {
  disposed = true;
  stopCurrentGeneration();
  offClipboard();
  if (errorTimer) clearTimeout(errorTimer);
});

function findNode(nodes: TreeNode[], path: string): TreeNode | null {
  for (const n of nodes) {
    if (n.path === path) return n;
    if (n.children) {
      const sub = findNode(n.children, path);
      if (sub) return sub;
    }
  }
  return null;
}

function isCurrentNode(
  fs: FileSystemBridge,
  root: string,
  showHidden: boolean,
  request: number,
  node: TreeNode,
): boolean {
  return isCurrent(fs, root, showHidden, request) && findNode(rootNodes.value, node.path) === node;
}

async function toggle(n: TreeNode) {
  if (!n.isDir) return;
  const fs = props.fs;
  const root = props.root;
  const showHidden = props.showHidden;
  const request = generation;
  if (!isCurrentNode(fs, root, showHidden, request, n)) return;
  if (!n.expanded) {
    if (pendingExpands.has(n.path)) {
      advanceWatchGeneration(n.path);
      advanceRefreshGeneration(n.path);
      pendingExpands.delete(n.path);
      return;
    }
    const watchRequest = advanceWatchGeneration(n.path);
    pendingExpands.set(n.path, watchRequest);
    try {
      if (n.children === null) {
        const refreshRequest = advanceRefreshGeneration(n.path);
        const children = await loadDir(fs, n.path, showHidden);
        if (
          !isCurrentNode(fs, root, showHidden, request, n)
          || watchGenerations.get(n.path) !== watchRequest
          || refreshGenerations.get(n.path) !== refreshRequest
        ) return;
        n.children = children;
      }
      if (!isCurrentNode(fs, root, showHidden, request, n) || watchGenerations.get(n.path) !== watchRequest) return;
      n.expanded = true;
      const id = await fs.watchDir(n.path);
      if (
        !isCurrentNode(fs, root, showHidden, request, n)
        || !n.expanded
        || watchGenerations.get(n.path) !== watchRequest
      ) {
        await unwatch(fs, id);
        return;
      }
      const previous = watchHandles.get(n.path);
      if (previous) {
        watchHandles.delete(n.path);
        await unwatch(previous.fs, previous.id);
        if (!isCurrentNode(fs, root, showHidden, request, n) || watchGenerations.get(n.path) !== watchRequest) {
          await unwatch(fs, id);
          return;
        }
      }
      watchHandles.set(n.path, { fs, id });
    } catch (err) {
      if (isCurrentNode(fs, root, showHidden, request, n)) {
        console.warn("plugin-fs: watcher unavailable or cap reached for", n.path, err);
      }
    } finally {
      if (pendingExpands.get(n.path) === watchRequest) pendingExpands.delete(n.path);
    }
  } else {
    advanceWatchGeneration(n.path);
    pendingExpands.delete(n.path);
    releaseSubtreeState(n.path);
    collapseDescendants(n);
    n.expanded = false;
  }
  if (isCurrentNode(fs, root, showHidden, request, n)) emit("dir-toggled", n.path, n.expanded);
}

// expandDir 确保目录处于展开态(键盘 → 与拖拽悬停都用)。
async function expandDir(n: TreeNode) {
  if (n.isDir && !n.expanded) await toggle(n);
}

/* ------------------------------------------------------------ 选择与焦点 */

const visibleRows = computed(() => flattenVisible(rootNodes.value));

function nodesFor(paths: Iterable<string>): TreeNode[] {
  const out: TreeNode[] = [];
  for (const p of paths) {
    const node = findNode(rootNodes.value, p);
    if (node) out.push(node);
  }
  return out;
}

// selectionItems 返回当前选中项的 { path, isDir },没有选中就退回焦点行。
function selectionItems(): Array<{ path: string; isDir: boolean }> {
  const paths = selectedPaths.value.size ? selectedPaths.value : new Set(focusedPath.value ? [focusedPath.value] : []);
  return nodesFor(paths).map((n) => ({ path: n.path, isDir: n.isDir }));
}

function selectOnly(path: string) {
  selectedPaths.value = new Set(path ? [path] : []);
  selectionAnchor.value = path;
  focusedPath.value = path;
}

function focusTree() {
  wrapRef.value?.focus();
}

function onRowClick(ev: MouseEvent, node: TreeNode) {
  focusTree();
  if (ev.shiftKey && selectionAnchor.value) {
    selectedPaths.value = new Set(rangePaths(visibleRows.value, selectionAnchor.value, node.path));
    focusedPath.value = node.path;
    return;
  }
  if (ev.ctrlKey || ev.metaKey) {
    selectedPaths.value = toggleSelection(selectedPaths.value, node.path);
    selectionAnchor.value = node.path;
    focusedPath.value = node.path;
    return;
  }
  selectOnly(node.path);
  if (node.isDir) void toggle(node);
  else emit("file-clicked", node.path);
}

function onRowDblClick(_ev: MouseEvent, node: TreeNode) {
  if (!node.isDir) emit("file-double-clicked", node.path);
}

// 内联新建/重命名的输入框就在树里,它的按键会冒泡上来。不挡住的话,
// 在输入框里按 Delete 会弹删除确认,按 Escape 会被当成清空选择。
function isTextEntry(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el) return false;
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || el.isContentEditable;
}

function onTreeKeydown(ev: KeyboardEvent) {
  if (isTextEntry(ev.target)) return;
  const flat = visibleRows.value;
  const row = findRow(flat, focusedPath.value);
  const action = treeKeyAction(ev, row);
  if (!action) return;
  ev.preventDefault();
  switch (action.type) {
    case "move": {
      const next = moveFocus(flat, focusedPath.value, action.delta);
      if (!next) return;
      if (action.extend && selectionAnchor.value) {
        selectedPaths.value = new Set(rangePaths(flat, selectionAnchor.value, next));
        focusedPath.value = next;
      } else {
        selectOnly(next);
      }
      scrollFocusIntoView();
      return;
    }
    case "focus":
      selectOnly(action.path);
      scrollFocusIntoView();
      return;
    case "expand": {
      const node = findNode(rootNodes.value, focusedPath.value);
      if (node) void expandDir(node);
      return;
    }
    case "collapse":
    case "toggle": {
      const node = findNode(rootNodes.value, focusedPath.value);
      if (node) void toggle(node);
      return;
    }
    case "open":
      if (focusedPath.value) emit("file-clicked", focusedPath.value);
      return;
    case "rename": {
      const node = findNode(rootNodes.value, focusedPath.value);
      if (node) inlineIntent.value = { kind: "rename", node, level: row?.level ?? 0 };
      return;
    }
    case "delete":
      requestDelete(nodesFor(selectedPaths.value.size ? selectedPaths.value : [focusedPath.value]), ev.shiftKey);
      return;
    case "selectAll":
      selectedPaths.value = new Set(flat.map((r) => r.path));
      return;
    case "clearSelection":
      selectedPaths.value = new Set();
      inlineIntent.value = null;
      menu.value = null;
      return;
    case "copy":
      doCopy();
      return;
    case "cut":
      doCut();
      return;
    case "paste":
      void doPaste(targetDirFor(findNode(rootNodes.value, focusedPath.value)));
      return;
    case "duplicate":
      void doDuplicate();
      return;
  }
}

function scrollFocusIntoView() {
  requestAnimationFrame(() => {
    const el = wrapRef.value?.querySelector(`[data-path="${CSS.escape(focusedPath.value)}"]`);
    (el as HTMLElement | null)?.scrollIntoView({ block: "nearest" });
  });
}

/* ------------------------------------------------------------ 右键菜单 */

function openMenuFromNode(ev: MouseEvent, node: TreeNode, level: number) {
  ev.preventDefault();
  ev.stopPropagation();
  if (!selectedPaths.value.has(node.path)) selectOnly(node.path);
  menu.value = { x: ev.clientX, y: ev.clientY, node, level, shift: ev.shiftKey };
}

// 空白区右键 = 以项目根为目标。最外层新建文件的主入口就是这里
// (以及树头部的两个按钮)。
function openMenuFromBlank(ev: MouseEvent) {
  ev.preventDefault();
  selectedPaths.value = new Set();
  focusedPath.value = "";
  menu.value = { x: ev.clientX, y: ev.clientY, node: null, level: -1, shift: ev.shiftKey };
}

function closeMenu() { menu.value = null; }

type MenuItem = { id: string; label: string; test: string; danger?: boolean; disabled?: boolean };

const menuItems = computed<Array<MenuItem | "sep">>(() => {
  const node = menu.value?.node ?? null;
  const items: Array<MenuItem | "sep"> = [
    { id: "newFile", label: T.newFile, test: "menu-new-file" },
    { id: "newFolder", label: T.newFolder, test: "menu-new-folder" },
    "sep",
  ];
  if (node) {
    items.push(
      { id: "copy", label: T.copy, test: "menu-copy" },
      { id: "cut", label: T.cut, test: "menu-cut" },
    );
  }
  items.push({ id: "paste", label: T.paste, test: "menu-paste", disabled: !clipboardEntry.value });
  if (node) {
    items.push(
      { id: "duplicate", label: T.duplicate, test: "menu-duplicate" },
      "sep",
      { id: "rename", label: T.rename, test: "menu-rename" },
      { id: "copyRelPath", label: T.copyRelPath, test: "menu-copy-rel-path" },
      { id: "copyAbsPath", label: T.copyAbsPath, test: "menu-copy-abs-path" },
      { id: "reveal", label: T.reveal, test: "menu-reveal" },
      "sep",
      { id: "delete", label: T.delete, test: "menu-delete", danger: true },
    );
  } else {
    items.push("sep", { id: "reveal", label: T.reveal, test: "menu-reveal" });
  }
  return items;
});

async function onMenuAction(id: string) {
  const anchor = menu.value;
  menu.value = null;
  if (!anchor) return;
  const node = anchor.node;
  const dir = targetDirFor(node);

  switch (id) {
    case "newFile":
    case "newFolder":
      inlineIntent.value = {
        kind: id as "newFile" | "newFolder",
        parentPath: dir,
        parentLevel: node?.isDir ? anchor.level + 1 : Math.max(anchor.level, 0),
      };
      if (node?.isDir) await expandDir(node);
      return;
    case "rename":
      if (node) inlineIntent.value = { kind: "rename", node, level: anchor.level };
      return;
    case "copy":
      doCopy();
      return;
    case "cut":
      doCut();
      return;
    case "paste":
      await doPaste(dir);
      return;
    case "duplicate":
      await doDuplicate();
      return;
    case "copyRelPath":
      await copyText(node ? node.path : props.root);
      return;
    case "copyAbsPath": {
      const abs = await runFsOp(() => props.fs.absPath(node ? node.path : props.root));
      if (abs !== null) await copyText(abs);
      return;
    }
    case "reveal":
      await runFsOp(() => props.fs.reveal(node ? node.path : props.root));
      return;
    case "delete":
      requestDelete(nodesFor(selectedPaths.value.size ? selectedPaths.value : node ? [node.path] : []), anchor.shift);
      return;
  }
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    showError("无法写入系统剪贴板");
  }
}

/* --------------------------------------------------------- 文件操作动作 */

function doCopy() {
  const items = selectionItems();
  if (!items.length) return;
  fileClipboard.copy(props.fs.projectId, items);
  cutPaths.value = new Set();
}

function doCut() {
  const items = selectionItems();
  if (!items.length) return;
  fileClipboard.cut(props.fs.projectId, items);
  cutPaths.value = new Set(items.map((it) => it.path));
}

// applyOps 顺序执行一批拷贝/移动。后端返回实际写入的目标路径(同名会自动改名),
// 同项目内的移动要把编辑器标签一起改过去。
async function applyOps(ops: ReturnType<typeof planPaste>["ops"]): Promise<boolean> {
  let ok = true;
  for (const op of ops) {
    const written = await runFsOp(() =>
      op.mode === "cut"
        ? props.fs.moveFrom(op.srcProjectId, op.srcPath, op.dstPath)
        : props.fs.copyFrom(op.srcProjectId, op.srcPath, op.dstPath),
    );
    if (written === null) { ok = false; continue; }
    if (op.mode === "cut" && op.srcProjectId === props.fs.projectId) {
      emit("path-renamed", op.srcPath, written);
    }
  }
  return ok;
}

async function doPaste(targetDir: string) {
  const entry = fileClipboard.read();
  if (!entry) return;
  const { ops, error } = planPaste(entry, props.fs.projectId, targetDir);
  if (error) { showError(error); return; }
  if (!ops.length) return;
  await applyOps(ops);
  if (entry.mode === "cut") {
    fileClipboard.clear();
    cutPaths.value = new Set();
  }
}

// doDuplicate 是「复制到原目录」,后端 UniqueName 负责生成「x copy」。
async function doDuplicate() {
  const items = selectionItems();
  if (!items.length) return;
  await applyOps(items.map((it) => ({
    mode: "copy" as const,
    srcProjectId: props.fs.projectId,
    srcPath: it.path,
    dstProjectId: props.fs.projectId,
    dstPath: it.path,
  })));
}

/* ------------------------------------------------------------ 内联新建 */

async function submitInline(name: string) {
  const intent = inlineIntent.value;
  inlineIntent.value = null;
  if (!intent) return;
  if (intent.kind === "newFile") {
    const path = joinPath(intent.parentPath, name);
    const created = await runFsOp(() => props.fs.createFile(path));
    if (created !== null) emit("file-clicked", path); // 新建后直接打开
    return;
  }
  if (intent.kind === "newFolder") {
    await runFsOp(() => props.fs.mkdir(joinPath(intent.parentPath, name)));
    return;
  }
  const from = intent.node.path;
  const to = joinPath(parentDir(from), name);
  if (from === to) return;
  const renamed = await runFsOp(() => props.fs.rename(from, to));
  if (renamed !== null) {
    emit("path-renamed", from, to);
    selectOnly(to);
  }
}

function cancelInline() { inlineIntent.value = null; }

/* ---------------------------------------------------------------- 删除 */

function requestDelete(nodes: TreeNode[], hard: boolean) {
  if (!nodes.length) return;
  deleteConfirm.value = { nodes, mode: hard ? "hard" : "trash" };
}

async function resolveDeleteConfirm(id: string) {
  const conf = deleteConfirm.value;
  deleteConfirm.value = null;
  if (!conf || id === "cancel") return;
  const failedToTrash: TreeNode[] = [];
  for (const node of conf.nodes) {
    if (id === "hard") {
      const done = await runFsOp(() => props.fs.remove(node.path, node.isDir));
      if (done !== null) emit("path-removed", node.path);
      continue;
    }
    try {
      await props.fs.trash(node.path);
      emit("path-removed", node.path);
    } catch (err) {
      const msg = (err as Error).message ?? "";
      if (msg.includes("no platform trash command available") || msg.includes("trash unavailable")) {
        failedToTrash.push(node);
      } else {
        showError(msg || "删除失败");
      }
    }
  }
  // 平台没有废纸篓时,用硬删确认再问一次。
  if (failedToTrash.length) deleteConfirm.value = { nodes: failedToTrash, mode: "hard" };
}

const deleteTitle = computed(() => {
  const conf = deleteConfirm.value;
  if (!conf) return "";
  if (conf.nodes.length === 1) {
    const name = baseName(conf.nodes[0].path);
    return conf.mode === "hard" ? T.confirmHardDelete(name) : T.confirmTrash(name);
  }
  return conf.mode === "hard"
    ? T.confirmHardDeleteMany(conf.nodes.length)
    : T.confirmTrashMany(conf.nodes.length);
});

function deleteButtons(mode: "trash" | "hard") {
  const primary = mode === "hard"
    ? { id: "hard", label: T.delete, kind: "danger" as const }
    : { id: "trash", label: T.moveToTrash, kind: "primary" as const };
  return [primary, { id: "cancel", label: T.cancel, kind: "secondary" as const }];
}

/* ---------------------------------------------------------------- 拖拽 */

function onDragStart(ev: DragEvent, node: TreeNode) {
  if (!selectedPaths.value.has(node.path)) selectOnly(node.path);
  dragItems = selectionItems();
  ev.dataTransfer?.setData("text/plain", node.path);
  if (ev.dataTransfer) ev.dataTransfer.effectAllowed = "move";
}

// 落点始终是一个目录:目录行落到自身,文件行落到它的父目录。
// 父目录是根时不高亮具体某一行,改用根的虚线框。
function onDragOverNode(ev: DragEvent, node: TreeNode) {
  if (!dragItems.length) return;
  ev.preventDefault();
  ev.stopPropagation();
  if (ev.dataTransfer) ev.dataTransfer.dropEffect = "move";
  const dir = targetDirFor(node);
  dropTarget.value = dir;
  rootDropActive.value = dir === props.root;
}

function onDragLeaveNode(_ev: DragEvent, node: TreeNode) {
  if (dropTarget.value === targetDirFor(node)) dropTarget.value = "";
}

function onDragOverRoot(ev: DragEvent) {
  if (!dragItems.length) return;
  ev.preventDefault();
  if (ev.dataTransfer) ev.dataTransfer.dropEffect = "move";
  dropTarget.value = "";
  rootDropActive.value = true;
}

function onDragLeaveRoot() { rootDropActive.value = false; }

async function onDropOnNode(ev: DragEvent, node: TreeNode) {
  ev.preventDefault();
  ev.stopPropagation();
  await performDrop(targetDirFor(node));
}

async function onDropOnRoot(ev: DragEvent) {
  ev.preventDefault();
  await performDrop(props.root);
}

async function performDrop(targetDir: string) {
  const items = dragItems;
  dragItems = [];
  dropTarget.value = "";
  rootDropActive.value = false;
  if (!items.length) return;
  const { ops, error } = planDrop(items, props.fs.projectId, targetDir);
  if (error) { showError(error); return; }
  if (!ops.length) return;
  await applyOps(ops);
}

function onDragEnd() {
  dragItems = [];
  dropTarget.value = "";
  rootDropActive.value = false;
}

/* ---------------------------------------------------------------- 对外 */

// 树头部的「新建」按钮走这里:目标固定是项目根,不依赖右键选中了什么。
function startNewAtRoot(kind: "newFile" | "newFolder") {
  inlineIntent.value = { kind, parentPath: props.root, parentLevel: 0 };
}

const rootInlineIntent = computed(() => {
  const intent = inlineIntent.value;
  if (!intent) return null;
  if (intent.kind !== "newFile" && intent.kind !== "newFolder") return null;
  return intent.parentPath === props.root ? intent : null;
});

defineExpose({
  refresh: startGeneration,
  newFile: () => startNewAtRoot("newFile"),
  newFolder: () => startNewAtRoot("newFolder"),
  paste: () => doPaste(props.root),
  hasClipboard: () => !!clipboardEntry.value,
});
</script>

<template>
  <div
    ref="wrapRef"
    class="tree-wrap"
    tabindex="0"
    data-test="file-tree"
    @click="closeMenu"
    @keydown="onTreeKeydown"
    @contextmenu.self="openMenuFromBlank"
    @dragover="onDragOverRoot"
    @dragleave="onDragLeaveRoot"
    @drop="onDropOnRoot"
    @dragend="onDragEnd"
  >
    <ul class="tree-root" :class="{ 'root-drop': rootDropActive }">
      <!-- 根级内联新建行。少了它,「在最外层新建文件」点了没有任何反应。 -->
      <li v-if="rootInlineIntent">
        <InlineEditRow
          :level="0"
          :icon="rootInlineIntent.kind === 'newFolder' ? 'folder' : 'file'"
          @submit="submitInline"
          @cancel="cancelInline"
        />
      </li>
      <li v-for="n in rootNodes" :key="n.path">
        <FileTreeNode
          :node="n"
          :level="0"
          :selected-paths="selectedPaths"
          :focused-path="focusedPath"
          :cut-paths="cutPaths"
          :drop-target="dropTarget"
          :inline-intent="inlineIntent"
          @row-click="onRowClick"
          @row-dblclick="onRowDblClick"
          @context="openMenuFromNode"
          @drag-start="onDragStart"
          @drag-over="onDragOverNode"
          @drag-leave="onDragLeaveNode"
          @drop-on="onDropOnNode"
          @inline-submit="submitInline"
          @inline-cancel="cancelInline"
        />
      </li>
    </ul>
    <!-- 空白填充区:让「树下方的空处」也能右键/放置到根。 -->
    <div class="tree-blank" @contextmenu="openMenuFromBlank" />

    <div v-if="treeError" class="tree-error" data-test="file-tree-error">{{ treeError }}</div>

    <div
      v-if="menu"
      ref="menuRef"
      class="ctx-menu"
      data-test="file-tree-menu"
      :style="menuStyle"
      @click.stop
      @contextmenu.prevent.stop
    >
      <template v-for="(item, i) in menuItems">
        <div v-if="item === 'sep'" :key="`sep-${i}`" class="ctx-sep" />
        <button
          v-else
          :key="item.id"
          :data-test="item.test"
          :class="{ danger: item.danger }"
          :disabled="item.disabled"
          @click="onMenuAction(item.id)"
        >
          {{ item.label }}
        </button>
      </template>
    </div>

    <ConfirmDialog
      v-if="deleteConfirm"
      :title="deleteTitle"
      :buttons="deleteButtons(deleteConfirm.mode)"
      @resolve="resolveDeleteConfirm"
    />
  </div>
</template>

<style scoped>
.tree-wrap { position: relative; height: 100%; display: flex; flex-direction: column; outline: none; }
.tree-root {
  list-style: none;
  margin: 0;
  padding: 0;
}
.tree-root.root-drop { outline: 1px dashed var(--ed-tab-active-bar, #539bf5); outline-offset: -1px; }
.tree-root > li { display: block; }
.tree-blank { flex: 1 1 auto; min-height: 24px; }
.tree-error {
  position: sticky;
  bottom: 0;
  padding: 4px 10px;
  font-size: 11px;
  color: var(--ed-error, #f47067);
  background: var(--ed-shell-bg, #22272e);
  border-top: 1px solid var(--ed-border, #444c56);
}
.ctx-menu {
  position: fixed;
  z-index: 60;
  background: var(--ed-shell-bg, #22272e);
  border: 1px solid var(--ed-border, #444c56);
  border-radius: 4px;
  padding: 4px 0;
  min-width: 160px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
  display: flex;
  flex-direction: column;
  /* 菜单比视口还高时(小窗口 + 完整节点菜单)内部滚动,而不是伸到视口外。 */
  overflow-y: auto;
}
.ctx-menu button { flex: 0 0 auto; }
.ctx-menu button {
  background: none;
  border: none;
  color: var(--ed-row-fg, #adbac7);
  font: inherit;
  font-size: 12px;
  padding: 6px 14px;
  text-align: left;
  cursor: pointer;
}
.ctx-menu button:hover:not(:disabled) { background: var(--ed-row-hover, rgba(255,255,255,0.06)); }
.ctx-menu button:disabled { opacity: 0.4; cursor: default; }
.ctx-menu button.danger { color: var(--ed-error, #f47067); }
.ctx-sep { height: 1px; margin: 4px 0; background: var(--ed-border, #444c56); }
</style>
