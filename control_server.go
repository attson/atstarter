package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"atstarter/internal/control"
	"atstarter/internal/runner"
	"atstarter/internal/scanner"
	"atstarter/internal/store"
)

type controlServer struct {
	app       *App
	statePath string
	token     string
	server    *http.Server
	listener  net.Listener
}

func controlStatePath(configPath string) string {
	return configPath + ".control.json"
}

func startControlServer(app *App, statePath string) (*controlServer, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &controlServer{app: app, statePath: statePath, token: token, listener: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/rpc", s.handleRPC)
	s.server = &http.Server{Handler: mux}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		_ = ln.Close()
		return nil, err
	}
	if err := control.WriteState(statePath, control.State{
		URL:     "http://" + ln.Addr().String(),
		Token:   token,
		PID:     os.Getpid(),
		Version: Version,
	}); err != nil {
		_ = ln.Close()
		return nil, err
	}
	go func() {
		_ = s.server.Serve(ln)
	}()
	return s, nil
}

func (s *controlServer) Close() {
	_ = os.Remove(s.statePath)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if s.server != nil {
		_ = s.server.Shutdown(ctx)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (s *controlServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeControl(w, control.Response{OK: false, Error: &control.Error{Code: "unauthorized", Message: "invalid control token"}})
		return
	}
	writeControl(w, control.Response{OK: true, Data: map[string]interface{}{"version": Version, "pid": os.Getpid()}})
}

func (s *controlServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeControl(w, control.Response{OK: false, Error: &control.Error{Code: "method_not_allowed", Message: "POST required"}})
		return
	}
	if !s.authorized(r) {
		writeControl(w, control.Response{OK: false, Error: &control.Error{Code: "unauthorized", Message: "invalid control token"}})
		return
	}
	var req control.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeControl(w, control.Response{OK: false, Error: &control.Error{Code: "bad_request", Message: err.Error()}})
		return
	}
	data, err := s.dispatch(req.Method, req.Params)
	if err != nil {
		writeControl(w, errorResponse(err))
		return
	}
	writeControl(w, control.Response{OK: true, Data: data})
}

func (s *controlServer) authorized(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+s.token
}

