// Short display label for a project's detected type.
// The backend stores fine-grained values like "node-pnpm" or "java-maven";
// the UI just shows the tool part ("pnpm", "maven").
export function typeLabel(type) {
  if (!type) return 'unknown'
  return type.replace(/^(node|java)-/, '')
}
