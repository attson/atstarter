package main

// projectFSHandler 把项目内文件当 HTTP 资源服务,供 webview 里的 <img>/<video>/
// <embed>(PDF) 通过稳定 URL 访问本地文件。图片/媒体/PDF 预览用。
//
// URL 形式: /projectfs/<projectID>/<base64.URLEncoding(relPath)>
// 只允许 GET / HEAD。安全: relPath 走 filetree 的 resolve(与 ReadFile 同款路径
// 穿越防护),projectID 必须是已登记项目。
//
// 与 wails 绑定的 *App 分开成独立类型: 避免把 net/http 类型图拖进自动生成的
// TS 绑定,也不暴露前端可调的 ServeHTTP。

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"atstarter/internal/filetree"
)

const projectFSURLPrefix = "/projectfs/"

type projectFSHandler struct {
	app *App
}

func newProjectFSHandler(app *App) *projectFSHandler {
	return &projectFSHandler{app: app}
}

func (h *projectFSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, projectFSURLPrefix) {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// /projectfs/<projectID>/<base64(relPath)>
	rest := strings.TrimPrefix(r.URL.Path, projectFSURLPrefix)
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	projectID := rest[:slash]
	encoded := rest[slash+1:]
	raw, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		http.Error(w, "bad path encoding", http.StatusBadRequest)
		return
	}
	relPath := string(raw)

	root, err := h.app.projectRoot(projectID)
	if err != nil {
		http.Error(w, "unknown project", http.StatusForbidden)
		return
	}
	full, err := filetree.ResolveWithin(root, relPath)
	if err != nil {
		// 只可能是路径穿越,统一 403。
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "stat failed", http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.Error(w, "is a directory", http.StatusForbidden)
		return
	}
	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "open failed", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	// http.ServeContent 处理 Range / Content-Type(按扩展名)/ 缓存头。
	http.ServeContent(w, r, filepath.Base(full), info.ModTime(), f)
}
