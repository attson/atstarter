const listeners = new Map()

export function EventsOnMultiple(eventName, callback, maxCallbacks = -1) {
  const list = listeners.get(eventName) || []
  list.push({ callback, maxCallbacks, calls: 0 })
  listeners.set(eventName, list)
  return () => EventsOff(eventName)
}

export function EventsOn(eventName, callback) {
  return EventsOnMultiple(eventName, callback, -1)
}

export function EventsOnce(eventName, callback) {
  return EventsOnMultiple(eventName, callback, 1)
}

export function EventsOff(eventName, ...additionalEventNames) {
  for (const name of [eventName, ...additionalEventNames]) listeners.delete(name)
}

export function EventsOffAll() {
  listeners.clear()
}

export function EventsEmit(eventName, ...args) {
  const list = listeners.get(eventName) || []
  const keep = []
  for (const item of list) {
    item.callback(...args)
    item.calls += 1
    if (item.maxCallbacks < 0 || item.calls < item.maxCallbacks) keep.push(item)
  }
  if (keep.length) listeners.set(eventName, keep)
  else listeners.delete(eventName)
}

export function BrowserOpenURL(url) {
  if (typeof window !== 'undefined') window.open(url, '_blank')
}

export async function ClipboardSetText(text) {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

export async function ClipboardGetText() {
  try { return await navigator.clipboard.readText() } catch { return '' }
}

export function LogPrint() {}
export function LogTrace() {}
export function LogDebug() {}
export function LogInfo() {}
export function LogWarning() {}
export function LogError() {}
export function LogFatal() {}
export function WindowReload() {}
export function WindowReloadApp() {}
export function WindowSetAlwaysOnTop() {}
export function WindowSetSystemDefaultTheme() {}
export function WindowSetLightTheme() {}
export function WindowSetDarkTheme() {}
export function WindowCenter() {}
export function WindowSetTitle() {}
export function WindowFullscreen() {}
export function WindowUnfullscreen() {}
export function WindowIsFullscreen() { return false }
export function WindowGetSize() { return { w: 1180, h: 720 } }
export function WindowSetSize() {}
export function WindowSetMaxSize() {}
export function WindowSetMinSize() {}
export function WindowSetPosition() {}
export function WindowGetPosition() { return { x: 0, y: 0 } }
export function WindowHide() {}
export function WindowShow() {}
export function WindowMaximise() {}
export function WindowToggleMaximise() {}
export function WindowUnmaximise() {}
export function WindowIsMaximised() { return false }
export function WindowMinimise() {}
export function WindowUnminimise() {}
export function WindowSetBackgroundColour() {}
export function ScreenGetAll() { return [] }
export function WindowIsMinimised() { return false }
export function WindowIsNormal() { return true }
export function Environment() { return { buildType: 'mock', platform: 'browser' } }
export function Quit() {}
export function Hide() {}
export function Show() {}
export function OnFileDrop() {}
export function OnFileDropOff() {}
export function CanResolveFilePaths() { return false }
export function ResolveFilePaths(files) { return files || [] }
