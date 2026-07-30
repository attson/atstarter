package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"atstarter/internal/control"
)

type cliEnvelope struct {
	OK    bool           `json:"ok"`
	Data  interface{}    `json:"data,omitempty"`
	Error *control.Error `json:"error,omitempty"`
}

type cliLogData struct {
	RunID string   `json:"runId"`
	Logs  []string `json:"logs"`
	Total int      `json:"total"`
}

func runCLI(args []string) int {
	return runCLIWithIO(args, os.Stdout, os.Stderr)
}

func runCLIWithIO(args []string, stdout, stderr io.Writer) int {
	_ = stderr
	configPath, rest, err := parseGlobalCLIFlags(args)
	if err != nil {
		writeCLIError(stdout, err)
		return 2
	}
	if len(rest) == 0 || rest[0] == "help" || rest[0] == "--help" || rest[0] == "-h" {
		writeCLIData(stdout, cliUsage())
		return 0
	}

	client := control.Client{
		StatePath: controlStatePath(configPath),
		HTTP:      &http.Client{Timeout: 3 * time.Second},
	}
	data, err := dispatchCLI(client, rest, stdout)
	if err != nil {
		writeCLIError(stdout, err)
		return exitCodeForError(err)
	}
	if data != nil {
		writeCLIData(stdout, data)
	}
	return 0
}

func parseGlobalCLIFlags(args []string) (string, []string, error) {
	configPath := defaultConfigPath()
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return "", nil, errors.New("--config requires a path")
			}
			i++
			configPath = args[i]
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
		case arg == "--json":
			// JSON is the only output mode; keep the flag for script readability.
		default:
			rest = append(rest, args[i:]...)
			return configPath, rest, nil
		}
	}
	return configPath, rest, nil
}

func dispatchCLI(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	switch args[0] {
	case "app":
		return runCLIApp(client, args[1:])
	case "scan":
		return runCLIScan(client, args[1:])
	case "project":
		return runCLIProject(client, args[1:], stdout)
	case "group":
		return runCLIGroup(client, args[1:])
	case "docker":
		return runCLIDocker(client, args[1:])
	case "container":
		return runCLIContainer(client, args[1:], stdout)
	case "compose":
		return runCLICompose(client, args[1:], stdout)
	default:
		return nil, control.RemoteError{Code: "unknown_command", Message: "unknown cli command: " + args[0]}
	}
}

func runCLIScan(client control.Client, args []string) (interface{}, error) {
	var roots []string
	add := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--add":
			add = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, usageError("unknown scan flag: " + args[i])
			}
			roots = append(roots, args[i])
		}
	}
	if len(roots) == 0 {
		return nil, usageError("at least one workspace root is required")
	}
	return callControl(client, "workspace.scan", targetParams{Roots: roots, Add: add})
}

func runCLIApp(client control.Client, args []string) (interface{}, error) {
	if len(args) == 0 {
		return nil, usageError("app command required")
	}
	switch args[0] {
	case "ping":
		return callControl(client, "app.ping", nil)
	case "start":
		fs := newCLIFlagSet("app start")
		wait := fs.Bool("wait", false, "wait for control service")
		timeout := fs.Duration("timeout", 20*time.Second, "wait timeout")
		if err := fs.Parse(args[1:]); err != nil {
			return nil, err
		}
		if data, err := callControl(client, "app.ping", nil); err == nil {
			return map[string]interface{}{"alreadyRunning": true, "app": data}, nil
		}
		pid, err := launchDesktopApp()
		if err != nil {
			return nil, control.RemoteError{Code: "launch_failed", Message: err.Error()}
		}
		if !*wait {
			return map[string]interface{}{"started": true, "pid": pid}, nil
		}
		data, err := waitForApp(client, *timeout)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"started": true, "pid": pid, "app": data}, nil
	default:
		return nil, usageError("unknown app command: " + args[0])
	}
}

