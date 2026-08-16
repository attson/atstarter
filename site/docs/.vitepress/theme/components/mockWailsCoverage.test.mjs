// 演示站把 `wailsjs/go/main/App` 和 `wailsjs/runtime/runtime` 两个路径 alias 到
// 本目录的 mock 上(见 .vitepress/config.mjs)。只要 frontend 新导入一个绑定而 mock
// 没跟上,rollup 就会以 "X is not exported by ..." 失败 —— 但站点构建只在 push 到
// main 之后才跑,合并前没有任何信号。
//
// 这个测试把「frontend 导入的名字 ⊆ mock 导出的名字」这个不变式提前到单元测试,
// 纯 node、无需装依赖,可以挂在 PR CI 上。
import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const frontendSrc = path.resolve(here, '../../../../../frontend/src')

function sourceFiles(dir, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) sourceFiles(full, out)
    // .ts 也要扫:文件浏览器的 fsBridge.ts 是绑定导入最集中的地方,
    // 漏掉它等于这个不变式对整个文件子系统失效。
    else if (/\.(vue|js|mjs|ts)$/.test(entry.name)) out.push(full)
  }
  return out
}

// 收集 frontend/src 下所有从 modulePattern 匹配路径具名导入的标识符。
// 返回 Map<导入名, 出现该导入的文件相对路径集合>,断言失败时能直接指到文件。
function namedImportsFrom(modulePattern) {
  const found = new Map()
  for (const file of sourceFiles(frontendSrc)) {
    const src = fs.readFileSync(file, 'utf8')
    const re = new RegExp(
      String.raw`import\s*\{([^}]*)\}\s*from\s*['"]([^'"]*${modulePattern})['"]`,
      'g',
    )
    for (const match of src.matchAll(re)) {
      for (const raw of match[1].split(',')) {
        // `Foo as Bar` 导入的是 Foo,mock 必须导出 Foo。
        const name = raw.trim().split(/\s+as\s+/)[0].trim()
        if (!name) continue
        if (!found.has(name)) found.set(name, new Set())
        found.get(name).add(path.relative(frontendSrc, file))
      }
    }
  }
  return found
}

function assertMockCovers(imported, mockModule, mockName) {
  assert.ok(imported.size > 0, `没有从 ${mockName} 对应路径解析到任何导入,正则可能失效了`)
  const missing = [...imported.keys()]
    .filter((name) => typeof mockModule[name] === 'undefined')
    .sort()
  assert.deepEqual(
    missing,
    [],
    `${mockName} 缺少这些导出(演示站构建会失败):\n` +
      missing.map((n) => `  - ${n}  ← ${[...imported.get(n)].join(', ')}`).join('\n'),
  )
}

test('mockWailsApp 覆盖 frontend 用到的全部 App 绑定', async () => {
  const mock = await import('./mockWailsApp.mjs')
  assertMockCovers(namedImportsFrom(String.raw`wailsjs/go/main/App`), mock, 'mockWailsApp.mjs')
})

test('mockWailsRuntime 覆盖 frontend 用到的全部 runtime 绑定', async () => {
  const mock = await import('./mockWailsRuntime.mjs')
  assertMockCovers(
    namedImportsFrom(String.raw`wailsjs/runtime/runtime`),
    mock,
    'mockWailsRuntime.mjs',
  )
})
