// Package store 负责 atstarter 配置(工作区 + 项目列表)的持久化。
package store

import (
	"crypto/sha1"
	"encoding/hex"
	"reflect"
)

const DefaultCommandID = "default"

// LaunchCommand 是项目下的一套可启动命令。
type LaunchCommand struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Cwd       string            `json:"cwd"`
	Env       map[string]string `json:"env"`
	IsDefault bool              `json:"isDefault"`
}

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
	Commands     []LaunchCommand   `json:"commands,omitempty"`
	ComposeFile  string            `json:"composeFile,omitempty"` // compose 文件相对路径;空则用 docker 默认发现
}

// GroupItem 指向一个项目的一条明确启动命令。
type GroupItem struct {
	ProjectID string `json:"projectId"`
	CommandID string `json:"commandId"`
}

// LaunchGroup 是可一键启动/停止的一组命令。
type LaunchGroup struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Items []GroupItem `json:"items"`
}

// Config 是配置文件的顶层结构。
type Config struct {
	Version    int           `json:"version"`
	Workspaces []string      `json:"workspaces"`
	Projects   []Project     `json:"projects"`
	Groups     []LaunchGroup `json:"groups"`
}

// IDForPath 由项目绝对路径生成稳定 ID(去重依据)。
func IDForPath(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])
}

// IDForCommand 为项目内非默认命令生成稳定 ID。
func IDForCommand(projectID, name, line string) string {
	id := IDForPath("command:" + projectID + ":" + name + ":" + line)
	return id[:12]
}

// NormalizeProjectCommands 兼容旧配置,并保证每个项目至少有一条 default 命令。
func NormalizeProjectCommands(p Project) Project {
	if len(p.Commands) == 0 && p.Command != "" {
		p.Commands = []LaunchCommand{{
			ID:        DefaultCommandID,
			Name:      "Default",
			Command:   p.Command,
			Args:      p.Args,
			Cwd:       p.Cwd,
			Env:       p.Env,
			IsDefault: true,
		}}
	}
	if len(p.Commands) == 0 {
		return p
	}
	if len(p.Commands) == 1 && p.Commands[0].ID == DefaultCommandID && p.Command != "" &&
		(p.Commands[0].Command != p.Command || !reflect.DeepEqual(p.Commands[0].Args, p.Args) ||
			p.Commands[0].Cwd != p.Cwd || !reflect.DeepEqual(p.Commands[0].Env, p.Env)) {
		p.Commands[0].Command = p.Command
		p.Commands[0].Args = p.Args
		p.Commands[0].Cwd = p.Cwd
		p.Commands[0].Env = p.Env
	}
	defaultIndex := -1
	for i := range p.Commands {
		if p.Commands[i].ID == "" {
			p.Commands[i].ID = IDForCommand(p.ID, p.Commands[i].Name, p.Commands[i].Command)
		}
		if p.Commands[i].Name == "" {
			p.Commands[i].Name = "Command"
		}
		if p.Commands[i].Args == nil {
			p.Commands[i].Args = []string{}
		}
		if p.Commands[i].IsDefault && defaultIndex == -1 {
			defaultIndex = i
		}
	}
	if defaultIndex == -1 {
		defaultIndex = 0
	}
	for i := range p.Commands {
		p.Commands[i].IsDefault = i == defaultIndex
		if i == defaultIndex {
			p.Commands[i].ID = DefaultCommandID
			p.Command = p.Commands[i].Command
			p.Args = p.Commands[i].Args
			p.Cwd = p.Commands[i].Cwd
			p.Env = p.Commands[i].Env
		} else if p.Commands[i].ID == DefaultCommandID {
			p.Commands[i].ID = IDForCommand(p.ID, p.Commands[i].Name, p.Commands[i].Command)
		}
	}
	return p
}

// DefaultCommand 返回项目默认命令。没有命令时返回零值。
func DefaultCommand(p Project) LaunchCommand {
	p = NormalizeProjectCommands(p)
	for _, c := range p.Commands {
		if c.IsDefault {
			return c
		}
	}
	return LaunchCommand{}
}