func runCLIProject(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	if len(args) == 0 {
		return nil, usageError("project command required")
	}
	switch args[0] {
	case "add":
		target, _, err := parseTarget(args[1:], "path")
		if err != nil {
			return nil, err
		}
		return callControl(client, "project.add", targetParams{Path: target})
	case "list":
		return callControl(client, "project.list", nil)
	case "commands":
		target, _, err := parseTarget(args[1:], "project")
		if err != nil {
			return nil, err
		}
		return callControl(client, "project.commands", targetParams{Project: target})
	case "detection-options":
		target, _, err := parseTarget(args[1:], "project")
		if err != nil {
			return nil, err
		}
		return callControl(client, "project.detection.options", targetParams{Project: target})
	case "switch-type":
		target, rest, err := parseTarget(args[1:], "project")
		if err != nil {
			return nil, err
		}
		fs := newCLIFlagSet("project switch-type")
		detectedType := fs.String("type", "", "detected type")
		if err := fs.Parse(rest); err != nil {
			return nil, err
		}
		if *detectedType == "" {
			return nil, usageError("--type is required")
		}
		return callControl(client, "project.detection.switch", targetParams{Project: target, Type: *detectedType})
	case "start", "stop", "status", "logs", "logs-clear", "restart":
		return runCLIProjectRuntime(client, args, stdout)
	default:
		return nil, usageError("unknown project command: " + args[0])
	}
}

func runCLIProjectRuntime(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	op := args[0]
	target, rest, err := parseTarget(args[1:], "project")
	if err != nil {
		return nil, err
	}
	fs := newCLIFlagSet("project " + op)
	command := fs.String("command", "", "command id or name")
	tail := fs.Int("tail", 0, "log tail line count")
	follow := fs.Bool("follow", false, "poll and stream new logs")
	wait := fs.Bool("wait", false, "wait until target state is reached")
	timeout := fs.Duration("timeout", 20*time.Second, "wait timeout")
	if err := fs.Parse(rest); err != nil {
		return nil, err
	}
	params := targetParams{Project: target, Command: *command, Tail: *tail}
	switch op {
	case "start":
		data, err := callControl(client, "project.start", params)
		if err != nil || !*wait {
			return data, err
		}
		return waitForState(client, "project.status", params, "running", *timeout)
	case "stop":
		data, err := callControl(client, "project.stop", params)
		if err != nil || !*wait {
			return data, err
		}
		return waitForState(client, "project.status", params, "stopped", *timeout)
	case "restart":
		if _, err := callControl(client, "project.stop", params); err != nil {
			return nil, err
		}
		data, err := callControl(client, "project.start", params)
		if err != nil || !*wait {
			return data, err
		}
		return waitForState(client, "project.status", params, "running", *timeout)
	case "status":
		return callControl(client, "project.status", params)
	case "logs":
		if *follow {
			return nil, streamLogs(client, stdout, "project.logs", params, *tail)
		}
		return callControl(client, "project.logs", params)
	case "logs-clear":
		return callControl(client, "project.logs.clear", params)
	default:
		return nil, usageError("unknown project command: " + op)
	}
}

