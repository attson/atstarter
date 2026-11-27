package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"rootyard/internal/projectenv"
	"rootyard/internal/projectenv/codexadapter"
	"rootyard/internal/projectenv/gitadapter"
)

type envErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type envEnvelope struct {
	OK    bool             `json:"ok"`
	Data  any              `json:"data,omitempty"`
	Error *envErrorPayload `json:"error,omitempty"`
}

type envCLIError struct {
	code    string
	message string
	detail  string
}

func (e envCLIError) Error() string { return e.message }

func envUsage(message string) error {
	return envCLIError{code: "usage", message: message}
}

func runEnvCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_ = stderr
	configPath, jsonMode, commandArgs, err := parseEnvironmentGlobalFlags(args)
	if err != nil {
		writeEnvironmentError(stdout, err)
		return 2
	}
	if len(commandArgs) == 0 {
		writeEnvironmentError(stdout, envUsage("environment command required"))
		return 2
	}
	if err := validateEnvironmentCommandShape(commandArgs); err != nil {
		writeEnvironmentError(stdout, err)
		return 2
	}

	service := projectenv.NewService(configPath, gitadapter.New(), codexadapter.New())
	data, err := dispatchEnvironmentCLI(context.Background(), service, commandArgs, stdin, stdout, jsonMode)
	if err != nil {
		writeEnvironmentErrorWithData(stdout, err, data)
		if environmentErrorCode(err) == "usage" {
			return 2
		}
		return 1
	}
	if data != nil {
		if jsonMode {
			writeEnvironmentJSON(stdout, envEnvelope{OK: true, Data: data})
		} else if !isRenderedEnvironmentOutput(data) {
			writeEnvironmentHuman(stdout, data)
		}
	}
	return 0
}

func parseEnvironmentGlobalFlags(args []string) (string, bool, []string, error) {
	configPath := defaultConfigPath()
	jsonMode := false
	rest := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		switch {
		case args[index] == "--config":
			if index+1 >= len(args) {
				return "", false, nil, envUsage("--config requires a path")
			}
			index++
			configPath = args[index]
		case strings.HasPrefix(args[index], "--config="):
			configPath = strings.TrimPrefix(args[index], "--config=")
		case args[index] == "--json":
			jsonMode = true
		default:
			rest = append(rest, args[index])
		}
	}
	return configPath, jsonMode, rest, nil
}

func validateEnvironmentCommandShape(args []string) error {
	switch args[0] {
	case "defaults", "preset", "rule":
		if len(args) < 2 {
			return envUsage(args[0] + " subcommand required")
		}
		allowed := map[string]map[string]bool{
			"defaults": {"get": true, "set": true},
			"preset":   {"list": true, "get": true, "set": true, "clone": true, "delete": true},
			"rule":     {"list": true, "get": true, "set": true, "delete": true},
		}
		if !allowed[args[0]][args[1]] {
			return envUsage("unknown " + args[0] + " subcommand: " + args[1])
		}
	case "scan", "resolve", "diff", "apply", "doctor":
		return nil
	default:
		return envUsage("unknown environment command: " + args[0])
	}
	return nil
}

func dispatchEnvironmentCLI(ctx context.Context, service *projectenv.Service, args []string, stdin io.Reader, stdout io.Writer, jsonMode bool) (any, error) {
	switch args[0] {
	case "defaults":
		return runEnvironmentDefaults(ctx, service, args[1:])
	case "preset":
		return runEnvironmentPreset(ctx, service, args[1:])
	case "rule":
		return runEnvironmentRule(ctx, service, args[1:])
	case "scan":
		return runEnvironmentScan(ctx, service, args[1:])
	case "resolve":
		return runEnvironmentResolve(ctx, service, args[1:])
	case "diff":
		plans, err := runEnvironmentPlan(ctx, service, args[1:], false)
		if err != nil {
			return nil, err
		}
		if !jsonMode {
			renderEnvironmentPlans(stdout, plans)
			return renderedEnvironmentOutput{}, nil
		}
		return plans, nil
	case "apply":
		return runEnvironmentApply(ctx, service, args[1:], stdin, stdout, jsonMode)
	case "doctor":
		return runEnvironmentDoctor(ctx, service, args[1:])
	default:
		return nil, envUsage("unknown environment command: " + args[0])
	}
}

