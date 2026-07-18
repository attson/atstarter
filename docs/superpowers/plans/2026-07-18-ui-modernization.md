# UI Modernization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land Dark (Linear-style) + Light (shadcn-style) themes on atstarter, tokenize colors/spacing/motion, refresh typography and density, and swap emoji/character icons for lucide — without touching the Go backend, Wails bindings, or the three-panel layout structure.

**Architecture:** Introduce CSS custom-property tokens layered as `tokens.css` (structure) + `theme.dark.css` / `theme.light.css` (color overrides scoped by `<html data-theme>`). Add 3 primitive Vue components (`AppButton`, `AppPill`, `AppIcon`) and a `useTheme` composable. Migrate the 12 existing components to consume tokens + primitives + lucide icons.

**Tech Stack:** Vue 3.2 (Composition API `<script setup>`), Vite 3, plain CSS (no preprocessor), `lucide-vue-next` for icons, `matchMedia` + `localStorage` for theme persistence, `node --test` for logic tests.

**Spec:** [`docs/superpowers/specs/2026-07-18-ui-modernization-design.md`](../specs/2026-07-18-ui-modernization-design.md) — read before starting.

## Global Constraints

- **Do not modify** any Go source (`app.go`, `main.go`, `internal/**`, `frontend/wailsjs/**` auto-generated).
- **Do not modify** existing Wails events or JS-Go binding surface — only visuals/interactions in `frontend/src/**`.
- **Preserve behavior:** every existing user action (add/scan/edit project, start/stop, groups, log tail) must continue to work identically after each task.
- **CSS tokens are the only source of color/spacing:** after Task 9, `grep -RE "#[0-9a-fA-F]{6}" frontend/src/components/ frontend/src/App.vue` must return no hits (SVG `stroke="currentColor"` is fine because it's a keyword, not hex).
- **Log panel stays dark** in both themes.
- **Vue file pattern:** every component uses `<script setup>` (matches existing codebase).
- **No new bundlers/preprocessors.** Only add the runtime dependency `lucide-vue-next`.
- **Wails dev command:** on Ubuntu 24.04 use `wails dev -tags webkit2_41`; on macOS `wails dev` suffices (per README).
- **Commit after every task** with the exact message template shown in that task's final step.

## File Structure

**New files (frontend/src):**
- `styles/tokens.css` — size/motion/z-index tokens + font stacks + shared structural styles
- `styles/theme.dark.css` — dark color-token overrides (default when `data-theme="dark"`)
- `styles/theme.light.css` — light color-token overrides (default when `data-theme="light"`)
- `composables/useTheme.js` — theme state + persistence + matchMedia binding
- `composables/useTheme.test.mjs` — node --test coverage for the composable's pure logic
- `components/ui/AppButton.vue` — Button primitive (4 variants × 2 sizes)
- `components/ui/AppPill.vue` — Status pill primitive (5 variants)
- `components/ui/AppIcon.vue` — Thin lucide wrapper (uniform stroke + size)
- `components/ui/ThemeToggle.vue` — Top-bar theme cycle button

**Modified files:**
- `frontend/package.json` — add `lucide-vue-next` dependency
- `frontend/src/main.js` — import 3 CSS files + init `useTheme`
- `frontend/src/style.css` — drop Nunito @font-face, keep alignment
- `frontend/src/App.vue` — token-ize topbar + insert `ThemeToggle` + swap buttons for `AppButton`
- `frontend/src/components/ProjectList.vue` — tokens + Search icon + width tokens
- `frontend/src/components/ProjectTreeNode.vue` — tokens + lucide chevron + tighter density
- `frontend/src/components/GroupTreeItem.vue` — tokens + lucide `FolderKanban` for badge
- `frontend/src/components/GroupTreeNode.vue` — tokens (used inside GroupDialog)
- `frontend/src/components/ProjectDetail.vue` — tokens + `AppButton` + `AppPill` + lucide
- `frontend/src/components/GroupDetail.vue` — tokens + primitives + lucide
- `frontend/src/components/LogPanel.vue` — token-ize borders, deepen bg to `#06070a`, sticky banner
- `frontend/src/components/EditProjectDialog.vue` — tokens + `AppButton` + dialog entrance transition
- `frontend/src/components/ScanDialog.vue` — tokens + `AppButton` + FolderOpen icon + transition
- `frontend/src/components/GroupDialog.vue` — tokens + `AppButton` + transition
- `frontend/src/components/AddProjectDialog.vue` — tokens + `AppButton` + transition
- `frontend/src/components/AddToGroupDialog.vue` — tokens + `AppButton` + transition

**Deleted files:**
- `frontend/src/assets/fonts/nunito-v16-latin-regular.woff2` (unused after style.css cleanup)

---

## Task 1: Design tokens layer + default dark theme

**Files:**
- Create: `frontend/src/styles/tokens.css`
- Create: `frontend/src/styles/theme.dark.css`
- Create: `frontend/src/styles/theme.light.css`
- Modify: `frontend/src/main.js`
- Modify: `frontend/src/style.css` (drop Nunito)
- Modify: `frontend/index.html` (add `data-theme="dark"` on `<html>`)

**Interfaces:**
- Consumes: none.
- Produces: CSS custom properties documented in the spec sections 3–4. All later tasks read from these variable names.

**Verification** (per step 6 below): the app opens dark by default and existing text/buttons still render (styling remains via the old scoped rules until later tasks migrate them). This proves tokens load without breaking anything.

- [ ] **Step 1: Write `frontend/src/styles/tokens.css`** — size + motion + typography + shared root

```css
:root {
  /* Spacing */
  --space-1: 2px; --space-2: 4px; --space-3: 6px;
  --space-4: 8px; --space-5: 10px; --space-6: 12px;
  --space-7: 16px; --space-8: 20px; --space-9: 24px; --space-10: 32px;

  /* Radius */
  --radius-sm: 4px;
  --radius-md: 7px;
  --radius-lg: 10px;
  --radius-full: 999px;

  /* Shadow */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, .04);
  --shadow-md: 0 8px 24px rgba(0, 0, 0, .10);
  --shadow-lg: 0 20px 40px rgba(0, 0, 0, .16);

  /* Motion */
  --dur-fast: 120ms;
  --dur-base: 200ms;
  --dur-slow: 320ms;
  --ease: cubic-bezier(.2, 0, 0, 1);

  /* Z-index */
  --z-menu: 20;
  --z-modal: 40;
  --z-toast: 60;

  /* Typography */
  --font-sans:
    -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto",
    "PingFang SC", "Microsoft YaHei", "Helvetica Neue", sans-serif;
  --font-mono:
    "SFMono-Regular", ui-monospace, Consolas, "Liberation Mono", monospace;

  --fs-lg: 22px;
  --fs-md: 16px;
  --fs-base: 13px;
  --fs-sm: 12px;
  --fs-xs: 11px;
  --fs-mono: 12px;

  --fw-regular: 400;
  --fw-medium: 500;
  --fw-semibold: 600;
}

html {
  font-family: var(--font-sans);
  font-size: var(--fs-base);
  color: var(--text);
  background: var(--bg);
  transition:
    background-color var(--dur-base) var(--ease),
    color var(--dur-base) var(--ease),
    border-color var(--dur-base) var(--ease);
}

@keyframes pulse-ring {
  0%, 100% { box-shadow: 0 0 0 2.5px var(--success-soft); }
  50%      { box-shadow: 0 0 0 4px var(--success-soft); }
}
```

- [ ] **Step 2: Write `frontend/src/styles/theme.dark.css`** — Dark color tokens

```css
html[data-theme="dark"] {
  --bg: #0a0b0f;
  --surface: #0c0e13;
  --elevated: #171921;
  --border: #1a1c22;
  --border-strong: #23262f;

  --text: #f4f5f7;
  --text-secondary: #d5d8de;
  --text-muted: #a6acb9;
  --text-subtle: #5e6371;

  --primary: #f4f5f7;
  --primary-fg: #0a0b0f;

  --success: #4ade80;
  --success-fg: #052e1a;
  --success-soft: rgba(74, 222, 128, .09);
  --success-line: rgba(74, 222, 128, .18);

  --warning: #f2c56b;
  --warning-soft: rgba(251, 191, 36, .08);
  --warning-line: rgba(251, 191, 36, .18);

  --danger: #ef4444;
  --danger-fg: #fca5a5;
  --danger-soft: rgba(239, 68, 68, .10);
  --danger-line: rgba(239, 68, 68, .22);

  --focus-ring: rgba(244, 245, 247, .18);
  --overlay: rgba(0, 0, 0, .55);
}
```

- [ ] **Step 3: Write `frontend/src/styles/theme.light.css`** — Light color tokens

```css
html[data-theme="light"] {
  --bg: #ffffff;
  --surface: #fafaf9;
  --elevated: #f4f4f5;
  --border: #e7e5e4;
  --border-strong: #e4e4e7;

  --text: #18181b;
  --text-secondary: #3f3f46;
  --text-muted: #52525b;
  --text-subtle: #a1a1aa;

  --primary: #18181b;
  --primary-fg: #fafaf9;

  --success: #16a34a;
  --success-fg: #ffffff;
  --success-soft: #dcfce7;
  --success-line: #bbf7d0;

  --warning: #a16207;
  --warning-soft: #fef9c3;
  --warning-line: #fde68a;

  --danger: #b91c1c;
  --danger-fg: #b91c1c;
  --danger-soft: #fef2f2;
  --danger-line: #fecaca;

  --focus-ring: rgba(24, 24, 27, .12);
  --overlay: rgba(0, 0, 0, .45);
}
```

- [ ] **Step 4: Rewrite `frontend/src/style.css`** — drop Nunito, keep alignment reset

```css
html {
  text-align: left;
}

body {
  margin: 0;
}

#app {
  height: 100vh;
  text-align: left;
}

button,
input,
textarea {
  font-family: inherit;
}
```

- [ ] **Step 5: Rewrite `frontend/src/main.js`** — import token layers before existing CSS

```js
import { createApp } from 'vue'
import App from './App.vue'
import './styles/tokens.css'
import './styles/theme.dark.css'
import './styles/theme.light.css'
import './style.css'

createApp(App).mount('#app')
```

- [ ] **Step 6: Add `data-theme="dark"` to `<html>` in `frontend/index.html`**

Open `frontend/index.html`, find the `<html …>` tag, add the attribute:

```html
<html lang="en" data-theme="dark">
```

If there's no lang attribute, just add `data-theme="dark"`. This is a temporary hardcode; Task 2 replaces it with runtime `useTheme` control.

- [ ] **Step 7: Verify in dev**

Run: `wails dev` (or `wails dev -tags webkit2_41` on Ubuntu 24.04).

Expected: window opens, background is now near-black `#0a0b0f`, all existing components still functional (buttons look wrong because their scoped styles still hardcode light colors — that's expected; they migrate in later tasks). No console errors.

Kill dev with Ctrl-C when confirmed.

- [ ] **Step 8: Delete unused Nunito font file**

Run: `rm frontend/src/assets/fonts/nunito-v16-latin-regular.woff2`
(Ignore if the file already doesn't exist.)

- [ ] **Step 9: Commit**

```bash
git add frontend/src/styles frontend/src/main.js frontend/src/style.css frontend/index.html frontend/src/assets/fonts/
git commit -m "feat(ui): add design tokens + dark/light theme foundations"
```

---

## Task 2: `useTheme` composable + theme cycling logic

**Files:**
- Create: `frontend/src/composables/useTheme.js`
- Create: `frontend/src/composables/useTheme.test.mjs`
- Modify: `frontend/src/main.js` (invoke `useTheme().init()`)
- Modify: `frontend/index.html` (remove hardcoded `data-theme="dark"` — runtime takes over)

**Interfaces:**
- Consumes: `matchMedia`, `localStorage`, `document.documentElement` (browser globals).
- Produces:
  - `useTheme(): { theme, resolvedTheme, cycleTheme, init }`
    - `theme: Ref<'system' | 'dark' | 'light'>`
    - `resolvedTheme: ComputedRef<'dark' | 'light'>`
    - `cycleTheme(): void` — advances `system → dark → light → system`
    - `init(): void` — reads `localStorage`, applies theme to `<html>`, binds `matchMedia` change listener when in `'system'` mode
  - Pure helpers exported for testing:
    - `resolveTheme(theme: string, prefersDark: boolean): 'dark' | 'light'`
    - `nextTheme(theme: string): 'system' | 'dark' | 'light'`

- [ ] **Step 1: Write the failing test at `frontend/src/composables/useTheme.test.mjs`**

```js
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { resolveTheme, nextTheme } from './useTheme.js'

test('resolveTheme: system + prefers dark → dark', () => {
  assert.equal(resolveTheme('system', true), 'dark')
})

test('resolveTheme: system + prefers light → light', () => {
  assert.equal(resolveTheme('system', false), 'light')
})

test('resolveTheme: explicit dark ignores system', () => {
  assert.equal(resolveTheme('dark', false), 'dark')
})

test('resolveTheme: explicit light ignores system', () => {
  assert.equal(resolveTheme('light', true), 'light')
})

test('resolveTheme: falls back to dark when input invalid', () => {
  assert.equal(resolveTheme('garbage', true), 'dark')
  assert.equal(resolveTheme(undefined, false), 'light')
})

test('nextTheme cycles system → dark → light → system', () => {
  assert.equal(nextTheme('system'), 'dark')
  assert.equal(nextTheme('dark'), 'light')
  assert.equal(nextTheme('light'), 'system')
})

test('nextTheme: unknown input starts a fresh cycle at dark', () => {
  assert.equal(nextTheme('unknown'), 'dark')
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test frontend/src/composables/useTheme.test.mjs`
Expected: FAIL — module not found or exports missing.

- [ ] **Step 3: Write `frontend/src/composables/useTheme.js`**

```js
import { ref, computed, watchEffect } from 'vue'

const STORAGE_KEY = 'atstarter.theme'
const VALID = new Set(['system', 'dark', 'light'])
const CYCLE = { system: 'dark', dark: 'light', light: 'system' }

export function resolveTheme(theme, prefersDark) {
  if (theme === 'dark' || theme === 'light') return theme
  if (theme === 'system') return prefersDark ? 'dark' : 'light'
  return prefersDark ? 'dark' : 'light'
}

export function nextTheme(theme) {
  return CYCLE[theme] || 'dark'
}

function loadTheme() {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    return VALID.has(value) ? value : 'system'
  } catch {
    return 'system'
  }
}

function saveTheme(value) {
  try { localStorage.setItem(STORAGE_KEY, value) } catch {}
}

const theme = ref(loadTheme())
const media = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null
const prefersDark = ref(media ? media.matches : true)

if (media) {
  const listener = (event) => { prefersDark.value = event.matches }
  if (media.addEventListener) media.addEventListener('change', listener)
  else media.addListener(listener)
}

const resolvedTheme = computed(() => resolveTheme(theme.value, prefersDark.value))

function applyTheme(value) {
  if (typeof document === 'undefined') return
  document.documentElement.setAttribute('data-theme', value)
}

function cycleTheme() {
  const value = nextTheme(theme.value)
  theme.value = value
  saveTheme(value)
}

let initialized = false
function init() {
  if (initialized) return
  initialized = true
  watchEffect(() => applyTheme(resolvedTheme.value))
}

export function useTheme() {
  return { theme, resolvedTheme, cycleTheme, init }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test frontend/src/composables/useTheme.test.mjs`
Expected: PASS — 7 tests.

- [ ] **Step 5: Wire `useTheme.init()` into `frontend/src/main.js`**

```js
import { createApp } from 'vue'
import App from './App.vue'
import './styles/tokens.css'
import './styles/theme.dark.css'
import './styles/theme.light.css'
import './style.css'
import { useTheme } from './composables/useTheme.js'

useTheme().init()
createApp(App).mount('#app')
```

- [ ] **Step 6: Remove hardcoded `data-theme="dark"` from `frontend/index.html`**

Change back to plain `<html lang="…">` (or `<html>` if there was no lang). The composable now controls the attribute at runtime.

- [ ] **Step 7: Manual verification in dev**

Run: `wails dev` (or with `-tags webkit2_41` on Ubuntu 24.04).

Expected on first launch (fresh localStorage):
- macOS in dark mode → app dark
- macOS in light mode → app light

Then in Chrome DevTools console (via WebKit inspector) run:
```
localStorage.setItem('atstarter.theme', 'light')
location.reload()
```
Expected: app forces light. Repeat with `'dark'` and `'system'` values.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/composables frontend/src/main.js frontend/index.html
git commit -m "feat(ui): add useTheme composable with system/dark/light cycling"
```

---

## Task 3: Install lucide + primitive components (`AppButton`, `AppPill`, `AppIcon`, `ThemeToggle`)

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/src/components/ui/AppIcon.vue`
- Create: `frontend/src/components/ui/AppButton.vue`
- Create: `frontend/src/components/ui/AppPill.vue`
- Create: `frontend/src/components/ui/ThemeToggle.vue`

**Interfaces:**
- Produces:
  - `<AppIcon :icon="LucideComponent" :size="14" />` — renders lucide with uniform stroke.
  - `<AppButton variant="primary|secondary|success|danger" size="sm|md" :disabled="…" :icon-only="…">`
  - `<AppPill variant="running|exited|error|stopped|neutral" :dot="…">`
  - `<ThemeToggle />` — self-contained; consumes `useTheme()` internally, shows `Sun | Moon | Monitor` based on `theme.value`, cycles on click.

Primitives coexist with old components; nothing is migrated yet.

- [ ] **Step 1: Add lucide-vue-next to dependencies**

Edit `frontend/package.json`, add to `dependencies`:

```json
{
  "name": "frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.2.37",
    "lucide-vue-next": "^0.462.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^3.0.3",
    "vite": "^3.0.7"
  }
}
```

Then install:

Run: `cd frontend && npm install`
Expected: writes `frontend/node_modules/lucide-vue-next/…` and updates `package-lock.json` (or generates one).

- [ ] **Step 2: Create `frontend/src/components/ui/AppIcon.vue`**

```vue
<script setup>
defineProps({
  icon: { type: [Object, Function], required: true },
  size: { type: Number, default: 14 },
  strokeWidth: { type: Number, default: 1.75 },
})
</script>

<template>
  <component :is="icon" :size="size" :stroke-width="strokeWidth" class="app-icon" />
</template>

<style scoped>
.app-icon {
  display: inline-block;
  flex-shrink: 0;
  vertical-align: -2px;
}
</style>
```

- [ ] **Step 3: Create `frontend/src/components/ui/AppButton.vue`**

```vue
<script setup>
defineProps({
  variant: { type: String, default: 'secondary' },
  size: { type: String, default: 'md' },
  disabled: { type: Boolean, default: false },
  iconOnly: { type: Boolean, default: false },
  type: { type: String, default: 'button' },
})
const emit = defineEmits(['click'])
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    :class="['app-btn', `variant-${variant}`, `size-${size}`, { 'icon-only': iconOnly }]"
    @click="emit('click', $event)"
  >
    <span v-if="$slots.icon" class="icon-slot"><slot name="icon" /></span>
    <span v-if="!iconOnly" class="label"><slot /></span>
  </button>
</template>

<style scoped>
.app-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  height: 30px;
  padding: 0 var(--space-6);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  line-height: 1;
  cursor: pointer;
  transition:
    background var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.app-btn:disabled {
  cursor: not-allowed;
  opacity: .45;
}

.app-btn.size-sm { height: 26px; padding: 0 var(--space-5); }
.app-btn.icon-only { width: 30px; padding: 0; }
.app-btn.size-sm.icon-only { width: 26px; }

.app-btn .icon-slot { display: inline-flex; align-items: center; }

.app-btn.variant-primary {
  background: var(--primary);
  color: var(--primary-fg);
}
.app-btn.variant-primary:hover:not(:disabled) { filter: brightness(.94); }

.app-btn.variant-secondary {
  background: var(--elevated);
  color: var(--text-secondary);
  border-color: var(--border-strong);
}
.app-btn.variant-secondary:hover:not(:disabled) {
  background: color-mix(in srgb, var(--elevated) 82%, var(--text) 10%);
}

.app-btn.variant-success {
  background: var(--success);
  color: var(--success-fg);
}
.app-btn.variant-success:hover:not(:disabled) { filter: brightness(1.05); }

.app-btn.variant-danger {
  background: var(--danger-soft);
  color: var(--danger-fg);
  border-color: var(--danger-line);
}
.app-btn.variant-danger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--danger-soft) 70%, var(--danger) 15%);
}

.app-btn:focus-visible {
  outline: 0;
  box-shadow: 0 0 0 3px var(--focus-ring);
}
</style>
```

- [ ] **Step 4: Create `frontend/src/components/ui/AppPill.vue`**

```vue
<script setup>
defineProps({
  variant: { type: String, default: 'neutral' },
  dot: { type: Boolean, default: false },
})
</script>

<template>
  <span :class="['app-pill', `variant-${variant}`]">
    <span v-if="dot" class="pill-dot" />
    <slot />
  </span>
</template>

<style scoped>
.app-pill {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px 9px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  line-height: 1.4;
  white-space: nowrap;
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.app-pill.variant-running {
  color: var(--success);
  background: var(--success-soft);
  border-color: var(--success-line);
}

.app-pill.variant-exited {
  color: var(--warning);
  background: var(--warning-soft);
  border-color: var(--warning-line);
}

.app-pill.variant-error {
  color: var(--danger-fg);
  background: var(--danger-soft);
  border-color: var(--danger-line);
}

.app-pill.variant-stopped {
  color: var(--text-muted);
  background: var(--elevated);
  border-color: var(--border-strong);
}

.app-pill.variant-neutral {
  color: var(--text-muted);
  background: transparent;
  border-color: var(--border-strong);
}
</style>
```

- [ ] **Step 5: Create `frontend/src/components/ui/ThemeToggle.vue`**

```vue
<script setup>
import { computed } from 'vue'
import { Sun, Moon, Monitor } from 'lucide-vue-next'
import { useTheme } from '../../composables/useTheme.js'
import AppIcon from './AppIcon.vue'
import AppButton from './AppButton.vue'

const { theme, cycleTheme } = useTheme()
const icon = computed(() => {
  if (theme.value === 'dark') return Moon
  if (theme.value === 'light') return Sun
  return Monitor
})
const label = computed(() => `Theme: ${theme.value}`)
</script>

<template>
  <AppButton
    variant="secondary"
    size="sm"
    icon-only
    :aria-label="label"
    :title="label"
    @click="cycleTheme"
  >
    <template #icon><AppIcon :icon="icon" :size="14" /></template>
  </AppButton>
</template>
```

- [ ] **Step 6: Smoke-test build**

Run: `cd frontend && npm run build`
Expected: build succeeds without errors. lucide-vue-next resolves.

- [ ] **Step 7: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/components/ui/
git commit -m "feat(ui): install lucide + primitives (AppButton, AppPill, AppIcon, ThemeToggle)"
```

---

## Task 4: Migrate top bar (`App.vue`) to tokens + primitives + `ThemeToggle`

**Files:**
- Modify: `frontend/src/App.vue`

**Interfaces:**
- Consumes: `AppButton`, `AppPill`, `AppIcon`, `ThemeToggle` from Task 3; tokens from Task 1.
- Produces: no new API; template refactor.

**Behavioral guarantee:** every button in the topbar still emits the same events (`showAddProject`, `showScan`, opening `GroupDialog`), just rendered as `AppButton`. Summary pills still show counts.

- [ ] **Step 1: Replace the `<template>` block in `frontend/src/App.vue`**

Find the `<template>` block and replace **only** the `<header class="topbar">…</header>` region with:

```vue
<header class="topbar">
  <div class="brand">atstarter</div>
  <div class="summary">
    <span class="summary-count">{{ projects.length }} projects</span>
    <AppPill variant="running" dot>{{ runningCount }} running</AppPill>
    <AppPill variant="exited">{{ exitedCount }} exited</AppPill>
  </div>
  <div class="top-actions">
    <ThemeToggle />
    <AppButton variant="secondary" size="sm" @click="editingGroup = null; showGroup = true">
      <template #icon><AppIcon :icon="FolderPlus" :size="14" /></template>
      New Group
    </AppButton>
    <AppButton variant="secondary" size="sm" @click="showScan = true">
      <template #icon><AppIcon :icon="Radar" :size="14" /></template>
      Scan
    </AppButton>
    <AppButton variant="primary" size="sm" @click="showAddProject = true">
      <template #icon><AppIcon :icon="Plus" :size="14" /></template>
      Add
    </AppButton>
  </div>
</header>
```

Leave the rest of the template (`<main>`, dialogs, etc.) untouched.

- [ ] **Step 2: Add imports to `<script setup>` in `App.vue`**

Just below the existing dialog component imports, add:

```js
import AppButton from './components/ui/AppButton.vue'
import AppPill from './components/ui/AppPill.vue'
import AppIcon from './components/ui/AppIcon.vue'
import ThemeToggle from './components/ui/ThemeToggle.vue'
import { FolderPlus, Radar, Plus } from 'lucide-vue-next'
```

- [ ] **Step 3: Replace the `<style>` block in `App.vue` with token-driven rules**

Replace the entire `<style>` block (the one that starts with `html, body, #app { height: 100%; margin: 0; }`) with:

```css
html, body, #app { height: 100%; margin: 0; }

.app-shell {
  display: grid;
  grid-template-rows: 48px 1fr;
  height: 100vh;
  font-family: var(--font-sans);
  background: var(--bg);
  color: var(--text);
}

.topbar {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: var(--space-7);
  padding: 0 var(--space-7);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
}

.brand {
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.01em;
}

.summary {
  display: flex;
  align-items: center;
  min-width: 0;
  flex-wrap: wrap;
  gap: var(--space-4);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}

.summary-count {
  font-weight: var(--fw-medium);
}

.top-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: var(--space-3);
}

.workspace {
  min-height: 0;
  display: flex;
}

@media (max-width: 820px) {
  .topbar {
    gap: var(--space-5);
    padding: 0 var(--space-6);
  }
}
</style>
```

- [ ] **Step 4: Verify in dev**

Run: `wails dev`
Expected:
- Topbar is 48px, dark background matches shell, brand + summary readable
- New Group / Scan / Add buttons render with icons; hover changes background
- Theme toggle icon cycles between Monitor / Moon / Sun on click; app instantly re-themes; the choice persists after full quit + relaunch
- Existing dialogs still open when buttons clicked (they'll look mismatched until later tasks)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.vue
git commit -m "feat(ui): migrate topbar to tokens + primitives + theme toggle"
```

---

## Task 5: Migrate project list + tree components

**Files:**
- Modify: `frontend/src/components/ProjectList.vue`
- Modify: `frontend/src/components/ProjectTreeNode.vue`
- Modify: `frontend/src/components/GroupTreeItem.vue`

**Interfaces:**
- Consumes: `AppIcon`, tokens, `lucide-vue-next` icons (`Search`, `ChevronRight`, `ChevronDown`, `FolderKanban`).
- Produces: no new event/prop changes.

**Behavioral guarantee:** search, click-to-select, expand/collapse, group vs project distinction all unchanged.

- [ ] **Step 1: Rewrite `frontend/src/components/ProjectList.vue`**

Full replacement:

```vue
<script setup>
import { computed, ref, watch } from 'vue'
import { Search } from 'lucide-vue-next'
import { buildProjectTree } from '../projectTree'
import ProjectTreeNode from './ProjectTreeNode.vue'
import GroupTreeItem from './GroupTreeItem.vue'
import AppIcon from './ui/AppIcon.vue'

const emit = defineEmits(['select', 'select-group', 'select-command', 'add', 'scan'])

const props = defineProps({ projects: Array, groups: Array, selectedId: String, selectedGroupId: String, statuses: Object })
const query = ref('')
const expandedDirs = ref({})
const expandedGroups = ref({})

const tree = computed(() => buildProjectTree(props.projects || [], props.statuses || {}, query.value))
const forceExpanded = computed(() => query.value.trim().length > 0)

function toggleDir(id) {
  expandedDirs.value = {
    ...expandedDirs.value,
    [id]: expandedDirs.value[id] === false,
  }
}

function toggleGroup(id) {
  expandedGroups.value = {
    ...expandedGroups.value,
    [id]: expandedGroups.value[id] === false,
  }
}

watch(() => props.projects, () => {
  expandedDirs.value = {}
})
</script>

<template>
  <aside class="project-list">
    <div class="search-wrap">
      <div class="search-field">
        <AppIcon :icon="Search" :size="14" class="search-icon" />
        <input v-model="query" class="search" placeholder="Search projects, path, command…" />
      </div>
    </div>
    <div class="tree-scroll">
      <div v-if="(groups || []).length" class="group-section">
        <div class="section-title">Groups</div>
        <GroupTreeItem
          v-for="g in groups"
          :key="g.id"
          :group="g"
          :projects="projects"
          :selected="g.id === selectedGroupId"
          :expanded="expandedGroups[`group:${g.id}`] !== false"
          @select="emit('select-group', $event)"
          @toggle="toggleGroup"
          @select-command="emit('select-command', $event)"
        />
      </div>
      <ProjectTreeNode
        v-for="node in tree"
        :key="node.id"
        :node="node"
        :selectedId="selectedId"
        :level="0"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        @select="emit('select', $event)"
        @toggle="toggleDir"
      />
      <div v-if="tree.length === 0" class="empty">
        <span v-if="query">没有匹配的项目</span>
        <span v-else>还没有项目。点击 Add 或 Scan 开始。</span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.project-list {
  width: 300px;
  min-width: 280px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  background: var(--surface);
  min-height: 0;
}

.search-wrap {
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.search-field {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: var(--space-4);
  color: var(--text-muted);
  pointer-events: none;
}

.section-title {
  color: var(--text-subtle);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
  text-transform: uppercase;
  margin: var(--space-2) var(--space-2) var(--space-3);
}

.search {
  width: 100%;
  box-sizing: border-box;
  height: 28px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: var(--elevated);
  color: var(--text);
  padding: 0 var(--space-4) 0 30px;
  font: inherit;
  font-size: var(--fs-sm);
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.search:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.tree-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
}

.group-section {
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border);
}

.empty {
  color: var(--text-muted);
  font-size: var(--fs-sm);
  padding: var(--space-7) var(--space-5);
}
</style>
```

- [ ] **Step 2: Rewrite `frontend/src/components/ProjectTreeNode.vue`**

Full replacement:

```vue
<script setup>
import { ChevronRight, ChevronDown } from 'lucide-vue-next'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({
  node: Object,
  selectedId: String,
  level: Number,
  expandedDirs: Object,
  forceExpanded: Boolean,
})
const emit = defineEmits(['select', 'toggle'])

function stateClass(state) {
  if (state === 'running') return 'running'
  if (state === 'error' || state === 'exited') return 'bad'
  return 'stopped'
}

function isExpanded(node) {
  return props.forceExpanded || props.expandedDirs[node.id] !== false
}

function hasChildren(node) {
  return node.children && node.children.length > 0
}
</script>

<template>
  <div v-if="node.type === 'directory'" class="tree-group">
    <button
      class="tree-row dir-row"
      :style="{ paddingLeft: `${10 + level * 16}px` }"
      @click="emit('toggle', node.id)"
    >
      <span class="chev">
        <AppIcon :icon="isExpanded(node) ? ChevronDown : ChevronRight" :size="12" />
      </span>
      <span class="dir-name">{{ node.label }}</span>
      <span class="count">{{ node.count }}</span>
    </button>
    <div v-if="isExpanded(node)" class="children">
      <ProjectTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selectedId="selectedId"
        :level="level + 1"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        @select="emit('select', $event)"
        @toggle="emit('toggle', $event)"
      />
    </div>
  </div>

  <div v-else class="tree-group">
    <div
      :class="['tree-row', 'project-row', { active: node.project.id === selectedId }]"
      :style="{ paddingLeft: `${12 + level * 16}px` }"
    >
      <button
        v-if="hasChildren(node)"
        class="project-toggle"
        @click.stop="emit('toggle', node.id)"
      >
        <AppIcon :icon="isExpanded(node) ? ChevronDown : ChevronRight" :size="12" />
      </button>
      <span v-else class="project-spacer" />
      <button class="project-main" @click="emit('select', node.project.id)">
        <span :class="['status-dot', stateClass((node.status || {}).State)]" />
        <span class="project-copy">
          <span class="project-name">{{ node.project.name }}</span>
        </span>
      </button>
      <span v-if="hasChildren(node)" class="count">{{ node.count }}</span>
      <span v-else class="type-pill">{{ node.project.detectedType || 'unknown' }}</span>
    </div>
    <div v-if="hasChildren(node) && isExpanded(node)" class="children">
      <ProjectTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selectedId="selectedId"
        :level="level + 1"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        @select="emit('select', $event)"
        @toggle="emit('toggle', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.tree-group { min-width: 0; }

.tree-row {
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.dir-row {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  height: 26px;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  border-radius: var(--radius-sm);
  padding-right: var(--space-3);
}

.dir-row:hover { background: var(--elevated); }

.chev, .count { color: var(--text-muted); }

.count {
  font-weight: var(--fw-regular);
  font-size: var(--fs-xs);
  padding-right: var(--space-3);
}

.dir-name, .project-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.children { position: relative; }

.project-row {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  min-height: 28px;
  margin: 1px 0;
  padding-top: 2px;
  padding-bottom: 2px;
  padding-right: var(--space-4);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  transition: background var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.project-row:hover { background: var(--elevated); }

.project-row.active {
  background: var(--elevated);
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.running {
  background: var(--success);
  animation: pulse-ring 2s ease-in-out infinite;
}

.status-dot.bad { background: var(--danger); box-shadow: 0 0 0 2.5px var(--danger-soft); }
.status-dot.stopped { background: var(--text-subtle); }

.project-toggle, .project-main {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.project-toggle {
  width: 14px;
  padding: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.project-spacer { width: 14px; }

.project-main {
  min-width: 0;
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr);
  align-items: center;
  gap: var(--space-4);
  padding: 0;
  text-align: left;
}

.project-copy {
  display: flex;
  align-items: center;
  min-width: 0;
}

.project-name {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.type-pill {
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: transparent;
  padding: 1px 7px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}
</style>
```

- [ ] **Step 3: Rewrite `frontend/src/components/GroupTreeItem.vue`**

Full replacement:

```vue
<script setup>
import { computed } from 'vue'
import { ChevronDown, ChevronRight, FolderKanban } from 'lucide-vue-next'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ group: Object, projects: Array, selected: Boolean, expanded: Boolean })
const emit = defineEmits(['select', 'toggle', 'select-command'])

function commandsFor(project) {
  if (project.commands && project.commands.length) return project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    isDefault: true,
  }]
}

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

const projectById = computed(() => {
  const out = {}
  for (const project of props.projects || []) out[project.id] = project
  return out
})

const members = computed(() => {
  const out = []
  for (const item of props.group.items || []) {
    const project = projectById.value[item.projectId]
    if (!project) continue
    const command = commandsFor(project).find((c) => c.id === (item.commandId || 'default'))
    if (!command) continue
    out.push({
      key: `${props.group.id}:${project.id}:${command.id || 'default'}`,
      project,
      command,
    })
  }
  return out
})
</script>

<template>
  <div class="group-wrap">
    <div :class="['group-item', { active: selected }]">
      <button class="toggle" @click.stop="emit('toggle', `group:${group.id}`)">
        <AppIcon :icon="expanded ? ChevronDown : ChevronRight" :size="12" />
      </button>
      <button class="group-main" @click="emit('select', group.id)">
        <span class="group-badge"><AppIcon :icon="FolderKanban" :size="12" /></span>
        <span class="group-copy">
          <span class="group-name">{{ group.name }}</span>
          <span class="group-count">{{ (group.items || []).length }} commands</span>
        </span>
      </button>
      <span class="count">{{ (group.items || []).length }}</span>
    </div>
    <div v-if="expanded" class="members">
      <button
        v-for="member in members"
        :key="member.key"
        class="member-row"
        @click="emit('select-command', { projectId: member.project.id, commandId: member.command.id || 'default' })"
      >
        <span class="member-dot" />
        <span class="member-project">{{ member.project.name }}</span>
        <span class="member-command">{{ member.command.name }}</span>
        <code>{{ lineFor(member.command) }}</code>
      </button>
    </div>
  </div>
</template>

<style scoped>
.group-wrap { min-width: 0; }

.group-item {
  width: 100%;
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  min-height: 28px;
  margin: 1px 0;
  padding: 3px var(--space-4) 3px var(--space-5);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.group-item:hover { background: var(--elevated); }

.group-item.active {
  background: var(--elevated);
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.toggle, .group-main {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.toggle {
  width: 14px;
  padding: 0;
  color: var(--text-muted);
}

.group-main {
  min-width: 0;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: var(--space-4);
  padding: 0;
  text-align: left;
}

.group-badge {
  width: 17px;
  height: 17px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text);
  background: var(--elevated);
  border: 1px solid var(--border-strong);
}

.group-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.group-name, .group-count {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-name {
  color: var(--text);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.group-count {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.count {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.members { padding: 1px 0 var(--space-3); }

.member-row {
  width: 100%;
  min-height: 26px;
  display: grid;
  grid-template-columns: 9px minmax(92px, 1fr) minmax(72px, 96px) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-3);
  padding: 3px var(--space-4) 3px 38px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.member-row:hover { background: var(--elevated); }

.member-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-subtle);
}

.member-project, .member-command, .member-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-project {
  color: var(--text);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.member-command {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.member-row code {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-family: var(--font-mono);
}
</style>
```

- [ ] **Step 4: Verify projectTree.test.mjs still passes**

Run: `node --test frontend/src/projectTree.test.mjs`
Expected: PASS (we didn't touch the logic, only the components).

- [ ] **Step 5: Manual dev check**

Run: `wails dev`
Expected:
- Sidebar 300px, dark surface, `Search` icon inside the input
- Chevrons are lucide, not `▸ ▾`
- Selected project row has subtle border ring (no blue-ish tint)
- Running dot pulses; stopped dot flat gray
- Groups section renders with `FolderKanban` icon replacing the `G` badge

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/ProjectList.vue frontend/src/components/ProjectTreeNode.vue frontend/src/components/GroupTreeItem.vue
git commit -m "feat(ui): migrate project sidebar to tokens + lucide + tighter density"
```

---

## Task 6: Migrate detail panels (`ProjectDetail.vue`, `GroupDetail.vue`)

**Files:**
- Modify: `frontend/src/components/ProjectDetail.vue`
- Modify: `frontend/src/components/GroupDetail.vue`

**Interfaces:**
- Consumes: `AppButton`, `AppPill`, `AppIcon`, tokens.
- Produces: same events / props as before.

- [ ] **Step 1: Rewrite `frontend/src/components/ProjectDetail.vue`**

Full replacement:

```vue
<script setup>
import { computed, ref, watch } from 'vue'
import { Play, Square, Pencil, FolderPlus, ChevronDown, ChevronUp } from 'lucide-vue-next'
import LogPanel from './LogPanel.vue'
import AppButton from './ui/AppButton.vue'
import AppPill from './ui/AppPill.vue'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ project: Object, status: Object, selectedCommandId: String })
const emit = defineEmits(['start', 'stop', 'edit', 'command-change', 'add-to-group'])
const commandMenuOpen = ref(false)

const commands = computed(() => {
  if (!props.project) return []
  if (props.project.commands && props.project.commands.length) return props.project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: props.project.command,
    args: props.project.args || [],
    cwd: props.project.cwd || '',
    isDefault: true,
  }]
})
const selectedCommand = computed(() =>
  commands.value.find((c) => c.id === props.selectedCommandId) ||
  commands.value.find((c) => c.isDefault) ||
  commands.value[0]
)
const selectedRunId = computed(() => props.project && selectedCommand.value
  ? `${props.project.id}:${selectedCommand.value.id || 'default'}`
  : ''
)
const commandLine = computed(() => selectedCommand.value
  ? [selectedCommand.value.command, ...(selectedCommand.value.args || [])].join(' ')
  : ''
)

const state = computed(() => (props.status || {}).State || 'stopped')
const pillVariant = computed(() => {
  if (state.value === 'running') return 'running'
  if (state.value === 'exited') return 'exited'
  if (state.value === 'error') return 'error'
  return 'stopped'
})

watch(() => props.selectedCommandId, () => {
  commandMenuOpen.value = false
})

function chooseCommand(command) {
  emit('command-change', command.id)
  commandMenuOpen.value = false
}
</script>

<template>
  <section class="detail" v-if="project">
    <div class="project-header">
      <div class="info">
        <div class="title-line">
          <h1>{{ project.name }}</h1>
          <AppPill :variant="pillVariant" :dot="state === 'running'">{{ state }}</AppPill>
          <AppPill variant="neutral">{{ project.detectedType || 'unknown' }}</AppPill>
        </div>
        <div class="path">{{ project.path }}</div>
        <div class="command-box">
          <span class="cmd-label">CMD</span>
          <div class="command-picker">
            <button class="command-trigger" @click="commandMenuOpen = !commandMenuOpen">
              <span>{{ selectedCommand && selectedCommand.name }}</span>
              <AppIcon :icon="commandMenuOpen ? ChevronUp : ChevronDown" :size="12" />
            </button>
            <div v-if="commandMenuOpen" class="command-menu">
              <button
                v-for="cmd in commands"
                :key="cmd.id"
                :class="{ active: selectedCommand && selectedCommand.id === cmd.id }"
                @click="chooseCommand(cmd)"
              >
                {{ cmd.name }}
              </button>
            </div>
          </div>
          <code>{{ commandLine }}</code>
        </div>
      </div>
      <div class="btns">
        <AppButton variant="secondary" @click="emit('add-to-group')">
          <template #icon><AppIcon :icon="FolderPlus" :size="14" /></template>
          Add Group
        </AppButton>
        <AppButton variant="secondary" @click="emit('edit')">
          <template #icon><AppIcon :icon="Pencil" :size="14" /></template>
          Edit
        </AppButton>
        <AppButton
          variant="danger"
          :disabled="(status || {}).State !== 'running'"
          @click="emit('stop', selectedCommand.id)"
        >
          <template #icon><AppIcon :icon="Square" :size="14" /></template>
          Stop
        </AppButton>
        <AppButton
          variant="success"
          :disabled="(status || {}).State === 'running'"
          @click="emit('start', selectedCommand.id)"
        >
          <template #icon><AppIcon :icon="Play" :size="14" /></template>
          Start
        </AppButton>
      </div>
    </div>
    <LogPanel :projectId="selectedRunId" :status="status" />
  </section>
  <section class="detail empty" v-else>
    <div>
      <h2>选择一个项目</h2>
      <p>从左侧目录树选择项目后查看命令、状态和实时日志。</p>
    </div>
  </section>
</template>

<style scoped>
.detail {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}

.detail.empty {
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  text-align: center;
}

.detail.empty h2 {
  margin: 0 0 var(--space-4);
  color: var(--text);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.015em;
}

.detail.empty p {
  margin: 0;
  font-size: var(--fs-base);
}

.project-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  gap: var(--space-8);
  padding: var(--space-7) var(--space-8);
  border-bottom: 1px solid var(--border);
  background: var(--bg);
}

.info {
  min-width: 0;
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
  min-width: 0;
}

h1 {
  max-width: min(560px, 100%);
  margin: 0;
  color: var(--text);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.015em;
  line-height: 1.15;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path {
  max-width: 100%;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-box {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  width: min(100%, 760px);
  box-sizing: border-box;
  min-width: 0;
  padding: var(--space-3) var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface);
}

.cmd-label {
  color: var(--text-subtle);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
}

.command-picker {
  position: relative;
  flex-shrink: 0;
}

.command-trigger {
  height: 24px;
  display: inline-flex;
  align-items: center;
  gap: var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated);
  color: var(--text);
  padding: 0 var(--space-3);
  font: inherit;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.command-trigger:hover { filter: brightness(1.1); }

.command-menu {
  position: absolute;
  z-index: var(--z-menu);
  top: 28px;
  left: 0;
  min-width: 150px;
  padding: var(--space-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.command-menu button {
  width: 100%;
  height: 28px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  padding: 0 var(--space-3);
  font: inherit;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.command-menu button:hover,
.command-menu button.active {
  color: var(--text);
  background: var(--elevated);
}

.command-box code {
  min-width: 0;
  color: var(--text);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btns {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--space-3);
  max-width: 420px;
}

@media (max-width: 980px) {
  .project-header {
    grid-template-columns: 1fr;
    gap: var(--space-6);
  }

  .btns {
    justify-content: flex-start;
    max-width: none;
  }
}

@media (max-width: 760px) {
  .project-header {
    padding: var(--space-6) var(--space-7);
  }

  .command-box {
    flex-wrap: wrap;
  }

  .command-box code {
    flex-basis: 100%;
  }
}
</style>
```

- [ ] **Step 2: Rewrite `frontend/src/components/GroupDetail.vue`**

Full replacement:

```vue
<script setup>
import { computed } from 'vue'
import { Play, Square, Pencil, Trash2 } from 'lucide-vue-next'
import AppButton from './ui/AppButton.vue'
import AppPill from './ui/AppPill.vue'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ group: Object, projects: Array })
const emit = defineEmits(['start', 'stop', 'edit', 'remove', 'select-command'])

function commandsFor(project) {
  if (project.commands && project.commands.length) return project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    isDefault: true,
  }]
}

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

const projectById = computed(() => {
  const out = {}
  for (const project of props.projects || []) out[project.id] = project
  return out
})

const members = computed(() => {
  const out = []
  for (const item of (props.group && props.group.items) || []) {
    const project = projectById.value[item.projectId]
    if (!project) continue
    const command = commandsFor(project).find((c) => c.id === (item.commandId || 'default'))
    if (!command) continue
    out.push({
      key: `${props.group.id}:${project.id}:${command.id || 'default'}`,
      project,
      command,
    })
  }
  return out
})
</script>

<template>
  <section class="group-detail" v-if="group">
    <header class="group-header">
      <div class="info">
        <div class="title-line">
          <h1>{{ group.name }}</h1>
          <AppPill variant="neutral">group</AppPill>
          <AppPill variant="stopped">{{ (group.items || []).length }} commands</AppPill>
        </div>
        <p>启动、停止或检查这组项目命令。</p>
      </div>
      <div class="btns">
        <AppButton variant="secondary" @click="emit('edit', group)">
          <template #icon><AppIcon :icon="Pencil" :size="14" /></template>
          Edit
        </AppButton>
        <AppButton variant="secondary" @click="emit('remove', group.id)">
          <template #icon><AppIcon :icon="Trash2" :size="14" /></template>
          Remove
        </AppButton>
        <AppButton variant="danger" @click="emit('stop', group.id)">
          <template #icon><AppIcon :icon="Square" :size="14" /></template>
          Stop
        </AppButton>
        <AppButton variant="success" @click="emit('start', group.id)">
          <template #icon><AppIcon :icon="Play" :size="14" /></template>
          Start
        </AppButton>
      </div>
    </header>

    <div class="members">
      <button
        v-for="member in members"
        :key="member.key"
        class="member-row"
        @click="emit('select-command', { projectId: member.project.id, commandId: member.command.id || 'default' })"
      >
        <span class="dot" />
        <span class="project-name">{{ member.project.name }}</span>
        <span class="command-name">{{ member.command.name }}</span>
        <code>{{ lineFor(member.command) }}</code>
      </button>
      <div v-if="members.length === 0" class="empty">这个分组还没有项目命令。</div>
    </div>
  </section>
</template>

<style scoped>
.group-detail {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}

.group-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  gap: var(--space-8);
  padding: var(--space-7) var(--space-8);
  border-bottom: 1px solid var(--border);
}

.info { min-width: 0; }

.title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.015em;
  line-height: 1.15;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

p {
  margin: var(--space-4) 0 0;
  color: var(--text-muted);
  font-size: var(--fs-base);
}

.btns {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: var(--space-3);
  max-width: 420px;
}

.members {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-7) var(--space-8);
  background: var(--surface);
}

.member-row {
  width: 100%;
  min-height: 38px;
  display: grid;
  grid-template-columns: 10px minmax(160px, 240px) minmax(90px, 140px) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-5);
  margin-bottom: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}

.member-row:hover {
  border-color: var(--border-strong);
  background: var(--elevated);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-subtle);
}

.project-name, .command-name, .member-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-name {
  color: var(--text);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.command-name {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.member-row code {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}

.empty {
  color: var(--text-muted);
  font-size: var(--fs-sm);
  padding: var(--space-8) var(--space-2);
}

@media (max-width: 980px) {
  .group-header {
    grid-template-columns: 1fr;
    gap: var(--space-6);
  }

  .btns {
    justify-content: flex-start;
    max-width: none;
  }
}
</style>
```

- [ ] **Step 3: Manual dev check**

Run: `wails dev`
Expected:
- Project detail header uses new pills (colored `running` / neutral type)
- CMD box has label `CMD`, picker chevron is lucide
- 4 buttons render with lucide icons (Add Group / Edit / Stop / Start)
- Start button success-green; Stop button soft-red
- Select a group in sidebar → Group detail renders similarly

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/ProjectDetail.vue frontend/src/components/GroupDetail.vue
git commit -m "feat(ui): migrate project + group detail panels to primitives"
```

---

## Task 7: Log panel token-ization + sticky banner

**Files:**
- Modify: `frontend/src/components/LogPanel.vue`

**Interfaces:**
- Consumes: tokens.
- Produces: no interface change.

**Behavioral guarantee:** log tail scrolls, banner reflects state, colors stay in the `stdout / stderr / running / exited-*` families.

- [ ] **Step 1: Rewrite `<style>` block in `frontend/src/components/LogPanel.vue`**

Replace the entire `<style scoped>` block with:

```css
.log-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #06070a;
}

.banner {
  height: 32px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  padding: 0 var(--space-6);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
  border-bottom: 1px solid #14161d;
  background: #0b0d13;
  position: sticky;
  top: 0;
  z-index: 1;
}

.banner.running { color: #86efac; }
.banner.exited-ok { color: #bef264; }
.banner.exited-bad,
.banner.error { color: #fca5a5; }
.banner.stopped { color: #94a3b8; }

.log-panel {
  flex: 1;
  overflow-y: auto;
  background: #06070a;
  color: #d1d5db;
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
  line-height: 1.55;
  padding: var(--space-6) var(--space-7);
  white-space: pre-wrap;
}

.log-line.stderr { color: #fca5a5; }

.empty-hint {
  color: #64748b;
  font-style: italic;
  padding: var(--space-2) 0;
}
```

Do not touch the `<script setup>` or `<template>` — behavior is unchanged.

- [ ] **Step 2: Manual dev check**

Run: `wails dev`
- Start a project (e.g. any tracked one), watch the banner say `● 运行中`, log stream in — background is deeper black, mono text unchanged.
- Stop it: banner shows `● 已退出(exit code 0)`.
- Switch app theme via topbar → log panel stays dark.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/LogPanel.vue
git commit -m "feat(ui): tokenize log panel borders + sticky banner"
```

---

## Task 8: Migrate all dialogs (4 files + `GroupTreeNode` used inside GroupDialog)

**Files:**
- Modify: `frontend/src/components/EditProjectDialog.vue`
- Modify: `frontend/src/components/ScanDialog.vue`
- Modify: `frontend/src/components/GroupDialog.vue`
- Modify: `frontend/src/components/AddProjectDialog.vue`
- Modify: `frontend/src/components/AddToGroupDialog.vue`
- Modify: `frontend/src/components/GroupTreeNode.vue`

**Interfaces:**
- Consumes: `AppButton`, `AppIcon`, tokens.
- Produces: no changes.

**Entrance/exit animation:** each dialog's `.mask` and `.dialog` use CSS transitions on `opacity` + `translateY` driven by `v-if="show"` mounting/unmounting via a `<Transition>` wrapper.

The shared pattern each dialog picks up:

- Mask: `background: var(--overlay); backdrop-filter: blur(4px);`
- Dialog: `background: var(--surface); border-radius: var(--radius-lg); box-shadow: var(--shadow-lg); border: 1px solid var(--border);`
- Inputs / textareas: `background: var(--bg); border: 1px solid var(--border-strong); color: var(--text);` and `--focus-ring` on focus
- Buttons: replace inline `<button>` with `AppButton` primitives
- Transition names: `dlg-fade` (defined below)

- [ ] **Step 1: Add shared dialog transition classes to `frontend/src/styles/tokens.css`**

Append to the bottom of `tokens.css`:

```css
.dlg-fade-enter-active { transition: opacity var(--dur-base) var(--ease); }
.dlg-fade-leave-active { transition: opacity var(--dur-fast) var(--ease); }
.dlg-fade-enter-from,
.dlg-fade-leave-to { opacity: 0; }

.dlg-pop-enter-active { transition: opacity var(--dur-base) var(--ease), transform var(--dur-base) var(--ease); }
.dlg-pop-leave-active { transition: opacity var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease); }
.dlg-pop-enter-from,
.dlg-pop-leave-to { opacity: 0; transform: translateY(4px); }
```

- [ ] **Step 2: Rewrite `frontend/src/components/EditProjectDialog.vue`**

Full replacement:

```vue
<script setup>
import { ref, watch } from 'vue'
import AppButton from './ui/AppButton.vue'

const props = defineProps({ project: Object, show: Boolean })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const commands = ref([])

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

function reset(p) {
  if (!p) return
  name.value = p.name
  const source = p.commands && p.commands.length ? p.commands : [{
    id: 'default',
    name: 'Default',
    command: p.command,
    args: p.args || [],
    cwd: p.cwd || '',
    isDefault: true,
  }]
  commands.value = source.map((c, index) => ({
    id: c.id || '',
    name: c.name || (index === 0 ? 'Default' : `Command ${index + 1}`),
    line: lineFor(c),
    cwd: c.cwd || '',
    isDefault: !!c.isDefault || index === 0,
  }))
}

watch(() => props.project, (p) => {
  reset(p)
}, { immediate: true })

function save() {
  emit('save', { name: name.value, commands: commands.value })
}

function addCommand() {
  commands.value.push({
    id: '',
    name: `Command ${commands.value.length + 1}`,
    line: '',
    cwd: '',
    isDefault: commands.value.length === 0,
  })
}

function removeCommand(index) {
  if (commands.value.length <= 1) return
  const wasDefault = commands.value[index].isDefault
  commands.value.splice(index, 1)
  if (wasDefault && commands.value.length) commands.value[0].isDefault = true
}

function setDefault(index) {
  commands.value = commands.value.map((c, i) => ({ ...c, isDefault: i === index }))
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog" v-if="show">
          <h3>编辑项目</h3>
          <label>名称<input v-model="name" /></label>
          <div class="commands-head">
            <span>启动命令</span>
            <AppButton variant="secondary" size="sm" @click="addCommand">Add command</AppButton>
          </div>
          <div class="command-list">
            <div v-for="(cmd, index) in commands" :key="cmd.id || index" class="command-row">
              <div class="command-top">
                <input v-model="cmd.name" placeholder="Name" />
                <AppButton
                  :variant="cmd.isDefault ? 'primary' : 'secondary'"
                  size="sm"
                  @click="setDefault(index)"
                >Default</AppButton>
                <AppButton
                  variant="secondary"
                  size="sm"
                  :disabled="commands.length <= 1"
                  @click="removeCommand(index)"
                >Remove</AppButton>
              </div>
              <input v-model="cmd.line" placeholder="如 pnpm run dev 或 go run main.go serve" />
              <input v-model="cmd.cwd" :placeholder="project && project.path" />
            </div>
          </div>
          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="save">保存</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(720px, calc(100vw - 36px));
  max-height: calc(100vh - 56px);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

.dialog label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.dialog input {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.dialog input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.commands-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.command-row {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.command-top {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--space-4);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  margin-top: var(--space-2);
}
</style>
```

- [ ] **Step 3: Rewrite `frontend/src/components/ScanDialog.vue`**

Full replacement:

```vue
<script setup>
import { ref } from 'vue'
import { FolderOpen } from 'lucide-vue-next'
import { ScanWorkspaces, AddScanned, PickDirectory } from '../../wailsjs/go/main/App'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ show: Boolean })
const emit = defineEmits(['close', 'added'])

const rootsText = ref('')
const candidates = ref([])
const checked = ref({})

async function scan() {
  const roots = rootsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  candidates.value = await ScanWorkspaces(roots)
  checked.value = {}
  candidates.value.forEach((c) => { checked.value[c.id] = c.detectedType !== 'unknown' })
}

async function pickDir() {
  const dir = await PickDirectory()
  if (!dir) return
  const lines = rootsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!lines.includes(dir)) lines.push(dir)
  rootsText.value = lines.join('\n')
  await scan()
}

async function add() {
  const chosen = candidates.value.filter((c) => checked.value[c.id])
  await AddScanned(chosen)
  emit('added')
  emit('close')
}

function toggle(id) {
  checked.value = { ...checked.value, [id]: !checked.value[id] }
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog" v-if="show">
          <h3>扫描工作区</h3>
          <textarea v-model="rootsText" rows="3"
            placeholder="每行一个根目录，支持 ~，如&#10;~/GolandProjects&#10;~/WebstormProjects"></textarea>
          <div class="root-actions">
            <AppButton variant="secondary" @click="pickDir">
              <template #icon><AppIcon :icon="FolderOpen" :size="14" /></template>
              选择文件夹
            </AppButton>
            <AppButton variant="primary" @click="scan">扫描</AppButton>
          </div>
          <div class="results">
            <button v-for="c in candidates" :key="c.id" :class="['row', { selected: checked[c.id] }]" @click="toggle(c.id)">
              <span class="check-mark">{{ checked[c.id] ? '✓' : '' }}</span>
              <span class="nm">{{ c.name }}</span>
              <span class="ty" :class="{ unknown: c.detectedType === 'unknown' }">{{ c.detectedType }}</span>
              <code>{{ [c.command, ...(c.args || [])].join(' ').trim() || '—' }}</code>
            </button>
          </div>
          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="add">加入选中</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(760px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

textarea {
  box-sizing: border-box;
  width: 100%;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  padding: var(--space-5);
  color: var(--text);
  background: var(--bg);
  font: inherit;
  font-size: var(--fs-sm);
  resize: vertical;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

textarea:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.root-actions {
  display: flex;
  gap: var(--space-3);
}

.root-actions > * { flex: 1; }

.results {
  max-height: 340px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.row {
  width: 100%;
  display: grid;
  grid-template-columns: 18px minmax(160px, 220px) 92px minmax(0, 1fr);
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-4) var(--space-5);
  border: 0;
  font-size: var(--fs-sm);
  font: inherit;
  text-align: left;
  background: transparent;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.row:last-child { border-bottom: 0; }
.row:hover { background: var(--elevated); }
.row.selected { background: var(--elevated); }

.check-mark {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--primary-fg);
  background: transparent;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.row.selected .check-mark {
  border-color: var(--primary);
  background: var(--primary);
}

.nm {
  color: var(--text);
  font-weight: var(--fw-semibold);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ty {
  color: var(--text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.ty.unknown { color: var(--text-subtle); }

.row code {
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
```

- [ ] **Step 4: Rewrite `frontend/src/components/GroupDialog.vue`**

Full replacement:

```vue
<script setup>
import { computed, ref, watch } from 'vue'
import { buildProjectTree } from '../projectTree'
import GroupTreeNode from './GroupTreeNode.vue'
import AppButton from './ui/AppButton.vue'

const props = defineProps({ show: Boolean, group: Object, projects: Array })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const checked = ref({})

function commandsFor(project) {
  if (project.commands && project.commands.length) return project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    isDefault: true,
  }]
}

function keyFor(projectId, commandId) {
  return `${projectId}:${commandId || 'default'}`
}

const commandOptions = computed(() => (props.projects || []).flatMap((project) =>
  commandsFor(project).map((command) => ({ project, command, key: keyFor(project.id, command.id) }))
))
const projectTree = computed(() => buildProjectTree(props.projects || [], {}, ''))

function reset() {
  name.value = props.group ? props.group.name : ''
  const next = {}
  for (const item of (props.group && props.group.items) || []) {
    next[keyFor(item.projectId, item.commandId)] = true
  }
  checked.value = next
}

watch(() => props.show, (show) => {
  if (show) reset()
})
watch(() => props.group, reset, { immediate: true })

function save() {
  const items = commandOptions.value
    .filter((option) => checked.value[option.key])
    .map((option) => ({ projectId: option.project.id, commandId: option.command.id || 'default' }))
  emit('save', {
    id: props.group ? props.group.id : '',
    name: name.value.trim() || 'New group',
    items,
  })
}

function toggleOption(key) {
  checked.value = { ...checked.value, [key]: !checked.value[key] }
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog" v-if="show">
          <h3>{{ group ? '编辑分组' : '新建分组' }}</h3>
          <label>名称<input v-model="name" placeholder="Local dev stack" /></label>

          <div class="commands-head">选择要启动的项目命令</div>
          <div class="options">
            <GroupTreeNode
              v-for="node in projectTree"
              :key="node.id"
              :node="node"
              :level="0"
              :checked="checked"
              @toggle="toggleOption"
            />
          </div>

          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="save">保存</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(820px, calc(100vw - 36px));
  max-height: calc(100vh - 56px);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

.dialog label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.dialog > label input {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.dialog > label input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.commands-head {
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.options {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
```

- [ ] **Step 5: Rewrite `frontend/src/components/AddProjectDialog.vue`**

Full replacement:

```vue
<script setup>
import { ref, watch } from 'vue'
import { FolderOpen } from 'lucide-vue-next'
import { PickDirectory } from '../../wailsjs/go/main/App'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ show: Boolean })
const emit = defineEmits(['close', 'save'])

const path = ref('')

watch(() => props.show, (show) => {
  if (show) path.value = ''
})

async function pickDir() {
  const dir = await PickDirectory()
  if (dir) path.value = dir
}

function save() {
  const value = path.value.trim()
  if (!value) return
  emit('save', value)
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog" v-if="show">
          <h3>添加项目</h3>
          <label>项目目录<input v-model="path" placeholder="/home/attson/GolandProjects/atstarter" /></label>
          <div class="inline-actions">
            <AppButton variant="secondary" @click="pickDir">
              <template #icon><AppIcon :icon="FolderOpen" :size="14" /></template>
              选择文件夹
            </AppButton>
          </div>
          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" :disabled="!path.trim()" @click="save">添加</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(520px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

input {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.inline-actions,
.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
```

- [ ] **Step 6: Rewrite `frontend/src/components/AddToGroupDialog.vue`**

Full replacement:

```vue
<script setup>
import { computed, ref, watch } from 'vue'
import AppButton from './ui/AppButton.vue'

const props = defineProps({
  show: Boolean,
  groups: Array,
  project: Object,
  command: Object,
})
const emit = defineEmits(['close', 'save'])

const mode = ref('existing')
const selectedGroupId = ref('')
const newGroupName = ref('')

const commandLine = computed(() => props.command
  ? [props.command.command, ...(props.command.args || [])].filter(Boolean).join(' ')
  : ''
)

watch(() => props.show, (show) => {
  if (!show) return
  mode.value = (props.groups || []).length ? 'existing' : 'new'
  selectedGroupId.value = (props.groups || [])[0]?.id || ''
  newGroupName.value = props.project ? `${props.project.name} group` : ''
})

function save() {
  if (mode.value === 'existing' && !selectedGroupId.value) return
  if (mode.value === 'new' && !newGroupName.value.trim()) return
  emit('save', {
    mode: mode.value,
    groupId: selectedGroupId.value,
    groupName: newGroupName.value.trim(),
  })
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog" v-if="show">
          <h3>添加到组</h3>
          <div class="target">
            <strong>{{ project && project.name }}</strong>
            <span>{{ command && command.name }}</span>
            <code>{{ commandLine }}</code>
          </div>

          <div class="mode-tabs">
            <button
              :class="['mode-tab', { active: mode === 'existing' }]"
              :disabled="!(groups || []).length"
              @click="mode = 'existing'"
            >已有组</button>
            <button
              :class="['mode-tab', { active: mode === 'new' }]"
              @click="mode = 'new'"
            >新建组</button>
          </div>

          <div v-if="mode === 'existing'" class="group-list">
            <button
              v-for="group in groups"
              :key="group.id"
              :class="['group-option', { selected: selectedGroupId === group.id }]"
              @click="selectedGroupId = group.id"
            >
              <span>{{ group.name }}</span>
              <small>{{ (group.items || []).length }} commands</small>
            </button>
          </div>
          <label v-else>组名<input v-model="newGroupName" placeholder="Local dev stack" /></label>

          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="save">添加</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(560px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0;
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

.target {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2) var(--space-5);
  padding: var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.target strong,
.target code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.target strong { color: var(--text); font-size: var(--fs-sm); font-weight: var(--fw-semibold); }

.target span {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.target code {
  grid-column: 1 / -1;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}

.mode-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.mode-tab {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--bg);
  color: var(--text-secondary);
  padding: 0 var(--space-5);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}

.mode-tab:disabled { cursor: not-allowed; opacity: .5; }

.mode-tab.active {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.group-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.group-option {
  min-height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.group-option:hover { background: var(--elevated); }

.group-option.selected {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.group-option small {
  color: var(--text-muted);
  font-weight: var(--fw-regular);
}

label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

input {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
```

- [ ] **Step 7: Update `frontend/src/components/GroupTreeNode.vue`**

Read the file first if you haven't. Replace the `<style>` block to use tokens; keep script and template unchanged. If script/template uses old triangle icons, swap them for lucide `ChevronDown`/`ChevronRight` via `AppIcon`. Concretely: read the file, then apply the following two changes:

1. If icons are `▸ ▾` characters, replace with `<AppIcon :icon="expanded ? ChevronDown : ChevronRight" :size="12" />` and add imports.
2. Replace all hex colors and pixel constants in `<style scoped>` with tokens (mirror the pattern from ProjectTreeNode / GroupTreeItem — surface, border, `--elevated` hover, `--text-*` text colors, `var(--space-*)` spacing, `var(--radius-*)` radii).

If a step-2 rewrite isn't obvious, follow the exact color-mapping from ProjectTreeNode Step 2 in Task 5: hex → token per this table:

| Old value | Token |
|---|---|
| `#f1f5f9` / `#f8fafc` (hover) | `var(--elevated)` |
| `#eef4ff` (active bg) | `var(--elevated)` |
| `#bfdbfe` (active ring) | `var(--border-strong)` |
| `#111827` | `var(--text)` |
| `#334155` / `#1f2937` | `var(--text-secondary)` |
| `#64748b` | `var(--text-muted)` |
| `#94a3b8` | `var(--text-subtle)` |
| `#e5e7eb` / `#d7dce5` / `#e2e8f0` | `var(--border)` |
| `#cbd5e1` / `#dbeafe` (accent border) | `var(--border-strong)` |
| `#2563eb` / `#1d4ed8` | `var(--text)` (used for group-name / member-command) |

- [ ] **Step 8: Manual verification**

Run: `wails dev`
Check each dialog opens with:
- Backdrop-blur behind
- Fade + subtle translateY in / out
- Inputs contrast with surface (not white-on-white)
- Buttons match `AppButton` styling
- Both themes: cycle theme via topbar toggle, reopen dialog, verify light/dark contrast

- [ ] **Step 9: Commit**

```bash
git add frontend/src/components/EditProjectDialog.vue frontend/src/components/ScanDialog.vue frontend/src/components/GroupDialog.vue frontend/src/components/AddProjectDialog.vue frontend/src/components/AddToGroupDialog.vue frontend/src/components/GroupTreeNode.vue frontend/src/styles/tokens.css
git commit -m "feat(ui): migrate all dialogs to tokens + transitions + primitives"
```

---

## Task 9: Cleanup + full theme verification + tests

**Files:**
- Modify (if grep finds hits): any stragglers
- No new files

**Goal:** the codebase has no residual hardcoded color hex or emoji, both themes look right end-to-end, and existing tests still pass.

- [ ] **Step 1: Scan for residual hardcoded hex colors in components + App.vue**

Run:
```bash
grep -RE "#[0-9a-fA-F]{6}" frontend/src/components/ frontend/src/App.vue
```

Expected: only `frontend/src/components/LogPanel.vue` lines (the intentionally hardcoded `#06070a` / `#0b0d13` / `#14161d` / `#86efac` / `#bef264` / `#fca5a5` / `#94a3b8` / `#64748b` / `#d1d5db` — log stays dark by spec).

If any other file returns hits, replace them with the appropriate tokens from Task 5's mapping table (or the danger/warning/success family) and re-run the grep.

- [ ] **Step 2: Scan for residual emoji characters**

Run:
```bash
grep -RP "[\x{1F300}-\x{1F9FF}]" frontend/src/
```

Expected: empty output. If any emoji (`📁 ▸ ▾ ▴ ● ○` etc.) remain in a component's `<template>`, replace with `<AppIcon>` + the matching lucide component. Text `▸ ▾ ▴` are treated as regular characters (not caught by the regex); grep additionally for those:

```bash
grep -RE "[▸▾▴▵●○]" frontend/src/
```

Any hits: replace with lucide equivalents (`ChevronRight` / `ChevronDown` / `ChevronUp` / dot spans).

- [ ] **Step 3: Run frontend logic tests**

```bash
node --test frontend/src/projectTree.test.mjs
node --test frontend/src/composables/useTheme.test.mjs
```

Both should PASS.

- [ ] **Step 4: Run Go backend tests (guard against accidental touch)**

```bash
go test ./...
go test -race ./internal/runner/
```

Both should PASS. If not: revert whatever backend file was touched — this plan should have modified none.

- [ ] **Step 5: Build the frontend production bundle**

```bash
cd frontend && npm run build
```

Expected: succeeds. Note bundle size in the output (should be sub-500 KB gzip; if larger, verify lucide is tree-shaking — it should be because we import named exports).

- [ ] **Step 6: End-to-end manual verification in Wails dev**

Run: `wails dev`

Perform:

1. **Fresh launch (system-follow):** if macOS is in dark mode, app opens dark; light mode → app opens light.
2. **Theme toggle:** click topbar toggle 3 times → theme cycles `system → dark → light → system`. Verify persistence: quit app (Cmd-Q), relaunch, chosen theme survives.
3. **System theme sync (in `'system'` mode):** open macOS System Settings → Appearance → toggle light/dark. App tracks.
4. **Project sidebar:** add a project (Add dialog), scan a workspace (Scan dialog), verify tree renders with lucide chevrons + dot status. Type in search — force expansion works.
5. **Project detail:** click project → header shows AppPill state pill, CMD box, 4 buttons. Start it (if runnable): dot pulses, log tails. Stop it: banner shifts to `exited`.
6. **Groups:** create a group via New Group dialog. Select group in sidebar → GroupDetail renders. Start/Stop group. Add project to group via detail's "Add Group" button.
7. **Every dialog:** open + close each of the 5 dialogs. Verify: mask-blur, fade+lift transition, dark/light contrast, buttons look consistent.
8. **Log panel:** stays dark in both themes.

- [ ] **Step 7: Commit any residual token/emoji cleanups**

```bash
git add -A frontend/src/
git commit -m "chore(ui): clean up residual hardcoded colors + emoji" \
  || echo "nothing to clean up — good"
```

- [ ] **Step 8: Final summary commit**

If any lockfile changed as a side-effect of steps 3/5:

```bash
git add frontend/package-lock.json
git commit -m "chore: sync lockfile" || true
```

Then verify tree is clean:

```bash
git status --short
```

Expected: empty (no unstaged changes).

- [ ] **Step 9: Print final diff summary for the PR / commit log**

```bash
git log --oneline main..HEAD
```

Expected output roughly:

```
<hash> chore: sync lockfile          (if any)
<hash> chore(ui): clean up residual hardcoded colors + emoji  (if any)
<hash> feat(ui): migrate all dialogs to tokens + transitions + primitives
<hash> feat(ui): tokenize log panel borders + sticky banner
<hash> feat(ui): migrate project + group detail panels to primitives
<hash> feat(ui): migrate project sidebar to tokens + lucide + tighter density
<hash> feat(ui): migrate topbar to tokens + primitives + theme toggle
<hash> feat(ui): install lucide + primitives (AppButton, AppPill, AppIcon, ThemeToggle)
<hash> feat(ui): add useTheme composable with system/dark/light cycling
<hash> feat(ui): add design tokens + dark/light theme foundations
<hash> docs: add UI modernization design spec
```

---

## Self-Review

**Spec coverage:**
- §3 (tokens) → Task 1
- §4 (typography) → Task 1 (font stacks + `--fs-*` tokens)
- §5 (primitives) → Task 3
- §6 (icons) → Task 3 (dep install) + used throughout Tasks 4–8
- §7 (theme system) → Task 2
- §8 (density) → Tasks 4–8 (numbers reflect spec: topbar 48, sidebar 300, project row 28, header padding 16/20, command box 36-ish via new smaller trigger)
- §9 (motion) → Task 1 keyframes + Task 8 dialog transitions + Task 5 pulse binding
- §10 (component migration) → covered by Tasks 4–8 (12 files touched)
- §11 (dependencies) → Task 3 add lucide + Task 1 removes Nunito
- §12 (rollout) → Tasks 1–9 map 1:1
- §13 (verification) → Task 9

**Placeholder scan:** no "TBD" / "similar to Task N" / "handle edge cases" — every step has concrete code or a concrete command.

**Type consistency:**
- `useTheme` exposes `theme` / `resolvedTheme` / `cycleTheme` / `init` — used identically in `main.js` (Task 2) and `ThemeToggle.vue` (Task 3).
- `AppButton` props (`variant`, `size`, `disabled`, `iconOnly`) match every call site in Tasks 4–8.
- `AppPill` variants (`running / exited / error / stopped / neutral`) match Task 6's `pillVariant` computed and the direct usages.
- `AppIcon` `icon` prop always receives a lucide component; `size` always a number.

No gaps found.
