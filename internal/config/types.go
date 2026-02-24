package config

// Config represents the agent-sync.yaml configuration file.
// See spec Section 3.
type Config struct {
	Version         int               `yaml:"version"`
	Variables       map[string]string `yaml:"variables,omitempty"`
	Sources         []Source          `yaml:"sources"`
	Targets         []Target          `yaml:"targets"`
	Overrides       []Override        `yaml:"overrides,omitempty"`
	Transforms      []Transform       `yaml:"transforms,omitempty"`
	ToolDefinitions []ToolDefinition  `yaml:"tool_definitions,omitempty"`
}

// Source defines an external source of agent files.
// See spec Section 5.
type Source struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // "git", "url", "local"

	// Git source fields (Section 5.1).
	Repo  string   `yaml:"repo,omitempty"`
	Ref   string   `yaml:"ref,omitempty"`
	Paths []string `yaml:"paths,omitempty"`

	// URL source fields (Section 5.2).
	URL      string `yaml:"url,omitempty"`
	Checksum string `yaml:"checksum,omitempty"`

	// Local source fields (Section 5.3).
	Path string `yaml:"path,omitempty"`
}

// Target defines where source files are written.
// See spec Section 7.
type Target struct {
	Source      string   `yaml:"source"`
	Tools       []string `yaml:"tools,omitempty"`
	Destination string   `yaml:"destination,omitempty"`
}

// Override defines a post-sync modification to a target file.
// See spec Section 6.2.
type Override struct {
	Target   string `yaml:"target"`
	Strategy string `yaml:"strategy"` // "append", "prepend", "replace"
	File     string `yaml:"file"`
}

// Transform defines a transformation applied to source files.
// See spec Section 6.
type Transform struct {
	Source     string            `yaml:"source"`
	Type       string            `yaml:"type"` // "template", "custom"
	Vars       map[string]string `yaml:"vars,omitempty"`
	Command    string            `yaml:"command,omitempty"`
	OutputHash string            `yaml:"output_hash,omitempty"`
}

// ToolDefinition defines a custom tool path mapping or overrides a built-in.
// See spec Section 3.2.
type ToolDefinition struct {
	Name        string `yaml:"name"`
	Destination string `yaml:"destination"`
}