func runEnvironmentDefaults(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	switch args[0] {
	case "get":
		if len(args) != 1 {
			return nil, envUsage("defaults get accepts no arguments")
		}
		doc, err := service.GetDocument(ctx)
		return doc.Config.Defaults, err
	case "set":
		fields, rest, err := parseEnvironmentFields(args[1:])
		if err != nil {
			return nil, err
		}
		var presetID *string
		for index := 0; index < len(rest); index++ {
			switch rest[index] {
			case "--preset":
				value, next, err := requiredEnvironmentValue(rest, index)
				if err != nil {
					return nil, err
				}
				presetID = &value
				index = next
			case "--no-preset":
				value := ""
				presetID = &value
			default:
				return nil, envUsage("unknown defaults set argument: " + rest[index])
			}
		}
		doc, err := service.GetDocument(ctx)
		if err != nil {
			return nil, err
		}
		return service.SaveDefaults(ctx, projectenv.SaveDefaultsRequest{
			Revision: doc.Revision, PresetID: presetID, Git: fields.Git, Codex: fields.Codex,
		})
	}
	return nil, envUsage("unknown defaults subcommand")
}

func runEnvironmentPreset(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return nil, envUsage("preset list accepts no arguments")
		}
		doc, err := service.GetDocument(ctx)
		return doc.Config.Presets, err
	case "get":
		if len(args) != 2 {
			return nil, envUsage("preset get requires one ID")
		}
		doc, err := service.GetDocument(ctx)
		if err != nil {
			return nil, err
		}
		for _, preset := range doc.Config.Presets {
			if preset.ID == args[1] {
				return preset, nil
			}
		}
		return nil, &projectenv.Error{Code: projectenv.ErrPresetNotFound, Message: "environment preset not found", Detail: args[1]}
	case "set":
		return runEnvironmentPresetSet(ctx, service, args[1:])
	case "clone":
		if len(args) < 2 {
			return nil, envUsage("preset clone requires an ID")
		}
		id := args[1]
		name := ""
		for index := 2; index < len(args); index++ {
			if args[index] != "--name" {
				return nil, envUsage("unknown preset clone argument: " + args[index])
			}
			value, next, err := requiredEnvironmentValue(args, index)
			if err != nil {
				return nil, err
			}
			name = value
			index = next
		}
		if name == "" {
			return nil, envUsage("preset clone requires --name")
		}
		doc, err := service.GetDocument(ctx)
		if err != nil {
			return nil, err
		}
		return service.ClonePreset(ctx, projectenv.ClonePresetRequest{Revision: doc.Revision, ID: id, Name: name})
	case "delete":
		if len(args) != 2 {
			return nil, envUsage("preset delete requires one ID")
		}
		doc, err := service.GetDocument(ctx)
		if err != nil {
			return nil, err
		}
		return service.DeletePreset(ctx, projectenv.DeletePresetRequest{Revision: doc.Revision, ID: args[1]})
	}
	return nil, envUsage("unknown preset subcommand")
}

func runEnvironmentPresetSet(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	fields, rest, err := parseEnvironmentFields(args)
	if err != nil {
		return nil, err
	}
	request := projectenv.SavePresetRequest{Git: fields.Git, Codex: fields.Codex}
	for index := 0; index < len(rest); index++ {
		value, next, err := requiredEnvironmentValue(rest, index)
		if err != nil {
			return nil, err
		}
		switch rest[index] {
		case "--id":
			request.ID = value
		case "--name":
			request.Name = &value
		case "--description":
			request.Description = &value
		default:
			return nil, envUsage("unknown preset set argument: " + rest[index])
		}
		index = next
	}
	if request.ID == "" && (request.Name == nil || *request.Name == "") {
		return nil, envUsage("new preset requires --name")
	}
	doc, err := service.GetDocument(ctx)
	if err != nil {
		return nil, err
	}
	request.Revision = doc.Revision
	return service.SavePreset(ctx, request)
}