func writeControl(w http.ResponseWriter, resp control.Response) {
	w.Header().Set("Content-Type", "application/json")
	if !resp.OK {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func errorResponse(err error) control.Response {
	var ce control.RemoteError
	if errors.As(err, &ce) {
		return control.Response{OK: false, Error: &control.Error{Code: ce.Code, Message: ce.Message, Hint: ce.Hint}}
	}
	return control.Response{OK: false, Error: &control.Error{Code: "operation_failed", Message: err.Error()}}
}

type targetParams struct {
	Project string       `json:"project,omitempty"`
	Command string       `json:"command,omitempty"`
	Group   string       `json:"group,omitempty"`
	Service string       `json:"service,omitempty"`
	ID      string       `json:"id,omitempty"`
	Path    string       `json:"path,omitempty"`
	Name    string       `json:"name,omitempty"`
	Type    string       `json:"type,omitempty"`
	Force   bool         `json:"force,omitempty"`
	Tail    int          `json:"tail,omitempty"`
	Add     bool         `json:"add,omitempty"`
	Roots   []string     `json:"roots,omitempty"`
	Items   []itemParams `json:"items,omitempty"`
}

type itemParams struct {
	Project string `json:"project,omitempty"`
	Command string `json:"command,omitempty"`
}

func decodeParams[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 {
		return v, nil
	}
	err := json.Unmarshal(raw, &v)
	return v, err
}

func (s *controlServer) dispatch(method string, raw json.RawMessage) (interface{}, error) {
	switch method {
	case "app.ping":
		return map[string]interface{}{"version": Version, "pid": os.Getpid()}, nil
	case "workspace.list":
		return s.app.GetWorkspaces()
	case "workspace.set":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if err := s.app.SetWorkspaces(p.Roots); err != nil {
			return nil, err
		}
		return map[string]interface{}{"roots": p.Roots}, nil
	case "workspace.scan":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if len(p.Roots) == 0 {
			return nil, control.RemoteError{Code: "missing_roots", Message: "at least one workspace root is required"}
		}
		candidates := s.app.ScanWorkspaces(p.Roots)
		result := map[string]interface{}{"candidates": projectSummaries(candidates)}
		if p.Add {
			detected := make([]store.Project, 0, len(candidates))
			for _, candidate := range candidates {
				if candidate.DetectedType != "unknown" {
					detected = append(detected, candidate)
				}
			}
			if err := s.app.AddScanned(detected); err != nil {
				return nil, err
			}
			result["added"] = len(detected)
		}
		return result, nil
	case "project.add":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if p.Path == "" {
			return nil, control.RemoteError{Code: "missing_path", Message: "project path is required"}
		}
		return s.app.AddProject(p.Path)
	case "project.list":
		projects, err := s.app.ListProjects()
		if err != nil {
			return nil, err
		}
		return projectSummaries(projects), nil
	case "project.commands":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		return store.NormalizeProjectCommands(project).Commands, nil
	case "project.detection.options":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		project = scanner.AddDetectionOptions(project)
		return project.DetectionOptions, nil
	case "project.detection.switch":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		return s.app.switchProjectDetectionType(project.ID, p.Type)
	case "project.start":
		return s.projectStartStop(raw, "start")
	case "project.stop":
		return s.projectStartStop(raw, "stop")
	case "project.status":
		p, project, cmd, err := s.resolveProjectCommand(raw)
		if err != nil {
			return nil, err
		}
		runID := runIDForCommand(project.ID, cmd.ID)
		return map[string]interface{}{"projectId": project.ID, "commandId": cmd.ID, "runId": runID, "status": s.app.GetStatus(runID), "target": p}, nil
	case "project.logs":
		p, project, cmd, err := s.resolveProjectCommand(raw)
		if err != nil {
			return nil, err
		}
		runID := runIDForCommand(project.ID, cmd.ID)
		return logsPayload(runID, s.app.GetLogs(runID), p.Tail, map[string]string{"projectId": project.ID, "commandId": cmd.ID}), nil
	case "project.logs.clear":
		_, project, cmd, err := s.resolveProjectCommand(raw)
		if err != nil {
			return nil, err
		}
		runID := runIDForCommand(project.ID, cmd.ID)
		s.app.ClearLogs(runID)
		return map[string]interface{}{"runId": runID}, nil
	case "group.list":
		return s.app.ListGroups()
	case "group.start":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		group, err := s.resolveGroup(p.Group)
		if err != nil {
			return nil, err
		}
		return s.app.StartGroup(group.ID)
	case "group.stop":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		group, err := s.resolveGroup(p.Group)
		if err != nil {
			return nil, err
		}
		return map[string]string{"groupId": group.ID}, s.app.StopGroup(group.ID)
	case "group.save":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		return s.saveGroupFromParams(p)
	case "group.remove":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		group, err := s.resolveGroup(p.Group)
		if err != nil {
			return nil, err
		}
		return map[string]string{"groupId": group.ID}, s.app.RemoveGroup(group.ID)
	case "group.add_item":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		return s.updateGroupItem(p, true)
	case "group.remove_item":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		return s.updateGroupItem(p, false)
	case "docker.info":
		return s.app.DockerAvailable(), nil
	case "container.list":
		return s.app.ListContainers()
	case "container.start", "container.stop", "container.restart", "container.remove":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		id := p.ID
		if id == "" {
			return nil, control.RemoteError{Code: "missing_container", Message: "container id is required"}
		}
		switch method {
		case "container.start":
			return map[string]string{"id": id}, s.app.StartContainer(id)
		case "container.stop":
			return map[string]string{"id": id}, s.app.StopContainer(id)
		case "container.remove":
			return map[string]interface{}{"id": id, "force": p.Force}, s.app.RemoveContainer(id, p.Force)
		default:
			return map[string]string{"id": id}, s.app.RestartContainer(id)
		}
	case "container.logs.start":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if p.ID == "" {
			return nil, control.RemoteError{Code: "missing_container", Message: "container id is required"}
		}
		err = s.app.FollowContainerLogs(p.ID)
		if err != nil {
			status := s.app.GetStatus(containerRunID(p.ID))
			if status.State != runner.StatusRunning {
				return nil, err
			}
		}
		return map[string]string{"runId": containerRunID(p.ID), "id": p.ID}, nil
	case "container.logs.stop":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if p.ID == "" {
			return nil, control.RemoteError{Code: "missing_container", Message: "container id is required"}
		}
		return map[string]string{"runId": containerRunID(p.ID), "id": p.ID}, s.app.StopFollowContainerLogs(p.ID)
	case "container.logs":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if p.ID == "" {
			return nil, control.RemoteError{Code: "missing_container", Message: "container id is required"}
		}
		runID := containerRunID(p.ID)
		return logsPayload(runID, s.app.GetLogs(runID), p.Tail, map[string]string{"id": p.ID}), nil
	case "container.status":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		if p.ID == "" {
			return nil, control.RemoteError{Code: "missing_container", Message: "container id is required"}
		}
		runID := containerRunID(p.ID)
		return map[string]interface{}{"id": p.ID, "runId": runID, "status": s.app.GetStatus(runID)}, nil
	case "compose.services":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		return s.app.ListComposeServices(project.ID)
	case "compose.up", "compose.stop", "compose.restart", "compose.down":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		switch method {
		case "compose.up":
			return map[string]string{"projectId": project.ID, "service": p.Service}, s.app.ComposeUp(project.ID, p.Service)
		case "compose.stop":
			return map[string]string{"projectId": project.ID, "service": p.Service}, s.app.ComposeStop(project.ID, p.Service)
		case "compose.restart":
			return map[string]string{"projectId": project.ID, "service": p.Service}, s.app.ComposeRestart(project.ID, p.Service)
		default:
			return map[string]string{"projectId": project.ID}, s.app.ComposeDown(project.ID)
		}
	case "compose.logs.start":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		err = s.app.FollowComposeLogs(project.ID, p.Service)
		if err != nil {
			status := s.app.GetStatus(composeRunID(project.ID, p.Service))
			if status.State != runner.StatusRunning {
				return nil, err
			}
		}
		return map[string]string{"runId": composeRunID(project.ID, p.Service), "projectId": project.ID, "service": p.Service}, nil
	case "compose.logs.stop":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		return map[string]string{"runId": composeRunID(project.ID, p.Service)}, s.app.StopFollowComposeLogs(project.ID, p.Service)
	case "compose.logs":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		runID := composeRunID(project.ID, p.Service)
		return logsPayload(runID, s.app.GetLogs(runID), p.Tail, map[string]string{"projectId": project.ID, "service": p.Service}), nil
	case "compose.logs.status":
		p, err := decodeParams[targetParams](raw)
		if err != nil {
			return nil, err
		}
		project, err := s.resolveProject(p.Project)
		if err != nil {
			return nil, err
		}
		runID := composeRunID(project.ID, p.Service)
		return map[string]interface{}{"projectId": project.ID, "service": p.Service, "runId": runID, "status": s.app.GetStatus(runID)}, nil
	default:
		return nil, control.RemoteError{Code: "unknown_method", Message: "unknown control method: " + method}
	}
}

func logsPayload(runID string, logs []string, tail int, extra map[string]string) map[string]interface{} {
	total := len(logs)
	if tail > 0 && len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}
	payload := map[string]interface{}{"runId": runID, "logs": logs, "total": total}
	for k, v := range extra {
		payload[k] = v
	}
	return payload
}

func (s *controlServer) saveGroupFromParams(p targetParams) (store.LaunchGroup, error) {
	name := strings.TrimSpace(p.Name)
	var existing store.LaunchGroup
	if p.Group != "" {
		group, err := s.resolveGroup(p.Group)
		if err != nil {
			return store.LaunchGroup{}, err
		}
		existing = group
		if name == "" {
			name = group.Name
		}
	}
	if name == "" {
		return store.LaunchGroup{}, control.RemoteError{Code: "missing_group_name", Message: "group name is required"}
	}
	items := existing.Items
	if p.Items != nil {
		resolved, err := s.resolveGroupItems(p.Items)
		if err != nil {
			return store.LaunchGroup{}, err
		}
		items = resolved
	}
	return s.app.SaveGroup(store.LaunchGroup{ID: existing.ID, Name: name, Items: items})
}

func (s *controlServer) updateGroupItem(p targetParams, add bool) (store.LaunchGroup, error) {
	group, err := s.resolveGroup(p.Group)
	if err != nil {
		return store.LaunchGroup{}, err
	}
	items, err := s.resolveGroupItems(p.Items)
	if err != nil {
		return store.LaunchGroup{}, err
	}
	if len(items) != 1 {
		return store.LaunchGroup{}, control.RemoteError{Code: "invalid_group_item", Message: "exactly one group item is required"}
	}
	item := items[0]
	if add {
		for _, existing := range group.Items {
			if existing.ProjectID == item.ProjectID && existing.CommandID == item.CommandID {
				return group, nil
			}
		}
		group.Items = append(group.Items, item)
	} else {
		next := group.Items[:0]
		for _, existing := range group.Items {
			if existing.ProjectID == item.ProjectID && existing.CommandID == item.CommandID {
				continue
			}
			next = append(next, existing)
		}
		group.Items = next
	}
	return s.app.SaveGroup(group)
}

func (s *controlServer) resolveGroupItems(items []itemParams) ([]store.GroupItem, error) {
	out := make([]store.GroupItem, 0, len(items))
	for _, item := range items {
		project, err := s.resolveProject(item.Project)
		if err != nil {
			return nil, err
		}
		cmd, err := resolveCommand(project, item.Command)
		if err != nil {
			return nil, err
		}
		out = append(out, store.GroupItem{ProjectID: project.ID, CommandID: cmd.ID})
	}
	return out, nil
}

func (s *controlServer) projectStartStop(raw json.RawMessage, op string) (interface{}, error) {
	_, project, cmd, err := s.resolveProjectCommand(raw)
	if err != nil {
		return nil, err
	}
	runID := runIDForCommand(project.ID, cmd.ID)
	if op == "start" {
		err = s.app.StartProjectCommand(project.ID, cmd.ID)
	} else {
		err = s.app.StopProjectCommand(project.ID, cmd.ID)
	}
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"projectId": project.ID, "commandId": cmd.ID, "runId": runID, "status": s.app.GetStatus(runID)}, nil
}

