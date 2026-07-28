package filetree

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher 封装 fsnotify,按数值句柄管理多个目录监听,变化时回调目录路径。
// 变化路径为「监听时传入的 relPath」(相对 root),便于前端定位刷新哪个目录。
type Watcher struct {
	mu       sync.Mutex
	fsw      *fsnotify.Watcher
	handles  map[int64]string  // id → 被监听的绝对目录
	relByAbs map[string]string // 绝对目录 → relPath(回调时转回相对)
	nextID   int64
	onChange func(relDir string)
}

// NewWatcher 创建监听器并启动事件循环。
func NewWatcher() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fsw:      fsw,
		handles:  make(map[int64]string),
		relByAbs: make(map[string]string),
	}
	go w.loop()
	return w, nil
}

// OnChange 注册变化回调(覆盖式)。回调参数是发生变化的目录 relPath。
func (w *Watcher) OnChange(fn func(relDir string)) {
	w.mu.Lock()
	w.onChange = fn
	w.mu.Unlock()
}

// Watch 监听 root/relPath 目录,返回数值句柄。relPath 空串表示 root。
func (w *Watcher) Watch(root, relPath string) (int64, error) {
	full, err := resolve(root, relPath)
	if err != nil {
		return 0, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.fsw.Add(full); err != nil {
		return 0, err
	}
	w.nextID++
	id := w.nextID
	w.handles[id] = full
	w.relByAbs[full] = relPath
	return id, nil
}

// Unwatch 取消句柄对应的监听。
func (w *Watcher) Unwatch(id int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	full, ok := w.handles[id]
	if !ok {
		return nil
	}
	delete(w.handles, id)
	// 仅当没有其它句柄监听同一目录时才 Remove
	stillWatched := false
	for _, p := range w.handles {
		if p == full {
			stillWatched = true
			break
		}
	}
	if !stillWatched {
		delete(w.relByAbs, full)
		return w.fsw.Remove(full)
	}
	return nil
}

// Close 关闭底层 watcher。
func (w *Watcher) Close() error {
	return w.fsw.Close()
}

func (w *Watcher) loop() {
	for ev := range w.fsw.Events {
		// fsnotify 事件是文件路径;取其父目录匹配监听目录,
		// 也兜底事件路径本身就是被监听目录的情况。
		w.mu.Lock()
		rel, ok := w.relByAbs[parentOf(ev.Name)]
		if !ok {
			rel, ok = w.relByAbs[ev.Name]
		}
		fn := w.onChange
		w.mu.Unlock()
		if ok && fn != nil {
			fn(rel)
		}
	}
}

func parentOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[:i]
		}
	}
	return p
}
