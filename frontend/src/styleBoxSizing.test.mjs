// style.css 没有全局 border-box 重置,所以 `width: 100%` 和左右内边距/边框写在同一个
// 规则块里时,元素会比容器宽出内边距 + 边框那么多。这已经出过两次:CommandRow 的
// 输入框(溢出 46px),以及 ProjectTreeNode 的 .tree-row(溢出 24px,让侧边栏出横向
// 滚动条,并使长项目名的省略号截断失效 —— 内容有地方溢出就不会被截断)。
//
// 检查是分轴的:width:100% 只看左右内边距/边框,height:100% 只看上下。
// 并且跳过 UA 样式本来就是 border-box 的元素(实测:button、select 是 border-box;
// div、input、textarea、span、a 是 content-box)。
//
// 已知盲区,通过不代表没有溢出:
//   - 认不出跨规则块的内边距。.tree-row 声明 width:100%、.project-row 声明 padding,
//     分属两个块,这里抓不到 —— 侧边栏那次就是这种形态。
//   - 只按同文件模板里的标签判断,动态组件或透传 class 认不出来。
// 布局仍需按 FRONTEND_STYLE 的清单实际渲染核对。
import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const srcDir = path.dirname(fileURLToPath(import.meta.url))

// UA 样式表里默认就是 border-box 的标签,这些元素上的 width:100% + 内边距无害。
const BORDER_BOX_TAGS = ['button', 'select']

function vueFiles(dir, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) vueFiles(full, out)
    else if (entry.name.endsWith('.vue')) out.push(full)
  }
  return out
}

const decl = (body, prop) => {
  const m = new RegExp(String.raw`(?:^|[;{\s])${prop}\s*:\s*([^;}]+)`).exec(body)
  return m ? m[1].trim() : null
}

// padding/border 简写按 CSS 的 1~4 值规则取指定轴上是否非零。
function shorthandAxis(value, axis) {
  const parts = value.split(/\s+/)
  const pick = axis === 'x' ? [parts[1] ?? parts[0], parts[3] ?? parts[1] ?? parts[0]]
    : [parts[0], parts[2] ?? parts[0]]
  return pick.some((v) => v && !/^(0(px|em|rem|%)?|none)$/.test(v))
}

function hasAxisPaddingOrBorder(body, axis) {
  const sides = axis === 'x' ? ['left', 'right'] : ['top', 'bottom']
  for (const prop of ['padding', 'border']) {
    const short = decl(body, prop)
    // border 简写形如 `1px solid var(--border)`,四边同值,直接看首段宽度。
    if (short !== null) {
      if (prop === 'border') {
        if (!/^(0(px|em|rem)?|none)\b/.test(short)) return true
      } else if (shorthandAxis(short, axis)) return true
    }
    for (const side of sides) {
      const one = decl(body, `${prop}-${side}`) ?? (prop === 'border' ? decl(body, `border-${side}-width`) : null)
      if (one !== null && !/^(0(px|em|rem|%)?|none)\b/.test(one)) return true
    }
  }
  return false
}

// 选择器最后一段的类名,用于回模板里找它挂在什么标签上。
function lastClass(selector) {
  const m = [...selector.matchAll(/\.([A-Za-z0-9_-]+)/g)]
  return m.length ? m[m.length - 1][1] : null
}

function resolvesToBorderBoxTag(selector, template) {
  const bare = selector.trim().split(/\s+/).pop()
  if (BORDER_BOX_TAGS.includes(bare)) return true
  const cls = lastClass(selector)
  if (!cls) return false
  for (const tag of BORDER_BOX_TAGS) {
    // 匹配该标签的开始标签整体(含换行),看 class / :class 里有没有这个类名。
    for (const open of template.matchAll(new RegExp(String.raw`<${tag}\b[^>]*>`, 'gs'))) {
      if (new RegExp(String.raw`\b${cls}\b`).test(open[0])) return true
    }
  }
  return false
}

export function violations(source) {
  const styleStart = source.indexOf('<style')
  if (styleStart === -1) return []
  const template = source.slice(0, styleStart)
  // 从 <style ...> 的结束尖括号之后开始,免得把标签本身当成选择器。
  const style = source.slice(source.indexOf('>', styleStart) + 1)
  const found = []
  for (const match of style.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const selector = match[1].trim().replace(/\s+/g, ' ')
    const body = match[2]
    if (/(?:^|[;{\s])box-sizing\s*:/.test(body)) continue
    for (const [prop, axis] of [['width', 'x'], ['height', 'y']]) {
      if (decl(body, prop) !== '100%') continue
      if (!hasAxisPaddingOrBorder(body, axis)) continue
      if (resolvesToBorderBoxTag(selector, template)) continue
      found.push(`${selector} (${prop}: 100%)`)
    }
  }
  return found
}

test('width/height 100% 与同轴内边距同块时必须显式声明 box-sizing', () => {
  const offenders = []
  for (const file of vueFiles(srcDir)) {
    for (const selector of violations(fs.readFileSync(file, 'utf8'))) {
      offenders.push(`${path.relative(srcDir, file)}  ->  ${selector}`)
    }
  }
  assert.deepEqual(
    offenders,
    [],
    '这些规则块把 100% 尺寸和同轴内边距/边框写在一起却没声明 box-sizing,\n' +
      '在没有全局重置的情况下会撑出容器:\n  ' + offenders.join('\n  '),
  )
})

test('检查逻辑本身的行为', () => {
  const wrap = (tpl, css) => `<template>${tpl}</template>\n<style scoped>\n${css}\n</style>`

  // CommandRow 当初的写法:div 容器里的 input,应当报出来
  assert.deepEqual(
    violations(wrap('<div><input></div>', 'input { width: 100%; padding: 0 10px; }')),
    ['input (width: 100%)'],
  )
  // 声明了 box-sizing 就放行
  assert.deepEqual(
    violations(wrap('<div><input></div>', 'input { box-sizing: border-box; width: 100%; padding: 0 10px; }')),
    [],
  )
  // button 的 UA 默认就是 border-box,不报
  assert.deepEqual(
    violations(wrap('<button class="row">x</button>', '.row { width: 100%; padding: 6px 10px; }')),
    [],
  )
  // 分轴:width 不受上下内边距影响
  assert.deepEqual(
    violations(wrap('<div class="a"></div>', '.a { width: 100%; padding: 6px 0; }')),
    [],
  )
  // 分轴:height 只看上下
  assert.deepEqual(
    violations(wrap('<div class="a"></div>', '.a { height: 100%; padding: 6px 0; }')),
    ['.a (height: 100%)'],
  )
  // border: 0 / none 不算
  assert.deepEqual(
    violations(wrap('<div class="a"></div><div class="b"></div>',
      '.a { width: 100%; border: 0; }\n.b { height: 100%; border: none; }')),
    [],
  )
  // 只有 border-bottom 时不影响宽度
  assert.deepEqual(
    violations(wrap('<div class="a"></div>', '.a { width: 100%; border-bottom: 1px solid red; }')),
    [],
  )
  // 已知盲区:内边距在另一个规则块里时抓不到(ProjectTreeNode 那次的形态)
  assert.deepEqual(
    violations(wrap('<div class="row"><div class="row-inner"></div></div>',
      '.row { width: 100%; }\n.row-inner { padding: 2px 8px; }')),
    [],
  )
})
