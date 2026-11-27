// Exceptions must be exact { file, text, reason } strings. Use source-root-relative
// files (the same spelling as AUDITED_UI_FILES), and explain the concrete reason,
// e.g. "Rootyard product name", "native config key", or "shell path example".
// Do not use regular expressions, globs, substring matching, or blanket reasons.
export const SOURCE_AUDIT_ALLOWLIST = [
  { file: 'App.vue', text: 'Rootyard', reason: 'Rootyard is the product display name and remains unchanged in every locale.' },
  { file: 'components/AddProjectDialog.vue', text: '/home/attson/GolandProjects/rootyard', reason: 'This literal example is a filesystem path, which remains raw in every locale.' },
  { file: 'components/GroupTreeNode.vue', text: '✓', reason: 'This check mark is the existing selected-command indicator, independent of locale.' },
  { file: 'components/BranchSwitcher.vue', text: '●', reason: 'This dot is the existing dirty-worktree indicator, independent of locale.' },
  { file: 'components/ScanDialog.vue', text: '✓', reason: 'This check mark is the existing selected-candidate indicator, independent of locale.' },
  { file: 'components/ScanDialog.vue', text: '—', reason: 'This em dash indicates an absent detected command without introducing presentation words.' },
  { file: 'components/ComposeDetail.vue', text: ',', reason: 'This comma separates raw Docker port mappings in their existing order; it contains no presentation words.' },
]
