import assert from 'node:assert/strict'
import test from 'node:test'
import { placeMenu } from './menuPlacement.js'

const viewport = { width: 800, height: 600 }
const size = { width: 160, height: 300 }

test('空间够时就放在点击点的右下', () => {
  const p = placeMenu({ x: 100, y: 100 }, size, viewport)
  assert.equal(p.left, 100)
  assert.equal(p.top, 100)
})

test('下方装不下就向上翻,菜单底边对齐点击点', () => {
  // 这就是截图里的情况:在靠近底部的行右键,菜单被切掉一半。
  const p = placeMenu({ x: 100, y: 500 }, size, viewport)
  assert.equal(p.top, 200, '500 - 300 = 200,整个菜单都在视口内')
  assert.ok(p.top + size.height <= viewport.height)
})

test('右侧装不下就向左翻', () => {
  const p = placeMenu({ x: 700, y: 100 }, size, viewport)
  assert.equal(p.left, 540)
  assert.ok(p.left + size.width <= viewport.width)
})

test('上下都装不下(菜单比视口高)就贴边并限高', () => {
  const tall = { width: 160, height: 900 }
  const p = placeMenu({ x: 100, y: 400 }, tall, viewport)
  assert.equal(p.top, 6, '贴上边距')
  assert.equal(p.maxHeight, 588, '600 - 6*2,菜单内部滚动')
})

test('点击点本身贴着边时不会算出负坐标', () => {
  const p = placeMenu({ x: 0, y: 0 }, size, viewport)
  assert.ok(p.left >= 0)
  assert.ok(p.top >= 0)

  const corner = placeMenu({ x: 800, y: 600 }, size, viewport)
  assert.ok(corner.left >= 0 && corner.left + size.width <= viewport.width)
  assert.ok(corner.top >= 0 && corner.top + size.height <= viewport.height)
})

test('视口比菜单还小也不会返回负值', () => {
  const tiny = { width: 80, height: 80 }
  const p = placeMenu({ x: 40, y: 40 }, size, tiny)
  assert.ok(p.left >= 0)
  assert.ok(p.top >= 0)
  assert.ok(p.maxHeight >= 0)
})

test('自定义边距被尊重', () => {
  const p = placeMenu({ x: 100, y: 590 }, size, viewport, 20)
  assert.equal(p.top, 290)
  assert.equal(p.maxHeight, 560)
})