func runEnvironmentRule(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return nil, envUsage("rule list accepts no arguments")
		}
		doc, err := service.GetDocument(ctx)
		return doc.Config.Rules, err
	case "get":
		if len(args) != 2 {
			return nil, envUsage("rule get requires one ID")
		}
		doc, err := service.GetDocument(ctx)
		if err != nil {
			return nil, err
		}
		for _, rule := range doc.Config.Rules {
			if rule.ID == args[1] {
				return rule, nil
			}
		}
		return nil, &projectenv.Error{Code: projectenv.ErrRuleNotFound, Message: "directory rule not found", Detail: args[1]}
	case "set":
		return runEnvironmentRuleSet(ctx, service, args[1:])
	case "delete":
		if len(args) != 2 {
			return nil, envUsage("rule delete requires one ID")
		}
		doc, err := service.GetDocument(ctx)
		if err != nil {
			return nil, err
		}
		return service.DeleteRule(ctx, projectenv.DeleteRuleRequest{Revision: doc.Revision, ID: args[1]})
	}
	return nil, envUsage("unknown rule subcommand")
}

func runEnvironmentRuleSet(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	fields, rest, err := parseEnvironmentFields(args)
	if err != nil {
		return nil, err
	}
	request := projectenv.SaveRuleRequest{Git: fields.Git, Codex: fields.Codex}
	for index := 0; index < len(rest); index++ {
		switch rest[index] {
		case "--recursive":
			value := true
			request.Recursive = &value
		case "--exact":
			value := false
			request.Recursive = &value
		case "--auto-discover":
			value := true
			request.AutoDiscover = &value
		case "--no-auto-discover":
			value := false
			request.AutoDiscover = &value
		case "--no-preset":
			value := ""
			request.PresetID = &value
		case "--id", "--name", "--path", "--preset":
			value, next, valueErr := requiredEnvironmentValue(rest, index)
			if valueErr != nil {
				return nil, valueErr
			}
			switch rest[index] {
			case "--id":
				request.ID = value
			case "--name":
				request.Name = &value
			case "--path":
				request.Path = &value
			case "--preset":
				request.PresetID = &value
			}
			index = next
		default:
			return nil, envUsage("unknown rule set argument: " + rest[index])
		}
	}
	if request.ID == "" && (request.Name == nil || request.Path == nil) {
		return nil, envUsage("new rule requires --name and --path")
	}
	doc, err := service.GetDocument(ctx)
	if err != nil {
		return nil, err
	}
	request.Revision = doc.Revision
	return service.SaveRule(ctx, request)
}

func runEnvironmentScan(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	ruleIDs := []string{}
	for index := 0; index < len(args); index++ {
		if args[index] != "--rule" {
			return nil, envUsage("unknown scan argument: " + args[index])
		}
		value, next, err := requiredEnvironmentValue(args, index)
		if err != nil {
			return nil, err
		}
		ruleIDs = append(ruleIDs, value)
		index = next
	}
	return service.Scan(ctx, projectenv.ScanRequest{RuleIDs: ruleIDs}, nil)
}

func runEnvironmentResolve(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	fields, rest, err := parseEnvironmentFields(args)
	if err != nil {
		return nil, err
	}
	if len(rest) != 1 || strings.HasPrefix(rest[0], "-") {
		return nil, envUsage("resolve requires one repository path")
	}
	return service.Resolve(ctx, projectenv.ResolveRequest{RepoPath: rest[0], Session: fields})
}

