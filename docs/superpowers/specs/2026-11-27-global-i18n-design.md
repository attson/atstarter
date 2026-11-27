# Rootyard Global Chinese and English i18n Design

**Date:** 2026-11-27

**Status:** Approved for implementation planning

## 1. Summary

Rootyard will support application-wide Simplified Chinese and English through `vue-i18n`. The first
launch follows the operating-system language. Users can choose Follow system, Simplified Chinese, or
English in the existing global Settings dialog; a saved choice applies immediately across every
mounted view and persists in `config.json.settings`.

The work covers all Rootyard-owned frontend copy: navigation, projects, groups, containers,
environments, file browsing and editing, logs, updates, dialogs, confirmations, empty states, status
labels, placeholders, tooltips, titles, and accessibility labels. Project data, paths, commands,
branches, container names, native configuration keys, and raw Git/Docker/Shell/filesystem errors are
not translated.

## 2. Goals

- Provide complete `zh-CN` and `en` message catalogs with identical key sets.
- Follow the system language on first launch and whenever the saved preference is `system`.
- Let users switch language from Settings without restarting or reloading.
- Persist the preference through the existing App Settings aggregate without breaking old configs.
- Avoid a visible wrong-language flash during startup.
- Update already-mounted components, open dialogs, dynamic labels, tooltips, and accessibility text.
- Preserve raw third-party diagnostic output exactly as received.
- Keep every existing workflow and top-level navigation geometry unchanged.

## 3. Non-goals

- Supporting locales other than `zh-CN` and `en` in this phase.
- Translating project names, group names, commands, paths, branch names, container names, logs, or
  native Git/Codex configuration values.
- Machine-translating backend or third-party error strings.
- Localizing dates, numbers, command output, or release notes beyond existing browser behavior.
- Adding a top-bar language control, a new settings page, or per-project language preferences.
- Splitting the application into locale-specific bundles or build artifacts.

## 4. Confirmed Product Behavior

### 4.1 Language preference

The persisted preference uses exactly these values:

| Value | Behavior |
|-------|----------|
| `system` | Resolve from `navigator.language`; any `zh` locale maps to `zh-CN`, all others map to `en` |
| `zh-CN` | Always render Simplified Chinese |
| `en` | Always render English |

An absent, blank, or unknown value normalizes to `system`. Existing `config.json` files remain valid
without a schema-version change. The Settings dialog presents the three choices in its Language
field. Saving applies the new effective locale immediately; canceling leaves both preference and
locale unchanged.

When `system` is active, Rootyard listens to the browser `languagechange` event where supported and
re-evaluates the effective locale. When an explicit locale is active, system changes have no effect.

### 4.2 Startup and cache

`config.json.settings.language` is authoritative. A small localStorage cache contains the last saved
preference so `main.js` can select the initial locale before Vue mounts. After `GetAppSettings`
returns, the backend value replaces the cached value and becomes authoritative. Every successful
settings save refreshes the cache. Invalid or unavailable storage falls back safely to `system`.

The cache key belongs to the Rootyard namespace and stores only the language-preference enum. It does
not duplicate the rest of App Settings.

### 4.3 Translation boundary

Rootyard-owned presentation copy is translated at render time. Stable domain values remain stable:

- helpers return status codes, kinds, or structured data rather than translated strings;
- components translate those values through message keys;
- confirmation state stores an action kind and payload, not a pretranslated title or message;
- server and native-tool errors are displayed verbatim, optionally preceded by a translated
  Rootyard-owned label when context is needed;
- brands and proper nouns such as Rootyard, Git, Codex, Docker, Wails, and shell names remain
  unchanged.

Changing locale therefore updates already-open views and dialogs instead of waiting for each workflow
to reopen.

## 5. Architecture

### 5.1 Frontend i18n module

Add a focused `frontend/src/i18n/` module:

```text
frontend/src/i18n/
  index.js        vue-i18n creation, exported locale controls, document language synchronization
  locale.js       pure preference normalization, system resolution, and cache helpers
  messages/
    en.js          English source catalog
    zh-CN.js       Simplified Chinese catalog
```

Use `vue-i18n@9`, which supports the repository's Vue 3.2 runtime. `main.js` creates one i18n instance
and registers it before mounting `App`. Composition API components use `useI18n`; templates may use
`$t` where no script access is required.

The i18n module exposes separate concepts:

- preference: `system | zh-CN | en`;
- effective locale: `zh-CN | en`;
- `setLanguagePreference(value)`: normalize, cache, update effective locale, and update
  `document.documentElement.lang`;
