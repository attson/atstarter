// 这些是源码级断言(和 fileBrowserSearch.test.mjs 同一路数):.vue 单文件组件
// 没法在纯 node 里挂载,但这里守的都是「曾经真的坏过」的接线点。
import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const read = (name) => fs.readFileSync(new URL(name, import.meta.url), 'utf8')

const fileTree = read('./FileTree.vue')
const fileTreeNode = read('./FileTreeNode.vue')
const fileBrowser = read('../FileBrowser.vue')
const codeEditor = read('./CodeEditor.vue')
const fileEditor = read('./FileEditor.vue')
const fsBridge = read('./fsBridge.ts')

test('最外层能新建:根级内联编辑行确实被渲染出来', () => {
  // 以前 newFile 的 parentPath 是 ""(根),但只有 FileTreeNode 里有 InlineEditRow,
  // 根没有对应节点 → 输入框永远不出现,点了没反应。
  assert.match(fileTree, /rootInlineIntent/)
  assert.match(fileTree, /intent\.parentPath === props\.root/)
  assert.match(fileTree, /v-if="rootInlineIntent"[\s\S]{0,200}<InlineEditRow/)
})

test('空白区右键与树头部按钮都能落到项目根', () => {
  assert.match(fileTree, /openMenuFromBlank/)
  assert.match(fileTree, /@contextmenu\.self="openMenuFromBlank"/)
  assert.match(fileTree, /startNewAtRoot/)
  assert.match(fileBrowser, /data-test="tree-new-file"/)
  assert.match(fileBrowser, /data-test="tree-new-folder"/)
})

test('根目录本身也被 watch,新建的文件才会自动出现', () => {
  assert.match(fileTree, /watchDirOnce\(fs, root, request\)/)
})

test('右键菜单覆盖复制/剪切/粘贴/副本/复制路径/在文件管理器中显示', () => {
  for (const id of [
    'menu-copy',
    'menu-cut',
    'menu-paste',
    'menu-duplicate',
    'menu-copy-rel-path',
    'menu-copy-abs-path',
    'menu-reveal',
  ]) {
    assert.match(fileTree, new RegExp(`"${id}"`), `右键菜单缺少 ${id}`)
  }
})

test('文件操作失败会让用户看见,而不是只 console.warn', () => {
  assert.match(fileTree, /function runFsOp/)
  assert.match(fileTree, /data-test="file-tree-error"/)
  assert.doesNotMatch(fileTree, /console\.warn\("file-explorer: inline action failed"/)
  assert.doesNotMatch(fileTree, /console\.warn\("file-explorer: delete failed"/)
})

test('fsBridge 暴露跨项目拷贝/移动并自报项目 ID', () => {
  assert.match(fsBridge, /CopyProjectPath/)
  assert.match(fsBridge, /MoveProjectPath/)
  assert.match(fsBridge, /RevealProjectPath/)
  assert.match(fsBridge, /ProjectAbsPath/)
  assert.match(fsBridge, /readonly projectId: string/)
  assert.match(fsBridge, /copyFrom\(srcProjectId: string/)
})

test('树支持多选、键盘导航与拖拽移动', () => {
  assert.match(fileTree, /selectedPaths/)
  assert.match(fileTree, /treeKeyAction/)
  assert.match(fileTree, /tabindex="0"/)
  assert.match(fileTree, /planDrop/)
  assert.match(fileTreeNode, /draggable="true"/)
  assert.match(fileTreeNode, /@dragstart/)
  assert.match(fileTreeNode, /@drop=/)
})

test('编辑器有多标签页,关标签才拦截未保存改动', () => {
  assert.match(fileBrowser, /EditorTabs/)
  assert.match(fileBrowser, /function requestClose/)
  assert.match(fileBrowser, /closeGuard/)
  assert.match(fileBrowser, /保存并关闭/)
  // 每个标签一个常驻实例:切标签不能重载文档。
  assert.match(fileBrowser, /v-for="tab in tabs\.tabs"[\s\S]{0,120}v-show="tab\.path === activePath"/)
})

test('改偏好/切主题不重建编辑器,否则未保存的改动会被丢掉', () => {
  assert.match(codeEditor, /Compartment/)
  for (const comp of ['numbersComp', 'wrapComp', 'tabComp', 'themeComp']) {
    assert.match(codeEditor, new RegExp(`${comp}\\.reconfigure`), `${comp} 没有走热切换`)
  }
  // 重新 load() 只允许由换文件/换项目触发。
  assert.match(codeEditor, /watch\(\(\) => \[props\.path, props\.fs\], \(\) => \{ void load\(\); \}\)/)
})

test('Markdown/SVG 的渲染视图可达,隐藏文件开关接到偏好上', () => {
  assert.doesNotMatch(fileBrowser, /view-mode="code"/, 'view-mode 不能再写死')
  assert.match(fileBrowser, /:view-mode="tab\.viewMode"/)
  assert.match(fileBrowser, /toggleViewMode/)
  assert.doesNotMatch(fileBrowser, /:show-hidden="true"/, 'showHidden 不能再写死')
  assert.match(fileBrowser, /:show-hidden="prefs\.showHidden"/)
})

test('工具栏与状态栏接上保存/还原/跳转行', () => {
  assert.match(fileBrowser, /EditorToolbar/)
  assert.match(fileBrowser, /EditorStatusBar/)
  assert.match(fileBrowser, /@goto="gotoLine"/)
  assert.match(codeEditor, /function revert\(\)/)
  assert.match(codeEditor, /function gotoLine\(/)
  assert.match(fileEditor, /revert: \(\) => codeEditorRef/)
})

test('重命名与删除会带着编辑器标签一起走', () => {
  assert.match(fileTree, /emit\("path-renamed"/)
  assert.match(fileTree, /emit\("path-removed"/)
  assert.match(fileBrowser, /@path-renamed="onPathRenamed"/)
  assert.match(fileBrowser, /@path-removed="onPathRemoved"/)
})