func runEnvironmentPlan(ctx context.Context, service *projectenv.Service, args []string, allowYes bool) ([]projectenv.ChangePlan, error) {
	fields, rest, err := parseEnvironmentFields(args)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	rules := []string{}
	for index := 0; index < len(rest); index++ {
		switch rest[index] {
		case "--rule", "--repo":
			value, next, valueErr := requiredEnvironmentValue(rest, index)
			if valueErr != nil {
				return nil, valueErr
			}
			if rest[index] == "--rule" {
				rules = append(rules, value)
			} else {
				paths = append(paths, value)
			}
			index = next
		case "--yes":
			if !allowYes {
				return nil, envUsage("--yes is only valid for apply")
			}
		default:
			if strings.HasPrefix(rest[index], "-") {
				return nil, envUsage("unknown plan argument: " + rest[index])
			}
			paths = append(paths, rest[index])
		}
	}
	if len(paths) == 0 && len(rules) == 0 {
		return nil, envUsage("repository path or --rule is required")
	}
	return service.Plan(ctx, projectenv.PlanRequest{RepositoryPaths: paths, RuleIDs: rules, Session: fields})
}

func runEnvironmentApply(ctx context.Context, service *projectenv.Service, args []string, stdin io.Reader, stdout io.Writer, jsonMode bool) (any, error) {
	yes := false
	acceptDangerous := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--yes" {
			yes = true
			continue
		}
		if arg == "--accept-dangerous" {
			acceptDangerous = true
			continue
		}
		filtered = append(filtered, arg)
	}
	plans, err := runEnvironmentPlan(ctx, service, filtered, true)
	if err != nil {
		return nil, err
	}
	interactive := !jsonMode && environmentReaderIsTerminal(stdin)
	if err := confirmEnvironmentApply(stdin, stdout, plans, yes, acceptDangerous, interactive); err != nil {
		return plans, err
	}
	results := service.Apply(ctx, plans)
	for _, result := range results {
		if result.Status == projectenv.ApplyFailed || result.Status == projectenv.ApplyBlocked || result.Status == projectenv.ApplyStale {
			return results, envCLIError{code: string(result.Status), message: "one or more environment targets were not applied"}
		}
	}
	return results, nil
}

func confirmEnvironmentApply(stdin io.Reader, stdout io.Writer, plans []projectenv.ChangePlan, yes, acceptDangerous, interactive bool) error {
	reader := bufio.NewReader(stdin)
	if !yes {
		if !interactive {
			return envCLIError{code: "confirmation_required", message: "apply requires --yes in non-interactive mode"}
		}
		renderEnvironmentPlans(stdout, plans)
		_, _ = fmt.Fprint(stdout, "Apply these changes? [y/N] ")
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			return envCLIError{code: "confirmation_required", message: "apply was not confirmed"}
		}
	}
	if plansHaveIssue(plans, projectenv.ErrDangerousCodexPermissions) {
		if !interactive && !acceptDangerous {
			return envCLIError{code: "dangerous_confirmation_required", message: "dangerous Codex permissions require --accept-dangerous"}
		}
		if !interactive {
			return nil
		}
		_, _ = fmt.Fprint(stdout, "Type APPLY DANGEROUS to allow Codex without sandboxing or approvals: ")
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if strings.TrimSpace(line) != "APPLY DANGEROUS" {
			return envCLIError{code: "dangerous_confirmation_required", message: "dangerous Codex permissions were not confirmed"}
		}
	}
	return nil
}

func runEnvironmentDoctor(ctx context.Context, service *projectenv.Service, args []string) (any, error) {
	fields, rest, err := parseEnvironmentFields(args)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	rules := []string{}
	for index := 0; index < len(rest); index++ {
		if rest[index] == "--rule" {
			value, next, err := requiredEnvironmentValue(rest, index)
			if err != nil {
				return nil, err
			}
			rules = append(rules, value)
			index = next
			continue
		}
		if strings.HasPrefix(rest[index], "-") {
			return nil, envUsage("unknown doctor argument: " + rest[index])
		}
		paths = append(paths, rest[index])
	}
	return service.Doctor(ctx, projectenv.DoctorRequest{RepositoryPaths: paths, RuleIDs: rules, Session: fields})
}

