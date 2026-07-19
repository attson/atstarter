// Short display label for a project's detected type.
// The backend stores fine-grained values like "node-pnpm"; the UI just shows "pnpm".
export function typeLabel(type) {
  if (!type) return 'unknown'
  return type.replace(/^node-/, '')
}
