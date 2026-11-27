import { checkHomeDemoLayout } from './homeDemoLayout.browser.mjs'

// Executable browser contract for the layout checker itself. Every fixture uses
// real iframe layout; no DOM queries or geometry APIs are mocked.
export async function checkHomeDemoLayoutContract(scenarios = ['missing terminal', 'hidden terminal', 'missing topbar controls', 'missing icon controls', 'delayed mount', 'stable geometry', 'wrong icon size']) {
  const outcomes = []
  for (const scenario of scenarios) {
    const frame = document.createElement('iframe')
    frame.style.cssText = 'position: fixed; top: 0; left: 0; border: 0; z-index: 2147483647; width: 1440px; height: 900px'
    const markup = `<main class="app-shell"><header class="topbar"><button>Projects</button><button class="icon-only" aria-label="Settings"></button></header><div class="term-host"><div class="xterm"><div class="xterm-screen"></div></div></div></main>`
    frame.srcdoc = `<style>* { box-sizing: border-box } body { margin: 0 } .app-shell { width: 100%; height: 600px } .topbar { display: flex; width: 100%; height: 48px } button { flex: none; height: 28px } .icon-only { width: 28px; height: 28px } .term-host, .xterm, .xterm-screen { width: 100%; height: 400px }</style>${scenario === 'delayed mount' ? '' : markup}`
    const loaded = new Promise((resolve) => frame.addEventListener('load', resolve, { once: true }))
    document.body.append(frame)
    await loaded
    try {
      const doc = frame.contentDocument
      if (scenario === 'missing terminal') doc.querySelector('.term-host').remove()
      if (scenario === 'hidden terminal') doc.querySelector('.term-host').style.display = 'none'
      if (scenario === 'missing topbar controls') doc.querySelector('.topbar').replaceChildren()
      if (scenario === 'missing icon controls') doc.querySelector('.icon-only').remove()
      if (scenario === 'wrong icon size') doc.querySelector('.icon-only').style.width = '29px'
      if (scenario === 'delayed mount') setTimeout(() => doc.body.insertAdjacentHTML('beforeend', markup), 180)
      if (scenario === 'stable geometry') {
        const bar = doc.querySelector('.topbar')
        bar.style.width = '1800px'
        setTimeout(() => { bar.style.width = '100%' }, 180)
      }
      const result = await checkHomeDemoLayout(frame, { timeoutMs: 5000 })
      const expected = ['delayed mount', 'stable geometry'].includes(scenario)
      const expectedFailure = scenario === 'wrong icon size' ? 'fixed icon controls resized' : 'expected a visible mounted terminal'
      const passed = result.passed === expected && (expected || result.failures.every((failure) => failure.includes(expectedFailure)))
      outcomes.push({ scenario, passed, expected, actual: result.passed, failures: result.failures })
    } catch (error) {
      outcomes.push({ scenario, passed: false, error: String(error) })
    } finally { frame.remove() }
  }
  return { passed: outcomes.every((outcome) => outcome.passed), outcomes }
}