func parseEnvironmentFields(args []string) (projectenv.EnvironmentFields, []string, error) {
	fields := projectenv.EnvironmentFields{Git: projectenv.FieldMap{}, Codex: projectenv.FieldMap{}}
	rest := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg != "--git-set" && arg != "--git-clear" && arg != "--git-inherit" &&
			arg != "--codex-set" && arg != "--codex-clear" && arg != "--codex-inherit" {
			rest = append(rest, arg)
			continue
		}
		value, next, err := requiredEnvironmentValue(args, index)
		if err != nil {
			return projectenv.EnvironmentFields{}, nil, err
		}
		index = next
		target := fields.Git
		if strings.HasPrefix(arg, "--codex-") {
			target = fields.Codex
		}
		switch arg {
		case "--git-set", "--codex-set":
			key, fieldValue, ok := strings.Cut(value, "=")
			if !ok {
				return projectenv.EnvironmentFields{}, nil, envUsage(arg + " requires key=value")
			}
			target[key] = projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: fieldValue}
		case "--git-clear", "--codex-clear":
			target[value] = projectenv.FieldIntent{Mode: projectenv.ModeClear}
		case "--git-inherit", "--codex-inherit":
			target[value] = projectenv.FieldIntent{Mode: projectenv.ModeInherit}
		}
	}
	if err := projectenv.ValidateEnvironmentFields(fields); err != nil {
		return projectenv.EnvironmentFields{}, nil, err
	}
	return fields, rest, nil
}

func requiredEnvironmentValue(args []string, index int) (string, int, error) {
	if index+1 >= len(args) || strings.HasPrefix(args[index+1], "--") {
		return "", index, envUsage(args[index] + " requires a value")
	}
	return args[index+1], index + 1, nil
}

type renderedEnvironmentOutput struct{}

func isRenderedEnvironmentOutput(value any) bool {
	_, ok := value.(renderedEnvironmentOutput)
	return ok
}

func renderEnvironmentPlans(writer io.Writer, plans []projectenv.ChangePlan) {
	for _, plan := range plans {
		_, _ = fmt.Fprintf(writer, "[%s] %s -> %s\n", plan.Adapter, plan.Target.WorktreePath, plan.Target.ConfigPath)
		for _, change := range plan.Changes {
			if change.Operation == projectenv.OperationUnmanaged {
				continue
			}
			current := "(absent)"
			if len(change.Current) > 0 {
				values := make([]string, 0, len(change.Current))
				for _, value := range change.Current {
					values = append(values, value.Value)
				}
				current = strings.Join(values, ", ")
			}
			_, _ = fmt.Fprintf(writer, "  %s: %s -> %s (%s, source %s/%s)\n", change.Key, current, change.Desired.Value, change.Operation, change.Source.Kind, change.Source.ID)
		}
	}
}

func plansHaveIssue(plans []projectenv.ChangePlan, code projectenv.ErrorCode) bool {
	for _, plan := range plans {
		for _, issue := range append(append([]projectenv.Issue(nil), plan.Warnings...), plan.Blockers...) {
			if issue.Code == code {
				return true
			}
		}
	}
	return false
}

func environmentReaderIsTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func writeEnvironmentHuman(writer io.Writer, data any) {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(writer, data)
		return
	}
	_, _ = writer.Write(append(encoded, '\n'))
}

func writeEnvironmentJSON(writer io.Writer, envelope envEnvelope) {
	_ = json.NewEncoder(writer).Encode(envelope)
}

func writeEnvironmentError(writer io.Writer, err error) {
	writeEnvironmentErrorWithData(writer, err, nil)
}

func writeEnvironmentErrorWithData(writer io.Writer, err error, data any) {
	payload := envErrorPayload{Code: environmentErrorCode(err), Message: err.Error()}
	var projectError *projectenv.Error
	if errors.As(err, &projectError) {
		payload.Message = projectError.Message
		payload.Detail = projectError.Detail
	}
	var cliError envCLIError
	if errors.As(err, &cliError) {
		payload.Message = cliError.message
		payload.Detail = cliError.detail
	}
	writeEnvironmentJSON(writer, envEnvelope{OK: false, Data: data, Error: &payload})
}

func environmentErrorCode(err error) string {
	var projectError *projectenv.Error
	if errors.As(err, &projectError) {
		return string(projectError.Code)
	}
	var cliError envCLIError
	if errors.As(err, &cliError) {
		return cliError.code
	}
	return "internal_error"
}
