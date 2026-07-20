package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"atstarter/internal/cmdparse"
	"atstarter/internal/detector"
	"atstarter/internal/runner"
	"atstarter/internal/scanner"
	"atstarter/internal/store"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 绑定层,组装各内部模块并暴露方法给前端。
type App struct {
	ctx    context.Context
	store  *store.Store
	runner *runner.Runner
}

type CommandInput struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Line      string            `json:"line"`
	Cwd       string            `json:"cwd"`
	Env       map[string]string `json:"env"`
	IsDefault bool              `json:"isDefault"`
}

type LaunchResult struct {
	ProjectID string `json:"projectId"`
	CommandID string `json:"commandId"`
	RunID     string `json:"runId"`
	Error     string `json:"error"`
}

// NewApp 用默认配置路径(用户配置目录)构造。
func NewApp() *App {
	return NewAppWithConfig(defaultConfigPath())
}

// NewAppWithConfig 用指定配置路径构造(测试用)。
func NewAppWithConfig(cfgPath string) *App {
	return &App{
		store:  store.New(cfgPath),
		runner: runner.New(5000),
	}
}

// defaultConfigPath 返回各平台标准配置目录下的 config.json。
func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "atstarter", "config.json")
}

// startup 由 Wails 在启动时调用,保存 ctx 并接好日志事件转发。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.runner.SetEmitter(func(l runner.LogLine) {
		runtime.EventsEmit(a.ctx, "log:"+l.ID, map[string]string{
			"stream": l.Stream, "text": l.Text,
		})
	})
	a.runner.SetStatusListener(func(id string, st runner.Status) {
		runtime.EventsEmit(a.ctx, "status:"+id, map[string]interface{}{
			"state": string(st.State), "pid": st.PID, "exitCode": st.ExitCode,
		})
	})
}

// shutdown 由 Wails 在退出时调用,停掉所有进程。
func (a *App) shutdown(ctx context.Context) {
	a.runner.StopAll()
}

// ---- 暴露给前端的方法 ----

// ListProjects 返回所有已保存项目。
func (a *App) ListProjects() ([]store.Project, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Projects, nil
}

// expandHome 把开头的 ~ 或 ~/... 展开为用户家目录。
// 其它形式(绝对路径、相对路径、~user)原样返回。空字符串原样返回。
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// normalizePath 返回清理过的绝对路径,作为项目去重与存储的规范形式。
// 先展开 ~,再取绝对路径。失败(极少见,如 os.Getwd 出错)时退回 filepath.Clean。
func normalizePath(path string) string {
	path = expandHome(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

// AddProject 识别目录并保存为项目。
func (a *App) AddProject(path string) (store.Project, error) {
	path = normalizePath(path)
	if _, err := os.Stat(path); err != nil {
		return store.Project{}, errors.New("path not found: " + path)
	}
	res := detector.Detect(path)
	p := store.Project{
		ID:           store.IDForPath(path),
		Name:         filepath.Base(path),
		Path:         path,
		DetectedType: res.Type,
		AutoDetected: true,
	}
	if res.Command != "" {
		if cmd, args, err := cmdparse.Parse(res.Command); err == nil {
			p.Command, p.Args = cmd, args
		}
	}
	p = store.NormalizeProjectCommands(p)
	if err := a.store.Add(p); err != nil {
		return store.Project{}, err
	}
	return p, nil
}

// ScanWorkspaces 扫描给定根目录,返回候选(不自动保存)。
// 每个根目录先展开 ~,让手输的 ~/xxx 也能扫描。
func (a *App) ScanWorkspaces(roots []string) []store.Project {
	expanded := make([]string, len(roots))
	for i, r := range roots {
		expanded[i] = expandHome(r)
	}
	return scanner.Scan(expanded)
}

// PickDirectory 调起系统原生文件夹选择器,返回选中的目录绝对路径。
// 用户取消时返回空字符串(无错误)。
func (a *App) PickDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作区根目录",
	})
}

