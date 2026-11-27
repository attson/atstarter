// Browser regression: pass a same-origin iframe displaying the real home demo.
// Run with AIPageGo browser_runJS after selecting a project with visible logs:
// const { checkHomeDemoLayout } = await import('/rootyard/.vitepress/theme/components/homeDemoLayout.browser.mjs')
// return await checkHomeDemoLayout(document.querySelector('iframe'))
function visible(element, win) {
  if (!element) return false
  const bounds = element.getBoundingClientRect()
  return bounds.width > 0 && bounds.height > 0 &&
    win.getComputedStyle(element).visibility !== 'hidden' &&
    element.checkVisibility?.({ checkOpacity: true, checkVisibilityCSS: true }) !== false
}

function layoutFailures(measurement, width, height) {
  const { appWidth, topbarWidth, clippedControls, icons, pageWidth, clientWidth } = measurement
  const failures = []
  if (measurement.width !== width || measurement.height !== height) failures.push(`${width}x${height}: incorrect frame viewport`)
  if (topbarWidth > appWidth + 1 || clippedControls.length) failures.push(`${width}x${height}: app clips controls after resizing`)
  if (icons.some((icon) => icon.width !== 28 || icon.height !== 28)) failures.push(`${width}x${height}: fixed icon controls resized`)
  if (pageWidth > clientWidth) failures.push(`${width}x${height}: horizontal page overflow`)
  return failures
}

async function stableLayout(frame, width, height, timeoutMs) {
  const deadline = performance.now() + timeoutMs
  let previous = ''
  let stableSamples = 0
  let latestMeasurement
  let reason = 'demo has not mounted'
  while (performance.now() < deadline) {
    const win = frame.contentWindow
    const doc = frame.contentDocument
    const app = doc?.querySelector('.app-shell')
    const bar = app?.querySelector('.topbar')
    const terminal = app?.querySelector('.term-host .xterm .xterm-screen')
    const controls = [...(bar?.querySelectorAll('button') || [])]
    const iconControls = [...(app?.querySelectorAll('.icon-only') || [])]
    const ready = win?.innerWidth === width && win?.innerHeight === height &&
      visible(app, win) && visible(bar, win) && visible(terminal, win) &&
      controls.length > 0 && iconControls.length > 0 &&
      [...controls, ...iconControls].every((element) => visible(element, win))
    if (ready) {
      const root = doc.documentElement
      const container = app.getBoundingClientRect()
      const controlBounds = controls.map((button) => button.getBoundingClientRect())
      const terminalBounds = terminal.getBoundingClientRect()
      const measurement = {
        width: win.innerWidth, height: win.innerHeight,
        appWidth: container.width, topbarWidth: bar.getBoundingClientRect().width,
        pageWidth: root.scrollWidth, clientWidth: root.clientWidth,
        clippedControls: controls.filter((button, index) => {
          const bounds = controlBounds[index]
          return bounds.left < container.left || bounds.right > container.right || button.scrollWidth > button.clientWidth
        }).map((button) => button.title || button.textContent.trim()),
        icons: iconControls.map((button) => ({ width: button.getBoundingClientRect().width, height: button.getBoundingClientRect().height })),
        controlCount: controls.length,
        terminalWidth: terminalBounds.width, terminalHeight: terminalBounds.height,
      }
      const signature = JSON.stringify([measurement, controlBounds])
      stableSamples = signature === previous ? stableSamples + 1 : 0
      previous = signature
      latestMeasurement = measurement
      if (stableSamples >= 3 && layoutFailures(measurement, width, height).length === 0) return { measurement }
      reason = 'demo geometry did not stabilize'
    } else {
      previous = ''
      latestMeasurement = undefined
      stableSamples = 0
      reason = 'expected a visible mounted terminal, topbar buttons, and icon controls at the requested viewport'
    }
    // Poll the measured condition; elapsed time alone never passes the check.
    await new Promise((resolve) => setTimeout(resolve, 25))
  }
  if (latestMeasurement && stableSamples >= 3) return { measurement: latestMeasurement }
  return { error: reason }
}

export async function checkHomeDemoLayout(frame, { timeoutMs = 5000 } = {}) {
  const measurements = []
  const failures = []
  for (const [width, height] of [[1440, 900], [1024, 768]]) {
    frame.style.width = `${width}px`
    frame.style.height = `${height}px`
    // Keep the terminal mounted while shrinking, so its previously fitted width
    // participates in the real browser's grid sizing calculation.
    const { measurement, error } = await stableLayout(frame, width, height, timeoutMs)
    if (error) {
      failures.push(`${width}x${height}: ${error}`)
      continue
    }
    measurements.push(measurement)
    failures.push(...layoutFailures(measurement, width, height))
  }
  return { passed: failures.length === 0, failures, measurements }
}
