// menuPlacement:把「鼠标点击坐标 + 菜单实测尺寸」算成不会被视口切掉的落位。
//
// 右键菜单是 position: fixed,直接用 clientX/clientY 的话,在靠近窗口底部
// 右键就会有一截伸到视口外面点不到。这里的规则:
//   1. 优先放在点击点的右下(常规方向)。
//   2. 下方装不下就向上翻,菜单底边对齐点击点。
//   3. 上下都装不下(菜单比视口还高)就贴边,并给一个 maxHeight 让它内部滚动。
// 水平方向同理。

const DEFAULT_MARGIN = 6

function clamp(value, min, max) {
  if (max < min) return min
  return Math.min(Math.max(value, min), max)
}

// place1d 是单个轴向的落位计算,水平垂直共用。
function place1d(anchor, size, viewport, margin) {
  const available = Math.max(viewport - margin * 2, 0)
  const length = Math.min(size, available)
  // 正方向放得下就正放。
  if (anchor + length + margin <= viewport) return { start: Math.max(anchor, margin), length }
  // 反方向放得下就翻过去。
  if (anchor - length >= margin) return { start: anchor - length, length }
  // 两边都放不下:贴边。
  return { start: clamp(viewport - length - margin, margin, Math.max(viewport - length - margin, margin)), length }
}

export function placeMenu(anchor, size, viewport, margin = DEFAULT_MARGIN) {
  const x = place1d(anchor.x, size.width, viewport.width, margin)
  const y = place1d(anchor.y, size.height, viewport.height, margin)
  return {
    left: Math.round(x.start),
    top: Math.round(y.start),
    // maxHeight 只在菜单真的比可用高度还高时才有约束意义,但恒定返回也无害:
    // 值永远 >= 实际高度,CSS 上不会产生多余滚动条。
    maxHeight: Math.round(Math.max(viewport.height - margin * 2, 0)),
  }
}