func (s *controlServer) resolveProjectCommand(raw json.RawMessage) (targetParams, store.Project, store.LaunchCommand, error) {
	p, err := decodeParams[targetParams](raw)
	if err != nil {
		return p, store.Project{}, store.LaunchCommand{}, err
	}
	project, err := s.resolveProject(p.Project)
	if err != nil {
		return p, store.Project{}, store.LaunchCommand{}, err
	}
	cmd, err := resolveCommand(project, p.Command)
	if err != nil {
		return p, store.Project{}, store.LaunchCommand{}, err
	}
	return p, project, cmd, nil
}

func (s *controlServer) resolveProject(target string) (store.Project, error) {
	cfg, err := s.app.store.Load()
	if err != nil {
		return store.Project{}, err
	}
	return resolveProjectFrom(cfg.Projects, target)
}

func resolveProjectFrom(projects []store.Project, target string) (store.Project, error) {
	if target == "" {
		return store.Project{}, control.RemoteError{Code: "missing_project", Message: "project id or name is required"}
	}
	var matches []store.Project
	for _, p := range projects {
		if p.ID == target || p.Name == target || strings.EqualFold(p.Name, target) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return store.Project{}, control.RemoteError{Code: "ambiguous_project", Message: "project target matches multiple projects; use project id"}
	}
	return store.Project{}, control.RemoteError{Code: "project_not_found", Message: "project not found: " + target}
}

