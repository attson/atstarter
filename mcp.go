package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"atstarter/internal/control"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func runMCP(stdin io.Reader, stdout io.Writer) int {
	server := &mcpServer{
		client: control.Client{
			StatePath: controlStatePath(defaultConfigPath()),
			HTTP:      &http.Client{Timeout: 3 * time.Second},
		},
	}
	reader := bufio.NewReader(stdin)
	for {
		msg, err := readMCPMessage(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			return 1
		}
		var req mcpRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		resp := server.handle(req)
		if err := writeMCPMessage(stdout, resp); err != nil {
			return 1
		}
	}
}

type mcpServer struct {
	client control.Client
}

func (s *mcpServer) handle(req mcpRequest) mcpResponse {
	id := rawID(req.ID)
	switch req.Method {
	case "initialize":
		return mcpResponse{JSONRPC: "2.0", ID: id, Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "atstarter", "version": Version},
		}}
	case "tools/list":
		return mcpResponse{JSONRPC: "2.0", ID: id, Result: map[string]interface{}{"tools": atstarterMCPTools()}}
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return mcpFailure(id, -32602, err.Error())
		}
		result := s.callTool(params.Name, params.Arguments)
		return mcpResponse{JSONRPC: "2.0", ID: id, Result: result}
	default:
		return mcpFailure(id, -32601, "method not found")
	}
}

func (s *mcpServer) callTool(name string, args map[string]interface{}) map[string]interface{} {
	var data interface{}
	var err error
	switch name {
	case "atstarter_app_ping":
		data, err = callControl(s.client, "app.ping", nil)
	case "atstarter_app_start":
		data, err = mcpAppStart(s.client, durationArg(args, "timeout", 20*time.Second))
	case "atstarter_scan":
		data, err = callControl(s.client, "workspace.scan", targetParams{Roots: stringSliceArg(args, "roots"), Add: boolArg(args, "add")})
	case "atstarter_project_add":
		data, err = callControl(s.client, "project.add", targetParams{Path: stringArg(args, "path")})
	case "atstarter_project_list":
		data, err = callControl(s.client, "project.list", nil)
	case "atstarter_project_commands":
		data, err = callControl(s.client, "project.commands", targetParams{Project: stringArg(args, "project")})
	case "atstarter_project_detection_options":
		data, err = callControl(s.client, "project.detection.options", targetParams{Project: stringArg(args, "project")})
	case "atstarter_project_switch_type":
		data, err = callControl(s.client, "project.detection.switch", targetParams{Project: stringArg(args, "project"), Type: stringArg(args, "type")})
	case "atstarter_project_start", "atstarter_project_stop", "atstarter_project_restart", "atstarter_project_status", "atstarter_project_logs":
		data, err = s.callProjectTool(name, args)
	case "atstarter_group_list":
		data, err = callControl(s.client, "group.list", nil)
	case "atstarter_group_create":
		data, err = callControl(s.client, "group.save", targetParams{Name: stringArg(args, "name"), Items: itemListArg(args, "items")})
	case "atstarter_group_update":
		data, err = callControl(s.client, "group.save", targetParams{Group: stringArg(args, "group"), Name: stringArg(args, "name"), Items: itemListArg(args, "items")})
	case "atstarter_group_remove":
		data, err = callControl(s.client, "group.remove", targetParams{Group: stringArg(args, "group")})
	case "atstarter_group_add_item":
		data, err = callControl(s.client, "group.add_item", targetParams{Group: stringArg(args, "group"), Items: []itemParams{{Project: stringArg(args, "project"), Command: stringArg(args, "command")}}})
	case "atstarter_group_remove_item":
		data, err = callControl(s.client, "group.remove_item", targetParams{Group: stringArg(args, "group"), Items: []itemParams{{Project: stringArg(args, "project"), Command: stringArg(args, "command")}}})
	case "atstarter_group_start":
		data, err = callControl(s.client, "group.start", targetParams{Group: stringArg(args, "group")})
	case "atstarter_group_stop":
		data, err = callControl(s.client, "group.stop", targetParams{Group: stringArg(args, "group")})
	case "atstarter_docker_info":
		data, err = callControl(s.client, "docker.info", nil)
	case "atstarter_container_list":
		data, err = callControl(s.client, "container.list", nil)
	case "atstarter_container_start", "atstarter_container_stop", "atstarter_container_restart", "atstarter_container_logs":
		data, err = s.callContainerTool(name, args)
	case "atstarter_compose_services", "atstarter_compose_up", "atstarter_compose_stop", "atstarter_compose_restart", "atstarter_compose_down", "atstarter_compose_logs":
		data, err = s.callComposeTool(name, args)
	default:
		err = control.RemoteError{Code: "unknown_tool", Message: "unknown MCP tool: " + name}
	}
	return mcpToolResult(data, err)
}

