# File Browser Spec

## Document Goal

This file describes the project-scoped file browser/editor. Its main constraints are path safety,
bounded reads/writes, predictable previews, and clean watcher lifecycle.

## 1. Core Interfaces

```go
ListProjectDir(projectID, relPath string) ([]filetree.Entry, error)
SearchProjectFiles(projectID, query string, limit int) (filetree.SearchResults, error)
ProjectFileMeta(projectID, relPath string) (filetree.FileMetaInfo, error)
ReadProjectFileBytes(projectID, relPath string, maxBytes int64) (filetree.FileBytes, error)
WriteProjectFileBytes(projectID, relPath string, data []byte, expectedModTime int64, createIfMissing bool) (int64, error)
ProjectAssetURL(projectID, relPath string) string
OpenProjectPath(projectID, relPath string) error
WatchProjectDir(projectID, relPath string) (int64, error)
UnwatchProjectDir(id int64) error
```

Legacy text helpers still exist:

```go
ReadProjectFile(projectID, relPath string) (filetree.FileContent, error)
WriteProjectFile(projectID, relPath, content string) error
CreateProjectFile(projectID, relPath string) error
MkdirProject(projectID, relPath string) error
RenameProject(projectID, from, to string) error
RemoveProjectPath(projectID, relPath string, recursive bool) error
TrashProjectPath(projectID, relPath string) error
```

### Interface Constraints

- Every backend operation starts from a project ID, resolves the stored project path, then resolves
  `relPath` under that root.
- `relPath` is project-relative. Frontend must not send absolute paths as a way to escape the project.
- Bytes APIs are preferred for editor/preview because they carry mod-time conflict data.

## 2. Dispatch Rules

| Key/Condition | Implementation | Description |
|---------------|----------------|-------------|
| Empty `relPath` | project root | Root directory listing/watch |
| Directory listing | `filetree.ListDir` | Direct children only, directories first, name ascending |
| Full walk | `filetree.WalkPaths` | Slash-separated paths for tree inputs, capped at 50,000 entries |
| Filename search | `filetree.Search` / `SearchPaths` | Case-insensitive file/dir name and relative-path search, capped by request |
| Preview bytes | `ReadFileBytes` | Capped by request or 16 MB hard limit |
| Text preview | `ReadFile` | Capped at 4 MB and rejects binary content |
| Write bytes | temp file + fsync + rename | Uses optional mod-time conflict check |
| Trash | OS trash through `trash-go` | Falls back through `ErrTrashUnavailable` handling |
| Watch | numeric watch handle | Emits changed relative directory |

### Dispatch Constraints

- Directories and files are handled separately; writing a directory is an error.
- Project-level search is filename/path search only. It must not read file contents.
- Rename resolves both source and destination under the same root.
- Remove with `recursive=false` must fail for non-empty directories.
- Watch handles must be unwatched when a component unmounts or switches root.

## 3. Implementation Patterns

### 3.1 Path Resolution Pattern

Applicable to all file operations.

1. Join project root and `relPath`.
2. Clean the result.
3. Require the result to equal root or have the root path plus separator prefix.
4. Return an error for path traversal.

Reference implementation: `internal/filetree/filetree.go`.

Current guard is lexical and does not resolve symlinks. A symlink inside the project that points outside
the root is an accepted local-project limitation unless product requirements change.

### 3.2 Preview Pattern

Applicable to `FileBrowser` and `fileExplorer/*` preview components.

1. Read metadata first when deciding preview type.
2. Use `previewKind.js`, `languageMap.ts`, and `highlight.ts` for frontend rendering decisions.
3. Use specialized previews for images, media, PDF, Markdown, binary, and code/text.
4. Show truncation or binary state instead of trying to render unsupported bytes as text.

Reference components: `BinaryBanner.vue`, `CodeEditor.vue`, `ImagePreview.vue`,
`MarkdownPreview.vue`, `MediaPreview.vue`, `PdfPreview.vue`.

### 3.3 Search Pattern

Applicable to `FileBrowser.vue` and `CodeEditor.vue`.

1. Left-side project search calls `SearchProjectFiles` through `fsBridge.searchPaths`.
2. Backend search matches file and directory basename or project-relative path, case-insensitively.
3. Backend search skips heavy/generated directories such as `.git`, `node_modules`, `vendor`, `dist`,
   `build`, `target`, `.next`, `.nuxt`, `.vite`, `.scannerwork`, `.review-tmp`, and `.review-reports`.
4. Backend search enforces both a result cap and a visited-entry cap. It returns a `truncated` flag when
   either cap is reached; frontend must show truncation state instead of continuing unbounded work.
5. Current-file content search is handled inside CodeMirror with `@codemirror/search`. It searches only
   the open editor document and must not call backend APIs or scan the whole project.

### 3.4 Edit Pattern

Applicable to text/code save flows.

1. Read bytes with mod time.
2. Edit in frontend state.
3. Write through `WriteProjectFileBytes` with the expected mod time.
4. If backend returns `stale_modtime`, surface a conflict instead of silently overwriting.
5. New files may use `createIfMissing=true`; existing writes must remain bounded by the 16 MB hard limit.

### 3.5 Watch Pattern

Applicable to live tree refresh.

1. `WatchProjectDir` returns a numeric handle.
2. Backend watcher callback emits `fs:dir-changed` with the watched relative directory string.
3. Frontend refreshes the affected directory.
4. `UnwatchProjectDir` is called for each handle when no longer needed.

## 4. Allowed And Forbidden

### Allowed Changes

- Add new preview components for recognized file types.
- Add frontend-only rendering helpers when backed by tests.
- Add backend metadata fields if they are project-relative and bounded.
- Improve symlink handling if tests preserve the root-safety contract.

### Forbidden Changes

- Do not accept arbitrary absolute paths from the frontend.
- Do not remove root traversal checks from create, read, write, rename, remove, trash, or watch.
- Do not read unbounded files into memory.
- Do not write files larger than the hard byte limit.
- Do not silently overwrite when `expectedModTime` conflicts.
- Do not keep file watchers alive after the consuming UI is gone.

## 5. New Implementation Checklist

### Required

- [ ] Resolve all paths through the filetree root guard.
- [ ] Keep bytes reads/writes bounded.
- [ ] Keep project-level search to filename/path matching; content search stays editor-local.
- [ ] Preserve atomic write behavior for bytes saves.
- [ ] Preserve conflict detection for editor saves.
- [ ] Cover backend file behavior with Go tests.

### Conditional

- [ ] If adding preview kinds, verify binary/truncated states.
- [ ] If changing project search, test skip directories, result caps, and directory trailing slash behavior.
- [ ] If changing watcher behavior, test duplicate handles and unwatch semantics.
- [ ] If changing frontend file operations, verify unmount cleanup.

### Verification

- [ ] `$GO test ./internal/filetree/`
- [ ] Relevant frontend `node --test` helpers if preview/edit routing logic changes.