func runCLIGroup(client control.Client, args []string) (interface{}, error) {
	if len(args) == 0 {
		return nil, usageError("group command required")
	}
	switch args[0] {
	case "list":
		return callControl(client, "group.list", nil)
	case "start", "stop":
		target, _, err := parseTarget(args[1:], "group")
		if err != nil {
			return nil, err
		}
		return callControl(client, "group."+args[0], targetParams{Group: target})
	case "create":
		name, rest, err := parseTarget(args[1:], "group name")
		if err != nil {
			return nil, err
		}
		items, err := parseGroupItemFlags("group create", rest)
		if err != nil {
			return nil, err
		}
		return callControl(client, "group.save", targetParams{Name: name, Items: items})
	case "update":
		target, rest, err := parseTarget(args[1:], "group")
		if err != nil {
			return nil, err
		}
		fs := newCLIFlagSet("group update")
		name := fs.String("name", "", "new group name")
		var itemFlags stringListFlag
		fs.Var(&itemFlags, "item", "project[:command]")
		if err := fs.Parse(rest); err != nil {
			return nil, err
		}
		items, err := parseGroupItems(itemFlags)
		if err != nil {
			return nil, err
		}
		return callControl(client, "group.save", targetParams{Group: target, Name: *name, Items: items})
	case "remove":
		target, _, err := parseTarget(args[1:], "group")
		if err != nil {
			return nil, err
		}
		return callControl(client, "group.remove", targetParams{Group: target})
	case "add-item", "remove-item":
		target, rest, err := parseTarget(args[1:], "group")
		if err != nil {
			return nil, err
		}
		items, err := parseGroupItemFlags("group "+args[0], rest)
		if err != nil {
			return nil, err
		}
		if len(items) != 1 {
			return nil, usageError("exactly one --item is required")
		}
		method := "group.add_item"
		if args[0] == "remove-item" {
			method = "group.remove_item"
		}
		return callControl(client, method, targetParams{Group: target, Items: items})
	default:
		return nil, usageError("unknown group command: " + args[0])
	}
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func parseGroupItemFlags(name string, args []string) ([]itemParams, error) {
	fs := newCLIFlagSet(name)
	var itemFlags stringListFlag
	fs.Var(&itemFlags, "item", "project[:command]")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return parseGroupItems(itemFlags)
}

func parseGroupItems(values []string) ([]itemParams, error) {
	if values == nil {
		return nil, nil
	}
	items := make([]itemParams, 0, len(values))
	for _, value := range values {
		project, command, _ := strings.Cut(value, ":")
		project = strings.TrimSpace(project)
		command = strings.TrimSpace(command)
		if project == "" {
			return nil, usageError("group item project is required")
		}
		items = append(items, itemParams{Project: project, Command: command})
	}
	return items, nil
}

func runCLIDocker(client control.Client, args []string) (interface{}, error) {
	if len(args) != 1 || args[0] != "info" {
		return nil, usageError("expected: docker info")
	}
	return callControl(client, "docker.info", nil)
}

func runCLIContainer(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	if len(args) == 0 {
		return nil, usageError("container command required")
	}
	switch args[0] {
	case "list":
		return callControl(client, "container.list", nil)
	case "start", "stop", "restart", "remove", "status", "logs", "logs-stop":
		return runCLIContainerRuntime(client, args, stdout)
	default:
		return nil, usageError("unknown container command: " + args[0])
	}
}

func runCLIContainerRuntime(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	op := args[0]
	target, rest, err := parseTarget(args[1:], "container")
	if err != nil {
		return nil, err
	}
	fs := newCLIFlagSet("container " + op)
	force := fs.Bool("force", false, "force removal")
	tail := fs.Int("tail", 200, "log tail line count")
	follow := fs.Bool("follow", false, "poll and stream new logs")
	if err := fs.Parse(rest); err != nil {
		return nil, err
	}
	params := targetParams{ID: target, Force: *force, Tail: *tail}
	switch op {
	case "start", "stop", "restart", "remove":
		return callControl(client, "container."+op, params)
	case "status":
		return callControl(client, "container.status", params)
	case "logs":
		if _, err := callControl(client, "container.logs.start", params); err != nil {
			return nil, err
		}
		if *follow {
			return nil, streamLogs(client, stdout, "container.logs", params, *tail)
		}
		time.Sleep(250 * time.Millisecond)
		return callControl(client, "container.logs", params)
	case "logs-stop":
		return callControl(client, "container.logs.stop", params)
	default:
		return nil, usageError("unknown container command: " + op)
	}
}

func runCLICompose(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	if len(args) == 0 {
		return nil, usageError("compose command required")
	}
	switch args[0] {
	case "services", "up", "stop", "restart", "down", "logs", "logs-stop", "logs-status":
		return runCLIComposeRuntime(client, args, stdout)
	default:
		return nil, usageError("unknown compose command: " + args[0])
	}
}

func runCLIComposeRuntime(client control.Client, args []string, stdout io.Writer) (interface{}, error) {
	op := args[0]
	target, rest, err := parseTarget(args[1:], "project")
	if err != nil {
		return nil, err
	}
	fs := newCLIFlagSet("compose " + op)
	service := fs.String("service", "", "compose service")
	tail := fs.Int("tail", 200, "log tail line count")
	follow := fs.Bool("follow", false, "poll and stream new logs")
	if err := fs.Parse(rest); err != nil {
		return nil, err
	}
	params := targetParams{Project: target, Service: *service, Tail: *tail}
	switch op {
	case "services":
		return callControl(client, "compose.services", params)
	case "up", "stop", "restart", "down":
		return callControl(client, "compose."+op, params)
	case "logs":
		if _, err := callControl(client, "compose.logs.start", params); err != nil {
			return nil, err
		}
		if *follow {
			return nil, streamLogs(client, stdout, "compose.logs", params, *tail)
		}
		time.Sleep(250 * time.Millisecond)
		return callControl(client, "compose.logs", params)
	case "logs-stop":
		return callControl(client, "compose.logs.stop", params)
	case "logs-status":
		return callControl(client, "compose.logs.status", params)
	default:
		return nil, usageError("unknown compose command: " + op)
	}
}

func parseTarget(args []string, label string) (string, []string, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", nil, usageError(label + " target is required")
	}
	return args[0], args[1:], nil
}