func (s *mcpServer) callProjectTool(name string, args map[string]interface{}) (interface{}, error) {
	params := targetParams{Project: stringArg(args, "project"), Command: stringArg(args, "command"), Tail: intArg(args, "tail")}
	switch name {
	case "atstarter_project_start":
		return callControl(s.client, "project.start", params)
	case "atstarter_project_stop":
		return callControl(s.client, "project.stop", params)
	case "atstarter_project_restart":
		if _, err := callControl(s.client, "project.stop", params); err != nil {
			return nil, err
		}
		return callControl(s.client, "project.start", params)
	case "atstarter_project_status":
		return callControl(s.client, "project.status", params)
	default:
		return callControl(s.client, "project.logs", params)
	}
}

func (s *mcpServer) callContainerTool(name string, args map[string]interface{}) (interface{}, error) {
	params := targetParams{ID: stringArg(args, "id"), Tail: intArg(args, "tail")}
	switch name {
	case "atstarter_container_start":
		return callControl(s.client, "container.start", params)
	case "atstarter_container_stop":
		return callControl(s.client, "container.stop", params)
	case "atstarter_container_restart":
		return callControl(s.client, "container.restart", params)
	default:
		if _, err := callControl(s.client, "container.logs.start", params); err != nil {
			return nil, err
		}
		time.Sleep(250 * time.Millisecond)
		return callControl(s.client, "container.logs", params)
	}
}

func (s *mcpServer) callComposeTool(name string, args map[string]interface{}) (interface{}, error) {
	params := targetParams{Project: stringArg(args, "project"), Service: stringArg(args, "service"), Tail: intArg(args, "tail")}
	switch name {
	case "atstarter_compose_services":
		return callControl(s.client, "compose.services", params)
	case "atstarter_compose_up":
		return callControl(s.client, "compose.up", params)
	case "atstarter_compose_stop":
		return callControl(s.client, "compose.stop", params)
	case "atstarter_compose_restart":
		return callControl(s.client, "compose.restart", params)
	case "atstarter_compose_down":
		return callControl(s.client, "compose.down", params)
	default:
		if _, err := callControl(s.client, "compose.logs.start", params); err != nil {
			return nil, err
		}
		time.Sleep(250 * time.Millisecond)
		return callControl(s.client, "compose.logs", params)
	}
}

func mcpAppStart(client control.Client, timeout time.Duration) (interface{}, error) {
	if data, err := callControl(client, "app.ping", nil); err == nil {
		return map[string]interface{}{"alreadyRunning": true, "app": data}, nil
	}
	pid, err := launchDesktopApp()
	if err != nil {
		return nil, control.RemoteError{Code: "launch_failed", Message: err.Error()}
	}
	data, err := waitForApp(client, timeout)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"started": true, "pid": pid, "app": data}, nil
}