// AddScanned 批量保存用户勾选的候选项目。
func (a *App) AddScanned(projects []store.Project) error {
	for _, p := range projects {
		p.Path = normalizePath(p.Path)
		p = store.NormalizeProjectCommands(p)
		if err := a.store.Add(p); err != nil {
			return err
		}
	}
	return nil
}

// UpdateProject 覆盖保存一个项目。
func (a *App) UpdateProject(p store.Project) error {
	p = store.NormalizeProjectCommands(p)
	return a.store.Update(p)
}

// UpdateProjectCommand 用 UI 单行命令更新项目的 command/args,并标记为手动。
func (a *App) UpdateProjectCommand(id, line string) (store.Project, error) {
	return a.UpdateProjectCommands(id, "", []CommandInput{{Name: "Default", Line: line, IsDefault: true}})
}

// UpdateProjectCommands 覆盖项目的多套启动命令。分组引用 commandId,因此已有 ID 会保留。
func (a *App) UpdateProjectCommands(id, name string, inputs []CommandInput) (store.Project, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return store.Project{}, err
	}
	for _, p := range cfg.Projects {
		if p.ID == id {
			if strings.TrimSpace(name) != "" {
				p.Name = strings.TrimSpace(name)
			}
			commands := make([]store.LaunchCommand, 0, len(inputs))
			defaultIndex := -1
			for i, input := range inputs {
				cmd, args, err := cmdparse.Parse(input.Line)
				if err != nil {
					return store.Project{}, err
				}
				if input.IsDefault && defaultIndex == -1 {
					defaultIndex = i
				}
				commandName := strings.TrimSpace(input.Name)
				if commandName == "" {
					commandName = "Command"
				}
				commands = append(commands, store.LaunchCommand{
					ID:        input.ID,
					Name:      commandName,
					Command:   cmd,
					Args:      args,
					Cwd:       input.Cwd,
					Env:       input.Env,
					IsDefault: input.IsDefault,
				})
			}
			if len(commands) == 0 {
				return store.Project{}, errors.New("at least one command is required")
			}
			if defaultIndex == -1 {
				defaultIndex = 0
			}
			for i := range commands {
				commands[i].IsDefault = i == defaultIndex
				line := cmdparse.Join(commands[i].Command, commands[i].Args)
				if i == defaultIndex {
					commands[i].ID = store.DefaultCommandID
					p.Command = commands[i].Command
					p.Args = commands[i].Args
					p.Cwd = commands[i].Cwd
					p.Env = commands[i].Env
				} else if commands[i].ID == "" || commands[i].ID == store.DefaultCommandID {
					commands[i].ID = store.IDForCommand(p.ID, commands[i].Name, line)
				}
			}
			p.Commands = commands
			p.AutoDetected = false
			p = store.NormalizeProjectCommands(p)
			if err := a.store.Update(p); err != nil {
				return store.Project{}, err
			}
			return p, nil
		}
	}
	return store.Project{}, errors.New("project not found: " + id)
}

// RemoveProject 删除项目(若在运行先停止)。
func (a *App) RemoveProject(id string) error {
	_ = a.StopProject(id)
	return a.store.Remove(id)
}

func runIDForCommand(projectID, commandID string) string {
	if commandID == "" {
		commandID = store.DefaultCommandID
	}
	return projectID + ":" + commandID
}

func commandByID(p store.Project, commandID string) (store.LaunchCommand, bool) {
	if commandID == "" {
		commandID = store.DefaultCommandID
	}
	p = store.NormalizeProjectCommands(p)
	for _, c := range p.Commands {
		if c.ID == commandID {
			return c, true
		}
	}
	return store.LaunchCommand{}, false
}

// StartProject 启动项目对应的进程。
func (a *App) StartProject(id string) error {
	return a.StartProjectCommand(id, store.DefaultCommandID)
}

