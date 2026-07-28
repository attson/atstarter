// fsBridge:把 atterm FileTree 需要的文件系统接口,适配到 atstarter 的 wails 绑定。
// 路径体系:相对项目根的 relPath,空串 "" = 根,统一 "/" 分隔(与后端 filetree 约定一致)。
import {
  ListProjectDir,
  ReadProjectFile,
  WriteProjectFile,
  ProjectFileMeta,
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

export interface FileSystemBridge {
  readonly identity: string
  listDir(path: string): Promise<DirEntry[]>
  readFile(path: string): Promise<FileContent>
  writeFile(path: string, content: string): Promise<void>
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
    readFile: (path) => ReadProjectFile(projectId, path) as Promise<FileContent>,
    writeFile: (path, content) => WriteProjectFile(projectId, path, content),
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
