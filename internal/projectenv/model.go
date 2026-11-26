package projectenv

type Mode string

const (
	ModeInherit Mode = "inherit"
	ModeSet     Mode = "set"
	ModeClear   Mode = "clear"
)

var SupportedGitKeys = []string{
	"user.name",
	"user.email",
	"user.signingKey",
	"commit.gpgSign",
	"tag.gpgSign",
	"gpg.format",
}

var SupportedCodexKeys = []string{
	"model",
	"model_reasoning_effort",
	"sandbox_mode",
	"approval_policy",
	"web_search",
}

type FieldIntent struct {
	Mode  Mode   `json:"mode"`
	Value string `json:"value,omitempty"`
}

type FieldMap map[string]FieldIntent

type EnvironmentFields struct {
	Git   FieldMap `json:"git,omitempty"`
	Codex FieldMap `json:"codex,omitempty"`
}

type LayerConfig struct {
	PresetID string   `json:"presetId,omitempty"`
	Git      FieldMap `json:"git,omitempty"`
	Codex    FieldMap `json:"codex,omitempty"`
}

type Preset struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Git         FieldMap `json:"git,omitempty"`
	Codex       FieldMap `json:"codex,omitempty"`
}

type Rule struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Recursive    bool     `json:"recursive"`
	AutoDiscover bool     `json:"autoDiscover"`
	PresetID     string   `json:"presetId,omitempty"`
	Git          FieldMap `json:"git,omitempty"`
	Codex        FieldMap `json:"codex,omitempty"`
}

type Config struct {
	Version  int         `json:"version"`
	Defaults LayerConfig `json:"defaults"`
	Presets  []Preset    `json:"presets"`
	Rules    []Rule      `json:"rules"`
}

type Document struct {
	Config   Config `json:"config"`
	Revision string `json:"revision"`
}

type SourceKind string

const (
	SourceDefault SourceKind = "default"
	SourcePreset  SourceKind = "preset"
	SourceRule    SourceKind = "rule"
	SourceSession SourceKind = "session"
)

type Source struct {
	Kind SourceKind `json:"kind"`
	ID   string     `json:"id,omitempty"`
	Name string     `json:"name,omitempty"`
}

type ResolvedField struct {
	Intent FieldIntent `json:"intent"`
	Source Source      `json:"source"`
}

type ResolvedEnvironment struct {
	RepoPath        string                   `json:"repoPath"`
	MatchingRuleIDs []string                 `json:"matchingRuleIds"`
	Git             map[string]ResolvedField `json:"git"`
	Codex           map[string]ResolvedField `json:"codex"`
}