func callControl(client control.Client, method string, params interface{}) (interface{}, error) {
	var data interface{}
	if err := client.Call(method, params, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func streamLogs(client control.Client, stdout io.Writer, method string, params targetParams, initialTail int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	seen := 0
	if initialTail > 0 {
		params.Tail = initialTail
		first, err := readLogs(client, method, params)
		if err != nil {
			return err
		}
		writeCLIData(stdout, first)
		seen = first.Total
	}
	params.Tail = 0
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			data, err := readLogs(client, method, params)
			if err != nil {
				return err
			}
			if data.Total < seen {
				seen = 0
			}
			if data.Total == seen {
				continue
			}
			start := len(data.Logs) - (data.Total - seen)
			if start < 0 {
				start = 0
			}
			data.Logs = data.Logs[start:]
			writeCLIData(stdout, data)
			seen = data.Total
		}
	}
}

func readLogs(client control.Client, method string, params targetParams) (cliLogData, error) {
	var data cliLogData
	if err := client.Call(method, params, &data); err != nil {
		return cliLogData{}, err
	}
	return data, nil
}

func waitForApp(client control.Client, timeout time.Duration) (interface{}, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		data, err := callControl(client, "app.ping", nil)
		if err == nil {
			return data, nil
		}
		last = err
		time.Sleep(300 * time.Millisecond)
	}
	if last != nil {
		return nil, control.RemoteError{Code: "app_start_timeout", Message: "desktop app did not become ready: " + last.Error()}
	}
	return nil, control.RemoteError{Code: "app_start_timeout", Message: "desktop app did not become ready"}
}

func waitForState(client control.Client, method string, params targetParams, state string, timeout time.Duration) (interface{}, error) {
	deadline := time.Now().Add(timeout)
	var last interface{}
	for time.Now().Before(deadline) {
		data, err := callControl(client, method, params)
		if err != nil {
			return nil, err
		}
		last = data
		if statusState(data) == state {
			return data, nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return last, nil
}

func statusState(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	status, ok := m["status"].(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"State", "state"} {
		if v, ok := status[key].(string); ok {
			return v
		}
	}
	return ""
}

func launchDesktopApp() (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(exe)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return pid, err
	}
	return pid, nil
}

func writeCLIData(w io.Writer, data interface{}) {
	_ = json.NewEncoder(w).Encode(cliEnvelope{OK: true, Data: data})
}

func writeCLIError(w io.Writer, err error) {
	_ = json.NewEncoder(w).Encode(cliEnvelope{OK: false, Error: controlError(err)})
}

func controlError(err error) *control.Error {
	var appErr control.AppNotRunningError
	if errors.As(err, &appErr) {
		return &control.Error{Code: "app_not_running", Message: appErr.Error(), Hint: "run: atstarter cli app start --wait"}
	}
	var remote control.RemoteError
	if errors.As(err, &remote) {
		return &control.Error{Code: remote.Code, Message: remote.Error(), Hint: remote.Hint}
	}
	return &control.Error{Code: "cli_error", Message: err.Error()}
}

func exitCodeForError(err error) int {
	var appErr control.AppNotRunningError
	if errors.As(err, &appErr) {
		return 3
	}
	return 1
}

func newCLIFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func usageError(message string) error {
	return control.RemoteError{Code: "usage", Message: message, Hint: "run: atstarter cli help"}
}

func cliUsage() map[string]interface{} {
	return map[string]interface{}{
		"usage": "atstarter cli [--config path] <resource> <command> [args]",
		"commands": []string{
			"app ping",
			"app start --wait",
			"scan <workspace-root>... --add",
			"project add <path>",
			"project list",
			"project commands <project>",
			"project detection-options <project>",
			"project switch-type <project> --type compose|go|node-pnpm",
			"project start|stop|restart|status <project> --command <command>",
			"project logs <project> --command <command> --tail 200 --follow",
			"project logs-clear <project> --command <command>",
			"group list|create|update|remove|add-item|remove-item|start|stop",
			"group create <name> --item <project>[:command]",
			"group add-item <group> --item <project>[:command]",
			"docker info",
			"container list|start|stop|restart|remove|status <container>",
			"container logs <container> --tail 200 --follow",
			"compose services|up|stop|restart|down <project> --service <service>",
			"compose logs <project> --service <service> --tail 200 --follow",
		},
	}
}