func atstarterMCPTools() []mcpTool {
	return []mcpTool{
		toolNoArgs("atstarter_app_ping", "Check whether the atstarter desktop app control service is running."),
		toolWithSchema("atstarter_app_start", "Start the atstarter desktop app if it is not running.", objectSchema(map[string]interface{}{"timeout": stringProp("Go duration such as 20s.")}, nil)),
		toolWithSchema("atstarter_scan", "Scan workspace roots and optionally add detected projects.", objectSchema(map[string]interface{}{"roots": arrayProp("Workspace roots.", stringProp("Workspace root path.")), "add": boolProp("Add detected non-unknown projects.")}, []string{"roots"})),
		toolWithSchema("atstarter_project_add", "Add one project directory.", objectSchema(map[string]interface{}{"path": stringProp("Project directory path.")}, []string{"path"})),
		toolNoArgs("atstarter_project_list", "List configured projects and launch commands."),
		toolWithSchema("atstarter_project_commands", "List launch commands for one project.", projectSchema()),
		toolWithSchema("atstarter_project_detection_options", "List available detection types for a project.", projectSchema()),
		toolWithSchema("atstarter_project_switch_type", "Switch a project between compose and ordinary command detection modes.", objectSchema(map[string]interface{}{"project": stringProp("Project id or name."), "type": stringProp("Detection type such as compose, go, node-pnpm.")}, []string{"project", "type"})),
		toolWithSchema("atstarter_project_start", "Start a project launch command.", projectCommandSchema(false)),
		toolWithSchema("atstarter_project_stop", "Stop a project launch command.", projectCommandSchema(false)),
		toolWithSchema("atstarter_project_restart", "Restart a project launch command.", projectCommandSchema(false)),
		toolWithSchema("atstarter_project_status", "Read project launch command runtime status.", projectCommandSchema(false)),
		toolWithSchema("atstarter_project_logs", "Read buffered project logs.", projectCommandSchema(true)),
		toolNoArgs("atstarter_group_list", "List launch groups."),
		toolWithSchema("atstarter_group_create", "Create a launch group.", groupSaveSchema(false)),
		toolWithSchema("atstarter_group_update", "Rename a launch group or replace its items.", groupSaveSchema(true)),
		toolWithSchema("atstarter_group_remove", "Delete a launch group.", objectSchema(map[string]interface{}{"group": stringProp("Group id or name.")}, []string{"group"})),
		toolWithSchema("atstarter_group_add_item", "Add one project command to a launch group.", groupItemSchema()),
		toolWithSchema("atstarter_group_remove_item", "Remove one project command from a launch group.", groupItemSchema()),
		toolWithSchema("atstarter_group_start", "Start a launch group.", objectSchema(map[string]interface{}{"group": stringProp("Group id or name.")}, []string{"group"})),
		toolWithSchema("atstarter_group_stop", "Stop a launch group.", objectSchema(map[string]interface{}{"group": stringProp("Group id or name.")}, []string{"group"})),
		toolNoArgs("atstarter_docker_info", "Check Docker availability."),
		toolNoArgs("atstarter_container_list", "List Docker containers."),
		toolWithSchema("atstarter_container_start", "Start a Docker container.", containerSchema(false)),
		toolWithSchema("atstarter_container_stop", "Stop a Docker container.", containerSchema(false)),
		toolWithSchema("atstarter_container_restart", "Restart a Docker container.", containerSchema(false)),
		toolWithSchema("atstarter_container_logs", "Read buffered Docker container logs, starting log follow first.", containerSchema(true)),
		toolWithSchema("atstarter_compose_services", "List Docker compose services for a project.", composeSchema(false)),
		toolWithSchema("atstarter_compose_up", "Run docker compose up -d for a project or service.", composeSchema(false)),
		toolWithSchema("atstarter_compose_stop", "Stop docker compose for a project or service.", composeSchema(false)),
		toolWithSchema("atstarter_compose_restart", "Restart docker compose for a project or service.", composeSchema(false)),
		toolWithSchema("atstarter_compose_down", "Run docker compose down for a project.", composeSchema(false)),
		toolWithSchema("atstarter_compose_logs", "Read buffered compose logs, starting log follow first.", composeSchema(true)),
	}
}

func toolNoArgs(name, description string) mcpTool {
	return toolWithSchema(name, description, objectSchema(nil, nil))
}

