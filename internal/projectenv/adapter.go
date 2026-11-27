package projectenv

import "context"

type Adapter interface {
	Name() string
	Identify(context.Context, string) (Target, error)
	Inspect(context.Context, Target) (Snapshot, error)
	Plan(ResolvedEnvironment, Snapshot) ChangePlan
	Apply(context.Context, ChangePlan) ApplyResult
	Doctor(context.Context, Target, ResolvedEnvironment) []Issue
}

type Target struct {
	Adapter      string `json:"adapter"`
	ID           string `json:"id"`
	WorktreePath string `json:"worktreePath"`
	CommonDir    string `json:"commonDir"`
	ConfigPath   string `json:"configPath"`
}

type OriginValue struct {
	Value      string `json:"value"`
	Origin     string `json:"origin"`
	MainConfig bool   `json:"mainConfig"`
}

type Snapshot struct {
	Target        Target                   `json:"target"`
	Values        map[string][]OriginValue `json:"values"`
	Fingerprint   string                   `json:"fingerprint"`
	Issues        []Issue                  `json:"issues"`
	Exists        bool                     `json:"exists"`
	NativeContent []byte                   `json:"-"`
}

type Operation string

const (
	OperationUnmanaged Operation = "unmanaged"
	OperationUnchanged Operation = "unchanged"
	OperationSet       Operation = "set"
	OperationReplace   Operation = "replace"
	OperationClear     Operation = "clear"
	OperationBlocked   Operation = "blocked"
)

type Change struct {
	Key       string        `json:"key"`
	Operation Operation     `json:"operation"`
	Current   []OriginValue `json:"current"`
	Desired   FieldIntent   `json:"desired"`
	Source    Source        `json:"source"`
	Issues    []Issue       `json:"issues,omitempty"`
}

type ChangePlan struct {
	Adapter         string                 `json:"adapter"`
	Target          Target                 `json:"target"`
	RepoPath        string                 `json:"repoPath"`
	Aliases         []RepositoryAlias      `json:"aliases,omitempty"`
	MatchingRuleIDs []string               `json:"matchingRuleIds,omitempty"`
	Snapshot        Snapshot               `json:"snapshot"`
	Fingerprint     string                 `json:"fingerprint"`
	Desired         map[string]FieldIntent `json:"desired"`
	Changes         []Change               `json:"changes"`
	Warnings        []Issue                `json:"warnings,omitempty"`
	Blockers        []Issue                `json:"blockers,omitempty"`
}

type ApplyStatus string

const (
	ApplySuccess   ApplyStatus = "success"
	ApplyUnchanged ApplyStatus = "unchanged"
	ApplySkipped   ApplyStatus = "skipped"
	ApplyStale     ApplyStatus = "stale"
	ApplyBlocked   ApplyStatus = "blocked"
	ApplyFailed    ApplyStatus = "failed"
)

type ApplyResult struct {
	Target      Target      `json:"target"`
	Status      ApplyStatus `json:"status"`
	Fingerprint string      `json:"fingerprint,omitempty"`
	Changes     []Change    `json:"changes,omitempty"`
	Issues      []Issue     `json:"issues,omitempty"`
}
