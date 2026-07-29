// fsBridge:把 atterm FileTree 需要的文件系统接口,适配到 atstarter 的 wails 绑定。
// 路径体系:相对项目根的 relPath,空串 "" = 根,统一 "/" 分隔(与后端 filetree 约定一致)。
import {
  ListProjectDir,
  ReadProjectFile,
  WriteProjectFile,
  ReadProjectFileBytes,
  WriteProjectFileBytes,
  ProjectAssetURL,
  OpenProjectPath,
  ProjectFileMeta,
  SearchProjectFiles,
  CreateProjectFile,
  MkdirProject,
  RenameProject,
  RemoveProjectPath,
  TrashProjectPath,
  WatchProjectDir,
  UnwatchProjectDir,
} from '../../../wailsjs/go/main/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'

export interface DirEntry {
  name: string
  isDir: boolean
  size?: number
}

export interface FileContent {
  content: string
  size: number
  truncated: boolean
  binary: boolean
}

export interface FileMetaInfo {
  size: number
  modTime: number
  isDir: boolean
  isBinary: boolean
}

export interface FileBytes {
  data: number[]
  modTime: number
  isBinary: boolean
  truncatedAt?: number
}

export interface SearchMatch {
  path: string
  name: string
  isDir: boolean
}

export interface SearchResults {
  matches: SearchMatch[]
  truncated: boolean
}

export interface FileSystemBridge {
  readonly identity: string
  listDir(path: string): Promise<DirEntry[]>
  searchPaths(query: string, limit: number): Promise<SearchResults>
  readFile(path: string): Promise<FileContent>
  writeFile(path: string, content: string): Promise<void>
  // 字节接口(CodeMirror 编辑器用:带 modtime 冲突检测)。
  readBytes(path: string, maxBytes: number): Promise<FileBytes>
  writeBytes(path: string, data: Uint8Array, expectedModTime: number, createIfMissing: boolean): Promise<number>
  // 资源 URL(img/video/pdf 预览用)。
  assetUrlFor(path: string): Promise<string>
  // 用系统默认程序打开(不支持预览的文件用)。
  openExternal(path: string): Promise<void>
  fileMeta(path: string): Promise<FileMetaInfo>
  createFile(path: string): Promise<void>
  mkdir(path: string): Promise<void>
  rename(from: string, to: string): Promise<void>
  remove(path: string, recursive: boolean): Promise<void>
  trash(path: string): Promise<void>
  watchDir(path: string): Promise<number>
  unwatchDir(id: number): Promise<void>
  onDirChanged(handler: (relDir: string) => void): () => void
}

// createProjectFSBridge 绑定到某个 projectID,所有路径都是相对该项目根的 relPath。
export function createProjectFSBridge(projectId: string): FileSystemBridge {
  return {
    identity: 'local:' + projectId,
    listDir: async (path) => {
      const entries = await ListProjectDir(projectId, path)
      return (entries || []).map((e) => ({ name: e.name, isDir: e.isDir, size: e.size }))
    },
    searchPaths: (query, limit) => SearchProjectFiles(projectId, query, limit) as Promise<SearchResults>,
    readFile: (path) => ReadProjectFile(projectId, path) as Promise<FileContent>,
    writeFile: (path, content) => WriteProjectFile(projectId, path, content),
    readBytes: (path, maxBytes) => ReadProjectFileBytes(projectId, path, maxBytes) as Promise<FileBytes>,
    writeBytes: (path, data, expectedModTime, createIfMissing) =>
      WriteProjectFileBytes(projectId, path, Array.from(data), expectedModTime, createIfMissing),
    assetUrlFor: (path) => ProjectAssetURL(projectId, path),
    openExternal: (path) => OpenProjectPath(projectId, path),
    fileMeta: (path) => ProjectFileMeta(projectId, path) as Promise<FileMetaInfo>,
    createFile: (path) => CreateProjectFile(projectId, path),
    mkdir: (path) => MkdirProject(projectId, path),
    rename: (from, to) => RenameProject(projectId, from, to),
    remove: (path, recursive) => RemoveProjectPath(projectId, path, recursive),
    trash: (path) => TrashProjectPath(projectId, path),
    watchDir: (path) => WatchProjectDir(projectId, path),
    unwatchDir: (id) => UnwatchProjectDir(id),
    onDirChanged: (handler) => EventsOn('fs:dir-changed', (relDir: string) => handler(relDir)),
  }
}