// StartProjectCommand 启动项目下指定命令,运行时 ID 为 projectId:commandId。
func (a *App) StartProjectCommand(id, commandID string) error {
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	for _, p := range cfg.Projects {
		if p.ID == id {
			cmd, ok := commandByID(p, commandID)
			if !ok {
				return errors.New("command not found: " + commandID)
			}
			dir := cmd.Cwd
			if dir == "" {
				dir = p.Path
			}
			env := map[string]string{}
			for k, v := range p.Env {
				env[k] = v
			}
			for k, v := range cmd.Env {
				env[k] = v
			}
			return a.runner.Start(runner.Spec{
				ID: runIDForCommand(p.ID, cmd.ID), Command: cmd.Command, Args: cmd.Args, Dir: dir, Env: env,
			})
		}
	}
	return errors.New("project not found: " + id)
}

// StopProject 停止项目进程。
func (a *App) StopProject(id string) error {
	cfg, err := a.store.Load()
	if err != nil {
		return a.runner.Stop(id)
	}
	for _, p := range cfg.Projects {
		if p.ID == id {
			for _, c := range store.NormalizeProjectCommands(p).Commands {
				_ = a.runner.Stop(runIDForCommand(p.ID, c.ID))
			}
			_ = a.runner.Stop(id)
			return nil
		}
	}
	return a.runner.Stop(id)
}

// StopProjectCommand 停止项目下指定命令。
func (a *App) StopProjectCommand(id, commandID string) error {
	return a.runner.Stop(runIDForCommand(id, commandID))
}

// GetStatus 返回项目运行时状态。
func (a *App) GetStatus(id string) runner.Status {
	return a.runner.Status(id)
}

// GetLogs 返回项目日志缓冲快照。
func (a *App) GetLogs(id string) []string {
	return a.runner.Logs(id)
}

func (a *App) ListGroups() ([]store.LaunchGroup, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Groups, nil
}

func (a *App) SaveGroup(group store.LaunchGroup) (store.LaunchGroup, error) {
	return a.store.SaveGroup(group)
}

func (a *App) RemoveGroup(id string) error {
	return a.store.RemoveGroup(id)
}

func (a *App) StartGroup(id string) ([]LaunchResult, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	var group *store.LaunchGroup
	for i := range cfg.Groups {
		if cfg.Groups[i].ID == id {
			group = &cfg.Groups[i]
			break
		}
	}
	if group == nil {
		return nil, errors.New("group not found: " + id)
	}
	results := make([]LaunchResult, 0, len(group.Items))
	for _, item := range group.Items {
		res := LaunchResult{ProjectID: item.ProjectID, CommandID: item.CommandID, RunID: runIDForCommand(item.ProjectID, item.CommandID)}
		if err := a.StartProjectCommand(item.ProjectID, item.CommandID); err != nil {
			res.Error = err.Error()
		}
		results = append(results, res)
	}
	return results, nil
}

func (a *App) StopGroup(id string) error {
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	for _, group := range cfg.Groups {
		if group.ID == id {
			for _, item := range group.Items {
				_ = a.StopProjectCommand(item.ProjectID, item.CommandID)
			}
			return nil
		}
	}
	return errors.New("group not found: " + id)
}

// SetWorkspaces 保存工作区根目录列表。
func (a *App) SetWorkspaces(dirs []string) error {
	return a.store.SetWorkspaces(dirs)
}

// GetWorkspaces 返回已保存的工作区根目录。
func (a *App) GetWorkspaces() ([]string, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Workspaces, nil
}

// GetProjectBranch 返回项目工作目录当前的 git 分支名(纯 UI 显示用)。
// 非 git 仓库、detached HEAD、命令超时或 git 不在 PATH 都返回空串,前端据此隐藏 pill。
func (a *App) GetProjectBranch(projectPath string) string {
	if projectPath == "" {
		return ""
	}
	// 快速否决:没有 .git 目录/文件的话根本不是仓库,免掉 exec 开销。
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	// symbolic-ref --short HEAD 在 detached 时非零退出,正好落到空串分支。
	out, err := exec.CommandContext(ctx, "git", "-C", projectPath, "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
