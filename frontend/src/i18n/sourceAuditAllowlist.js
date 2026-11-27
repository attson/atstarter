// Exceptions must be exact { file, text, reason } strings. Use source-root-relative
// files (the same spelling as AUDITED_UI_FILES), and explain the concrete reason,
// e.g. "Rootyard product name", "native config key", or "shell path example".
// Do not use regular expressions, globs, substring matching, or blanket reasons.
export const SOURCE_AUDIT_ALLOWLIST = [
  { file: 'App.vue', text: 'Rootyard', reason: 'Rootyard is the product display name and remains unchanged in every locale.' },
  { file: 'components/AddProjectDialog.vue', text: '/home/attson/GolandProjects/rootyard', reason: 'This literal example is a filesystem path, which remains raw in every locale.' },
  { file: 'components/fileExplorer/EditorStatusBar.vue', text: 'UTF-8', reason: 'UTF-8 is the native document encoding identifier and remains unchanged in every locale.' },
  { file: 'components/ProjectDetail.vue', text: 'running', reason: 'Native RunStatus.State enum value checked before translating the corresponding status label.' },
  { file: 'components/ProjectDetail.vue', text: 'exited', reason: 'Native RunStatus.State enum value checked before translating the corresponding status label.' },
  { file: 'components/ProjectDetail.vue', text: 'error', reason: 'Native RunStatus.State enum value checked before translating the corresponding status label.' },
  { file: 'components/ProjectDetail.vue', text: 'stopped', reason: 'Native RunStatus.State enum value checked before translating the corresponding status label.' },
  { file: 'projectEnvironmentUi.js', text: 'description', reason: 'Native environment-preset config field key used to read the original user-owned description.' },
  { file: 'projectEnvironmentUi.js', text: 'Description', reason: 'Go-shaped native environment-preset config field key used to read the original user-owned description.' },
]