- `refreshSystemLocale()`: update only when the preference is `system`.

English is the runtime fallback locale. Missing message keys must never be rendered to users.

### 5.2 App Settings persistence

Add `Language string` to both `store.AppSettings` and the Wails-facing `main.AppSettings`. Backend
normalization accepts only `system`, `zh-CN`, or `en`; the zero value becomes `system`. Save and load
round-trip the field with shell and log typography settings. Regenerate Wails bindings after the
exported shape changes.

`frontend/src/appSettings.js` gains the normalized `language` property. `App.vue` applies the loaded
preference after `GetAppSettings` and applies a saved value only after `SaveAppSettings` succeeds.
`SettingsDialog.vue` edits language in the same draft object as the other settings.

### 5.3 Message catalog organization

Keys are semantic and grouped by feature rather than component filename. Top-level namespaces include
`common`, `navigation`, `settings`, `projects`, `groups`, `containers`, `environments`, `files`,
`editor`, `logs`, `updates`, `dialogs`, `status`, and `accessibility`.

Messages use interpolation for names, counts, paths, and actions. Pluralization uses vue-i18n choice
syntax where English grammar requires it. Catalog values contain presentation copy only; code and
domain data never become keys.

The English and Chinese catalogs must have identical recursive key sets. A Node test fails on missing,
extra, empty, or non-string leaf values.

### 5.4 Component migration

Migrate all user-facing strings under `frontend/src`, including the existing mixture of Chinese and
English. This includes:

- `App.vue` top-level navigation, context menus, confirmation flows, and status summaries;
- every component under `components/`, `components/environments/`, and `components/fileExplorer/`;
- UI primitives whose visible or accessible labels are owned by Rootyard;
- helper-generated presentation labels in modules such as project detection, Git branches, container
  status, and environment summaries.

Where a helper currently returns a display string, prefer returning a stable code and translating in
the consuming component. A helper may accept a translator only when returning structured display copy
is its sole responsibility and doing so avoids duplicating domain logic.

No layout redesign accompanies migration. Existing stable dimensions, truncation, tooltips, and
responsive behavior remain authoritative. English expansion must wrap or truncate intentionally
without shifting fixed controls.

## 6. Error Handling

- Invalid persisted or cached language values normalize to `system` without surfacing an error.
- localStorage read/write failures are ignored and do not block startup or saving backend settings.
- If the backend settings load fails, the synchronously cached preference remains active and the
  existing raw error behavior is preserved.
- If a translation key is missing in development or tests, validation fails. Runtime falls back to
  English and never exposes an internal key as intended copy.
- Native and backend error details remain unmodified. Translation applies only to Rootyard-owned
  surrounding labels or actions.

## 7. Accessibility and Document Metadata

Every translated visible command retains an equivalent localized accessible name. Icon-only controls
localize their `title` and `aria-label`; form controls localize labels and placeholders independently.
The document root `lang` attribute always matches the effective locale. Keyboard behavior, focus
order, roles, and dialog associations do not change.

## 8. Testing and Verification

Backend tests cover:

- absent language migration to `system`;
- accepted enum values and rejection/normalization of unknown values;
- App Settings save/load round trips without regressing shell or typography fields;
- generated Wails model consistency.

Frontend tests cover:

- preference normalization and system-language resolution;
- cache read/write failure behavior;
- immediate locale and `<html lang>` updates;
- system-mode refresh and explicit-locale stability;
- backend-loaded settings overriding the startup cache;
- exact recursive key parity and non-empty message leaves;
- Settings draft/save/cancel behavior;
- representative dynamic interpolation, pluralization, dialogs, status labels, tooltips, and
  accessibility labels;
- a source-level audit that identifies remaining user-facing hardcoded copy, with explicit allowlists
  only for brands, domain values, and test fixtures.

Completion checks include the full Go suite, project-environment race suite, all frontend Node tests,
the Vite production build, and Windows compilation for changed Go models. Live browser verification
must exercise both languages in light and dark themes at 1024x768 and 1440x900. It must verify the
settings switch, immediate whole-app updates, reload persistence, open-dialog reactivity, no horizontal
page scroll, no clipped controls, and no console or asset failures.

## 9. Documentation and Rollout

Update `docs/specs/DOMAIN_MODEL.md` to include language in App Settings and
`docs/specs/FRONTEND_STYLE.md` with the i18n boundary and translation rules. Update README user-facing
feature descriptions where settings or screenshots imply a single language.

The change is backward compatible and ships in the existing feature branch and PR #74. No native
project configuration is touched by locale changes. The worktree remains available for PR feedback.
