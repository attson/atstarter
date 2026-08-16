import assert from 'node:assert/strict'
import test from 'node:test'
import {
  baseName,
  isDescendant,
  isIntoSelf,
  isSameDirMove,
  joinPath,
  parentDir,
  rewritePrefix,
  targetDirFor,
} from './pathOps.js'

test('joinPath 在根下不加前导斜杠', () => {
  assert.equal(joinPath('', 'a.txt'), 'a.txt')
  assert.equal(joinPath('sub', 'a.txt'), 'sub/a.txt')
  assert.equal(joinPath('sub/', 'a.txt'), 'sub/a.txt')
})

test('parentDir 顶层项的父目录是根', () => {
  assert.equal(parentDir('a.txt'), '')
  assert.equal(parentDir('sub/a.txt'), 'sub')
  assert.equal(parentDir('a/b/c.txt'), 'a/b')
})

test('baseName 取最后一段', () => {
  assert.equal(baseName('a.txt'), 'a.txt')
  assert.equal(baseName('sub/deep/a.txt'), 'a.txt')
})

test('isDescendant 不把自己算作自己的子孙', () => {
  assert.equal(isDescendant('sub/a.txt', 'sub'), true)
  assert.equal(isDescendant('sub', 'sub'), false)
  assert.equal(isDescendant('subway/a.txt', 'sub'), false, '前缀相同但不是同一层目录')
  assert.equal(isDescendant('a.txt', ''), true, '根是一切的祖先')
  assert.equal(isDescendant('', ''), false)
})

test('targetDirFor 目录落自身、文件落父目录、空节点落根', () => {
  assert.equal(targetDirFor({ path: 'sub', isDir: true }), 'sub')
  assert.equal(targetDirFor({ path: 'sub/a.txt', isDir: false }), 'sub')
  assert.equal(targetDirFor({ path: 'a.txt', isDir: false }), '')
  assert.equal(targetDirFor(null), '')
})

test('isSameDirMove 认出原地移动', () => {
  assert.equal(isSameDirMove('sub/a.txt', 'sub'), true)
  assert.equal(isSameDirMove('a.txt', ''), true)
  assert.equal(isSameDirMove('sub/a.txt', ''), false)
})

test('isIntoSelf 拦住目录落进自己或子孙', () => {
  assert.equal(isIntoSelf('sub', 'sub'), true)
  assert.equal(isIntoSelf('sub', 'sub/deep'), true)
  assert.equal(isIntoSelf('sub', 'other'), false)
  assert.equal(isIntoSelf('sub', ''), false)
})

test('rewritePrefix 跟随重命名改写自身与子孙', () => {
  assert.equal(rewritePrefix('old', 'old', 'new'), 'new')
  assert.equal(rewritePrefix('old/a.txt', 'old', 'new'), 'new/a.txt')
  assert.equal(rewritePrefix('oldish/a.txt', 'old', 'new'), 'oldish/a.txt')
  assert.equal(rewritePrefix('other', 'old', 'new'), 'other')
})
