// package.json script 补全的纯逻辑:解析输入、过滤候选、回写整行。
// 不碰 DOM、不碰 Vue,便于用 node:test 覆盖。

// 触发补全的包管理器。只认 `<pm> run <name>` 这一种写法:
// `pnpm dev` 这类省略 run 的简写要跟 install/add/build 等 pm 自带子命令区分,
// 规则复杂且容易误补,暂不支持。
const PACKAGE_MANAGERS = ['npm', 'pnpm', 'yarn', 'bun']

// 形如 `<前导空白><pm><空白>run<空白><脚本名前缀>`,且脚本名后不能再有内容。
const RUN_LINE = new RegExp(
  `^(\\s*(?:${PACKAGE_MANAGERS.join('|')})\\s+run\\s+)([^\\s]*)$`,
  'i',
)

/**
 * 解析输入框当前值,判断光标处是否处于「脚本名补全」上下文。
 *
 * @param {string} line 命令整行
 * @returns {{head: string, prefix: string} | null}
 *   head 是脚本名之前的全部原文(含原始空白),prefix 是已输入的脚本名前缀;
 *   不处于补全上下文时返回 null。
 */
export function scriptCompleteContext(line) {
  if (typeof line !== 'string') return null
  const m = RUN_LINE.exec(line)
  if (!m) return null
  return { head: m[1], prefix: m[2] }
}

/**
 * 按前缀过滤候选脚本。前缀命中排在子串命中之前,组内保持传入顺序
 * (后端已按脚本名字典序返回)。大小写不敏感。
 *
 * @param {Array<{name: string, script: string}>} scripts
 * @param {string} prefix
 */
export function filterScripts(scripts, prefix) {
  if (!Array.isArray(scripts) || scripts.length === 0) return []
  const needle = String(prefix || '').toLowerCase()
  if (!needle) return scripts
  const starts = []
  const contains = []
  for (const s of scripts) {
    const name = String(s.name || '').toLowerCase()
    if (name.startsWith(needle)) starts.push(s)
    else if (name.includes(needle)) contains.push(s)
  }
  return [...starts, ...contains]
}

/**
 * 把选中的脚本名写回整行,只替换尾部脚本名,保留原始空白。
 * 非补全上下文原样返回。
 *
 * @param {string} line
 * @param {string} name
 */
export function applyScript(line, name) {
  const ctx = scriptCompleteContext(line)
  if (!ctx) return line
  return ctx.head + name
}