func toolWithSchema(name, description string, schema map[string]interface{}) mcpTool {
	return mcpTool{Name: name, Description: description, InputSchema: schema}
}

func projectSchema() map[string]interface{} {
	return objectSchema(map[string]interface{}{"project": stringProp("Project id or name.")}, []string{"project"})
}

func projectCommandSchema(withTail bool) map[string]interface{} {
	props := map[string]interface{}{
		"project": stringProp("Project id or name."),
		"command": stringProp("Command id or name. Omit for default."),
	}
	if withTail {
		props["tail"] = intProp("Maximum log lines to return.")
	}
	return objectSchema(props, []string{"project"})
}

func containerSchema(withTail bool) map[string]interface{} {
	props := map[string]interface{}{"id": stringProp("Container id.")}
	if withTail {
		props["tail"] = intProp("Maximum log lines to return.")
	}
	return objectSchema(props, []string{"id"})
}

func composeSchema(withTail bool) map[string]interface{} {
	props := map[string]interface{}{
		"project": stringProp("Project id or name."),
		"service": stringProp("Compose service name. Omit for all services."),
	}
	if withTail {
		props["tail"] = intProp("Maximum log lines to return.")
	}
	return objectSchema(props, []string{"project"})
}

func groupSaveSchema(existing bool) map[string]interface{} {
	props := map[string]interface{}{
		"name":  stringProp("Group name."),
		"items": arrayProp("Project command items.", objectSchema(map[string]interface{}{"project": stringProp("Project id or name."), "command": stringProp("Command id or name. Omit for default.")}, []string{"project"})),
	}
	required := []string{"name"}
	if existing {
		props["group"] = stringProp("Existing group id or name.")
		required = []string{"group"}
	}
	return objectSchema(props, required)
}

func groupItemSchema() map[string]interface{} {
	return objectSchema(map[string]interface{}{
		"group":   stringProp("Group id or name."),
		"project": stringProp("Project id or name."),
		"command": stringProp("Command id or name. Omit for default."),
	}, []string{"group", "project"})
}

func objectSchema(properties map[string]interface{}, required []string) map[string]interface{} {
	if properties == nil {
		properties = map[string]interface{}{}
	}
	schema := map[string]interface{}{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringProp(description string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": description}
}

func intProp(description string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description}
}

func boolProp(description string) map[string]interface{} {
	return map[string]interface{}{"type": "boolean", "description": description}
}

func arrayProp(description string, items map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"type": "array", "description": description, "items": items}
}

func mcpToolResult(data interface{}, err error) map[string]interface{} {
	env := cliEnvelope{OK: true, Data: data}
	if err != nil {
		env = cliEnvelope{OK: false, Error: controlError(err)}
	}
	b, _ := json.Marshal(env)
	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": string(b)}},
		"isError": err != nil,
	}
}

func stringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func boolArg(args map[string]interface{}, key string) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return false
}

func stringSliceArg(args map[string]interface{}, key string) []string {
	raw, ok := args[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func itemListArg(args map[string]interface{}, key string) []itemParams {
	raw, ok := args[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]itemParams, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, itemParams{Project: stringArg(m, "project"), Command: stringArg(m, "command")})
	}
	return out
}

func intArg(args map[string]interface{}, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

func durationArg(args map[string]interface{}, key string, fallback time.Duration) time.Duration {
	s := stringArg(args, key)
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func rawID(raw json.RawMessage) interface{} {
	var id interface{}
	if err := json.Unmarshal(raw, &id); err != nil {
		return string(raw)
	}
	return id
}

func mcpFailure(id interface{}, code int, message string) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Error: &mcpError{Code: code, Message: message}}
}

func readMCPMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, err
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, errors.New("missing Content-Length")
	}
	msg := make([]byte, contentLength)
	_, err := io.ReadFull(r, msg)
	return msg, err
}

func writeMCPMessage(w io.Writer, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(b))
	buf.Write(b)
	_, err = w.Write(buf.Bytes())
	return err
}
