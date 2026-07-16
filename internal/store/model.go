// Package store 负责 atstarter 配置(工作区 + 项目列表)的持久化。
package store

import (
	"crypto/sha1"
	"encoding/hex"
)

// Project 是一个可启动项目的完整配置。
type Project struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Path         string            `json:"path"`
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	Cwd          string            `json:"cwd"`
	Env          map[string]string `json:"env"`
	DetectedType string            `json:"detectedType"`
	AutoDetected bool              `json:"autoDetected"`
}

// Config 是配置文件的顶层结构。
type Config struct {
	Version    int       `json:"version"`
	Workspaces []string  `json:"workspaces"`
	Projects   []Project `json:"projects"`
}

// IDForPath 由项目绝对路径生成稳定 ID(去重依据)。
func IDForPath(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])
}
