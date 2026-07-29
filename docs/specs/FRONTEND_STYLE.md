# Frontend Style Spec

## Document Goal

This file constrains frontend structure, visual language, and interaction behavior for the Vue desktop
app. It is intentionally operational, not marketing-oriented.

## 1. Core Interfaces

```js
import { ListProjects, StartProjectCommand } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
```

```js
commandFormsForProject(project)
envTextToMap(text)
envMapToText(env)
```

### Interface Constraints

- Vue components call Wails bindings, not Go internals or local config files.
- Pure frontend state transformations belong in small modules such as `commandForms.js`,
  `envVars.js`, `dockerState.js`, and `projectTree.js` with `node:test` coverage.
- Event payloads from Wails use lower-case keys; frontend polling results may use Go-shaped keys.

## 2. Layout And Dispatch Rules

| Area | Component | Rule |
|------|-----------|------|
| App shell | `App.vue` | Top-level app state and Projects/Containers view switching |
| Project tree | `ProjectList`, tree node components | Navigation, scan/add/group entry points |
| Project detail | `ProjectDetail.vue` | Header, command row, logs/files tabs |
| Compose detail | `ComposeDetail.vue` | Compose lifecycle and service logs |
| Containers | `ContainerPanel.vue` | Host container list and lifecycle |
| File browser | `FileBrowser.vue`, `fileExplorer/*` | Project-scoped file operations |
| Dialogs | `*Dialog.vue` | Modal editing/confirmation flows |
| UI primitives | `components/ui/*` | Buttons, icons, pills, theme toggle |

### Layout Constraints

- Do not introduce a third-party UI component library.
- Do not create landing-page or marketing layouts. The first screen is the usable launcher.
- Keep operational density compact: headers, command rows, tabs, and controls should use stable heights
  and tokenized spacing.
- Do not nest cards inside cards. Use framed surfaces only for dialogs, repeated items, and true tools.

## 3. Implementation Patterns

### 3.1 Theme Token Pattern

Applicable to all styled components and to the embedded site demo.

1. Define shared values in `frontend/src/styles/tokens.css`.
2. Override semantic theme variables in `theme.light.css` and `theme.dark.css`.
3. In components, reference CSS variables from scoped styles.
4. Use `lucide-vue-next` icons inside controls when an icon exists.

Reference implementation: `frontend/src/styles/*`, `components/ui/*`.

The VitePress home demo imports the same tokens and theme files. Its page-level `html[data-theme]`
bridge is specified in `SITE.md`; frontend components should not special-case VitePress.

### 3.2 Project Detail Header Pattern

Applicable to `ProjectDetail.vue`.

1. Title line contains project name, status pill, type/switch pill, branch pill, missing pill.
2. Project path is a full-width row with monospaced text.
3. Path stays single-line with ellipsis and uses `title` so hover shows the full path.
4. Command row contains command picker, command line, and the `Edit` action.
5. Start/stop/restart controls are grouped as icon buttons.

The `Edit` action belongs on the command row because it edits command-related fields: command name,
line, cwd, and env.

### 3.3 Command Form Pattern

Applicable to `EditProjectDialog.vue` and add/edit command flows.

1. Build forms through `commandFormsForProject(project)`.
2. If command `cwd` is empty, display `project.path` as the input value so users can edit directly.
3. Round-trip env as one `KEY=value` per line.
4. On save, parse env text through `envTextToMap`.
5. Keep exactly one default command selected in the form.

Reference implementation: `frontend/src/commandForms.js`, `frontend/src/envVars.js`.

### 3.4 Env Text Pattern

`envTextToMap` behavior is part of the UI contract:

- Blank lines are ignored.
- Lines without `=` are ignored.
- Empty keys are ignored.
- Keys and values are trimmed.
- Later duplicate keys overwrite earlier keys.
- Serialization sorts keys for stable display.

## 4. Allowed And Forbidden

### Allowed Changes

- Add small UI primitives under `components/ui/` when repeated patterns justify them.
- Add scoped CSS in component files when it references existing tokens.
- Add pure frontend helper modules with `node:test` coverage.
- Use browser-native controls when they fit the workflow better than custom controls.

### Forbidden Changes

- Do not hardcode one-off colors, radii, shadows, or font sizes when a token exists.
- Do not use viewport-width font scaling.
- Do not use visible instructional copy to explain basic controls inside the app surface.
- Do not let long paths, commands, project names, or button text overflow their containers.
- Do not move command editing away from the command row without updating this spec and tests.
- Do not add decorative gradient/orb backgrounds or marketing hero sections.

## 5. New Implementation Checklist

### Required

- [ ] Use Composition API and existing component patterns.
- [ ] Use CSS variables for colors, spacing, radii, typography, shadows, transitions.
- [ ] Keep text truncation/tooltip behavior for long paths and commands.
- [ ] Keep icon-only run controls titled for hover/accessibility.
- [ ] Add pure helper tests when state transformation logic changes.

### Conditional

- [ ] If adding a Wails method, regenerate bindings and update imports.
- [ ] If adding a new route/view, ensure Projects/Containers shell behavior stays predictable.
- [ ] If adding file preview/edit UI, follow `FILE_BROWSER.md`.
- [ ] If the change affects the public home demo, follow `SITE.md`.

### Verification

- [ ] Run relevant `node --test frontend/src/*.test.mjs frontend/src/composables/*.test.mjs`.
- [ ] Run `cd frontend && npm run build` for UI or generated binding changes.
