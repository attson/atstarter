// Browser regression: pass a same-origin iframe displaying the real home demo.
// Run with AIPageGo browser_runJS after the frame has mounted:
// const { checkHomeDemoLayout } = await import('/rootyard/.vitepress/theme/components/homeDemoLayout.browser.mjs')
// return await checkHomeDemoLayout(document.querySelector('iframe'))
export async function checkHomeDemoLayout(frame) {
  const measurements = []
  const failures = []
  for (const [width, height] of [[1440, 900], [1024, 768]]) {
    frame.style.width = `${width}px`
    frame.style.height = `${height}px`
    // Keep a mounted terminal while shrinking, so its previous fitted width
    // participates in the real browser's grid sizing calculation.
    await new Promise((resolve) => setTimeout(resolve, 100))
    const win = frame.contentWindow
    const doc = frame.contentDocument
    const app = doc.querySelector('.app-shell')
    const bar = app.querySelector('.topbar')
    const root = doc.documentElement
    const appWidth = app.getBoundingClientRect().width
    const topbarWidth = bar.getBoundingClientRect().width
    const clippedControls = [...bar.querySelectorAll('button')]
      .filter((button) => {
        const bounds = button.getBoundingClientRect()
        const container = app.getBoundingClientRect()
        return bounds.left < container.left || bounds.right > container.right || button.scrollWidth > button.clientWidth
      })
      .map((button) => button.title || button.textContent.trim())
    const icons = [...app.querySelectorAll('.icon-only')]
      .map((button) => ({ width: button.getBoundingClientRect().width, height: button.getBoundingClientRect().height }))
    const measurement = {
      width: win.innerWidth, height: win.innerHeight, appWidth, topbarWidth,
      pageWidth: root.scrollWidth, clientWidth: root.clientWidth, clippedControls, icons,
    }
    measurements.push(measurement)
    if (win.innerWidth !== width || win.innerHeight !== height) failures.push(`${width}x${height}: incorrect frame viewport`)
    if (topbarWidth > appWidth + 1 || clippedControls.length) failures.push(`${width}x${height}: app clips controls after resizing`)
    if (icons.some((icon) => icon.width !== 28 || icon.height !== 28)) failures.push(`${width}x${height}: fixed icon controls resized`)
    if (root.scrollWidth > root.clientWidth) failures.push(`${width}x${height}: horizontal page overflow`)
  }
  return { passed: failures.length === 0, failures, measurements }
}