func resolveCommand(project store.Project, target string) (store.LaunchCommand, error) {
	project = store.NormalizeProjectCommands(project)
	if target == "" {
		target = store.DefaultCommandID
	}
	var matches []store.LaunchCommand
	for _, c := range project.Commands {
		if c.ID == target || c.Name == target || strings.EqualFold(c.Name, target) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return store.LaunchCommand{}, control.RemoteError{Code: "ambiguous_command", Message: "command target matches multiple commands; use command id"}
	}
	return store.LaunchCommand{}, control.RemoteError{Code: "command_not_found", Message: "command not found: " + target}
}

func (s *controlServer) resolveGroup(target string) (store.LaunchGroup, error) {
	cfg, err := s.app.store.Load()
	if err != nil {
		return store.LaunchGroup{}, err
	}
	if target == "" {
		return store.LaunchGroup{}, control.RemoteError{Code: "missing_group", Message: "group id or name is required"}
	}
	var matches []store.LaunchGroup
	for _, g := range cfg.Groups {
		if g.ID == target || g.Name == target || strings.EqualFold(g.Name, target) {
			matches = append(matches, g)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return store.LaunchGroup{}, control.RemoteError{Code: "ambiguous_group", Message: "group target matches multiple groups; use group id"}
	}
	return store.LaunchGroup{}, control.RemoteError{Code: "group_not_found", Message: "group not found: " + target}
}

type projectSummary struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Path         string                `json:"path"`
	DetectedType string                `json:"detectedType"`
	Commands     []store.LaunchCommand `json:"commands"`
}

func projectSummaries(projects []store.Project) []projectSummary {
	out := make([]projectSummary, 0, len(projects))
	for _, p := range projects {
		p = store.NormalizeProjectCommands(p)
		out = append(out, projectSummary{ID: p.ID, Name: p.Name, Path: p.Path, DetectedType: p.DetectedType, Commands: p.Commands})
	}
	return out
}
