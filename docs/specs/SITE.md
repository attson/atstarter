# Site Spec

## Document Goal

This file describes the VitePress documentation site and product home page. It covers the public
marketing/docs surface, the embedded real-app demo, mock data boundaries, theme bridging, screenshot
assets, and Pages deployment checks.

## 1. Site Responsibilities

The site lives under `site/docs` and is published to GitHub Pages at `/atstarter/`.

| Area | File | Responsibility |
|------|------|----------------|
| VitePress config | `site/docs/.vitepress/config.mjs` | Base path, metadata, nav, sidebar, module aliases |
| Theme entry | `site/docs/.vitepress/theme/index.js` | VitePress theme extension, home class, reveal behavior, theme bridge |
| Site CSS | `site/docs/.vitepress/theme/custom.css` | VitePress page styling and home layout |
| Home content | `site/docs/index.md` | Public home page content and screenshots |
| Guide content | `site/docs/guide/*.md` | User documentation and troubleshooting |
| Home app demo | `site/docs/.vitepress/theme/components/HomeDemo.vue` | Embedded product UI |
| Mock Wails API | `site/docs/.vitepress/theme/components/mockWailsApp.mjs` | Browser-safe fake App bindings |
| Mock runtime | `site/docs/.vitepress/theme/components/mockWailsRuntime.mjs` | Browser-safe fake Wails runtime events |
| Public assets | `site/docs/public/*` | Favicon and screenshot resources |

The site must explain the product without becoming a separate product implementation. When the home
page needs an interactive launcher, it embeds the real Vue app instead of duplicating UI.

## 2. Home Demo Contract

The home page demo uses `frontend/src/App.vue` in embedded mode:

```vue
<FrontendApp embedded />
```

### Required

- Import the real frontend app and global frontend styles from `frontend/src`.
- Route Wails imports through Vite aliases to mock modules under `site/docs/.vitepress/theme/components`.
- Keep demo data synthetic. Paths, project names, container names, logs, release URLs, and file contents
  must not contain real user or repository-private data.
- Seed enough state to demonstrate running projects, stopped/exited projects, groups, logs, Docker state,
  branches, and file preview/edit behavior.
- Keep the demo usable as the first home-page viewport, not just as a screenshot.

### Forbidden

- Do not hand-build an approximate copy of the app UI for the home demo.
- Do not introduce site-only UI components to replace existing product components.
- Do not read local files or call real backend APIs from the published site.
- Do not add explanatory copy inside the embedded demo surface.

## 3. Mock Wails Contract

`mockWailsApp.mjs` mirrors the subset of generated Wails bindings that the real frontend imports.

### Behavior Rules

- Return cloned data from list/read methods so UI actions cannot mutate fixtures by accident.
- Emit `status:<runID>` and `log:<runID>` through `mockWailsRuntime.mjs` when start/stop actions run.
- Preserve Go-shaped polling status keys such as `State`, `PID`, and `ExitCode`, and lower-case event
  payload keys such as `state`, `pid`, and `exitCode`.
- Keep file APIs project-scoped and synthetic. Mock file paths should look realistic but remain fake.
- Keep git branch APIs synthetic and in-memory. The demo must never run a real `git` command.
- Keep release/update data synthetic and non-actionable.
- `mockWailsCoverage.test.mjs` enforces "every binding the frontend imports is exported by the mock".
  It scans `.vue`, `.js`, `.mjs`, and `.ts` under `frontend/src` — `.ts` matters because `fsBridge.ts`
  is where most file bindings are imported.

### Fixture Rules

- Use invented product/project names such as demo workspaces or sample services.
- Use generic local paths under `/Users/demo/...`.
- Never copy real customer, personal, workspace, token, host, or private repository data into fixtures.

## 4. Theme And Asset Contract

The desktop app owns product theme tokens in:

- `frontend/src/styles/tokens.css`
- `frontend/src/styles/theme.light.css`
- `frontend/src/styles/theme.dark.css`

The site imports these tokens for the embedded app and uses VitePress theme state as the source of
truth for the page theme. The bridge in `site/docs/.vitepress/theme/index.js` must keep
`html[data-theme]` synchronized with VitePress `html.dark`, including after route changes and after the
embedded frontend writes `data-theme`.

### Asset Rules

- The site favicon is `site/docs/public/atstarter-icon.svg` and must be referenced with the Pages base:
  `/atstarter/atstarter-icon.svg`.
- Screenshots in `site/docs/public/shot-*.png` should show current product UI, not outdated mockups.
- Screenshot names should be stable when content is refreshed so Markdown references do not churn.
- Public images must not reveal private project paths, tokens, logs, hostnames, or user data.

## 5. Build And Deploy Contract

The site package uses:

```bash
cd site
npm run docs:dev
npm run docs:build
npm run docs:preview
```

Because `HomeDemo.vue` imports frontend source files, Pages CI must install both `frontend` and `site`
dependencies. Vite aliases must resolve:

- Vue imports to the site dependency tree.
- CodeMirror packages to the frontend dependency tree.
- Wails App/runtime imports to mock modules.

## 6. Allowed And Forbidden

### Allowed Changes

- Add public guide pages under `site/docs/guide/`.
- Refresh screenshot assets when the app UI changes.
- Extend mock bindings when the embedded app imports new Wails methods.
- Add source-level tests that assert the site continues to embed the real app and use mock Wails modules.

### Forbidden Changes

- Do not make the site depend on a running local backend.
- Do not push real local config or user data into mock fixtures.
- Do not hardcode a VitePress base other than `/atstarter/` for GitHub Pages.
- Do not remove the route-aware theme bridge while the embedded frontend can write `html[data-theme]`.
- Do not let CodeMirror or Vue resolve to duplicate incompatible package instances in the site build.

## 7. New Implementation Checklist

### Required

- [ ] Keep `HomeDemo.vue` importing `frontend/src/App.vue`.
- [ ] Keep Wails App/runtime imports aliased to site mock modules.
- [ ] Keep mock data synthetic.
- [ ] Keep `html[data-theme]` synchronized with VitePress dark/light state.
- [ ] Keep public assets free of sensitive information.

### Conditional

- [ ] If adding frontend imports to the embedded demo, ensure Pages installs and aliases needed packages.
- [ ] If changing theme behavior, test route changes between `/` and `/guide/`.
- [ ] If refreshing screenshots, verify all referenced image paths still exist.
- [ ] If changing mock Wails behavior, update `mockWails.test.mjs`.

### Verification

- [ ] `cd site && node --test docs/.vitepress/theme/components/homeDemoSource.test.mjs`
- [ ] `cd site && node --test docs/.vitepress/theme/components/mockWails.test.mjs`
- [ ] `cd site && node --test docs/.vitepress/theme/components/homeDemoBundle.test.mjs`
- [ ] `cd site && npm run docs:build`
