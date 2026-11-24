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
| App settings | `SettingsDialog.vue` | Default shell and live log typography preferences |
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

### 3.2 Project Navigation View Pattern

Applicable to `ProjectList.vue`, `ProjectTreeNode.vue`, and `GroupTreeItem.vue`.

1. Project navigation offers `list` and `card` modes through an icon segmented control. The default is
   `list`; the current value is stored by the frontend in `localStorage` under
   `projectList.viewMode`. Missing, inaccessible, or unknown values normalize to `list`.
2. List mode keeps the compact existing rows and does not read Git branch metadata. Card mode uses the
   shared `ProjectCardContent.vue` for pinned, regular tree, worktree, and group-member project nodes;
   group members also retain their selected Launch Command name.
3. Right-clicking the card-view icon opens the shared checkable field menu without changing the current
   view. Project name, run status, missing state, and child count stay visible; detected type, branch,
   shortened path, and Launch Command are independently optional. The menu stays open while toggling and
   closes on outside click, `Escape`, or a view switch.
4. Optional card fields are stored as booleans under `projectList.cardFields`; all four default to visible.
   Missing or invalid fields independently normalize to visible, unknown fields are ignored, and storage
   failures keep the current in-memory choice without blocking navigation. One preference applies to
   pinned, regular tree, worktree, and group-member cards; a card without a Launch Command name renders no
   empty command row.
5. Card mode reads branches through `GetProjectBranch(project.path)` only while the branch field is
   visible, caches display state by Project ID, and limits reads to four concurrent calls. Refresh on
   entering card mode, when the project collection changes, and when the branch field is re-enabled.
   Hiding the field stops queued reads from starting and ignores results from reads already in flight. Do
   not poll or perform network Git operations. Empty branch values display `无分支信息`; rejected reads
   display `分支读取失败`, and neither state blocks navigation.
6. A successful detail-view branch switch updates the matching card immediately. The event must carry
   the Project ID captured when the switch started, and an older in-flight branch read must not
   overwrite the switched value.
7. Project name, branch, path, and group command remain single-line and ellipsized in the 300px sidebar;
   each full value is available through `title`, and cards must not introduce horizontal overflow.

Reference implementation: `frontend/src/projectNavigation.js`,
`frontend/src/components/ProjectCardContent.vue`.

### 3.3 Project Detail Header Pattern

Applicable to `ProjectDetail.vue`.

1. Title line contains project name, status pill, type/switch pill, branch pill, missing pill.
2. Project path is a full-width row with monospaced text.
3. Path stays single-line with ellipsis and uses `title` so hover shows the full path.
4. Command row contains command picker, command line, and the `Edit` action.
5. Start/stop/restart controls are grouped as icon buttons.

The `Edit` action belongs on the command row because it edits command-related fields: command name,
line, cwd, and env.

### 3.4 Command Form Pattern

Applicable to `EditProjectDialog.vue` and add/edit command flows. One command renders as one
`CommandRow.vue`; the dialog owns the list (add/remove/default), the row owns a single command.

1. Build forms through `commandFormsForProject(project)`.
2. If command `cwd` is empty, display `project.path` as the input value so users can edit directly.
3. Field order inside a row is fixed: name + default/remove actions, then `cwd`, then command line,
   then env. `cwd` sits above the command line because it scopes the command.
4. The `cwd` input carries a trailing inline folder-icon button that calls `PickDirectoryFrom` with
   the current effective cwd, so the native picker opens where the user already is.
5. Round-trip env as one `KEY=value` per line.
6. On save, parse env text through `envTextToMap`.
7. Keep exactly one default command selected in the form.

Reference implementation: `frontend/src/commandForms.js`, `frontend/src/envVars.js`,
`frontend/src/components/CommandRow.vue`.

### 3.5 Script Completion Pattern

Applicable to the command-line input in `CommandRow.vue`.

1. Completion triggers only on `<pm> run <prefix>` where `<pm>` is `npm`, `pnpm`, `yarn`, or `bun`,
   and nothing follows the prefix. Bare `pnpm dev` is out of scope: separating scripts from a package
   manager's own subcommands (`install`, `add`, `build`) is ambiguous and mis-completes.
2. Candidates come from `ListPackageScripts(effectiveCwd)`, where `effectiveCwd` is the row's `cwd`
   falling back to `project.path`. Results are cached per directory for the life of the dialog.
3. Matching is case-insensitive; prefix hits sort before substring hits, each group keeping the
   backend's alphabetical order.
4. The overlay is a custom floating list rendered with `components/ui` tokens, not a native
   `<datalist>`, so it follows the theme and can show each script's actual command.
5. Keys: `ArrowUp`/`ArrowDown` move, `Enter`/`Tab` accept, `Escape` closes without bubbling to the
   dialog. Options use `@mousedown.prevent` so clicking does not blur the input.
6. Failure to read scripts closes the overlay silently. Completion never blocks or errors typing.

Reference implementation: `frontend/src/scriptComplete.js`.

### 3.6 Env Text Pattern

`envTextToMap` behavior is part of the UI contract:

- Blank lines are ignored.
- Lines without `=` are ignored.
- Empty keys are ignored.
- Keys and values are trimmed.
- Later duplicate keys overwrite earlier keys.
- Serialization sorts keys for stable display.

### 3.7 App Settings Pattern

1. The top bar gear opens `SettingsDialog.vue`; settings remain a modal tool rather than a new app view.
2. Default shell offers automatic account-shell discovery, discovered executable shells, and an
   explicit custom absolute path.
3. Log typography offers font family, font size, and non-negative character spacing with a live
   terminal preview.
4. Saved log typography applies to every mounted `LogPanel` immediately. Shell changes apply only to
   subsequently started processes.
5. Settings persist through `GetAppSettings`/`SaveAppSettings` in `config.json.settings`; frontend
   normalization lives in `appSettings.js` and has `node:test` coverage.

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
- [ ] Declare `box-sizing: border-box` on any element that combines `width: 100%` with padding or a
      border. There is no global reset in `style.css`, so the default `content-box` silently widens
      the element past its container.

### Conditional

- [ ] If adding a Wails method, regenerate bindings and update imports.
- [ ] If adding a new route/view, ensure Projects/Containers shell behavior stays predictable.
- [ ] If adding file preview/edit UI, follow `FILE_BROWSER.md`.
- [ ] If the change affects the public home demo, follow `SITE.md`.

### Verification

- [ ] Run relevant `node --test frontend/src/*.test.mjs frontend/src/composables/*.test.mjs`.
- [ ] Run `cd frontend && npm run build` for UI or generated binding changes.
- [ ] Check layout against the app's own stylesheet, not only the VitePress home demo. VitePress
      applies a global `border-box` reset that the app does not, so box-model bugs can look fine in
      the demo and break in the app.
