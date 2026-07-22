// dockerState.js — 容器快照的分组与筛选纯函数。

// groupContainers 把容器按 compose 归属分组。
export function groupContainers(list) {
  const compose = {}
  const standalone = []
  for (const c of list || []) {
    if (c.compose) {
      ;(compose[c.compose] ||= []).push(c)
    } else {
      standalone.push(c)
    }
  }
  return { compose, standalone }
}

// filterContainers 按名字子串(大小写不敏感)筛选;空关键字返回全部。
export function filterContainers(list, keyword) {
  const kw = (keyword || '').trim().toLowerCase()
  if (!kw) return list || []
  return (list || []).filter((c) => (c.name || '').toLowerCase().includes(kw))
}
