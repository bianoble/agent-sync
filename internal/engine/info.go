package engine

import (
	"github.com/bianoble/agent-sync/internal/cache"
	"github.com/bianoble/agent-sync/internal/config"
	"github.com/bianoble/agent-sync/internal/target"
)

// ConfigLayerStatus describes a config layer's load status for display.
type ConfigLayerStatus struct {
	Level  string // "system", "user", "project"
	Path   string
	Loaded bool
}

// InfoResult holds tool information for the info command.
type InfoResult struct {
	Version     string
	ConfigPath  string
	LockPath    string
	CacheDir    string
	Tools       []ToolInfo
	ConfigChain []ConfigLayerStatus
	CacheSize   int64
	SpecVersion int
}

// ToolInfo describes a tool definition.
type ToolInfo struct {
	Name        string
	Destination string
	IsCustom    bool
}

// Info gathers tool information.
func Info(version string, cfg *config.Config, c *cache.Cache, tm *target.ToolMap, configPath, lockPath string) (*InfoResult, error) {
	r := &InfoResult{
		Version:     version,
		SpecVersion: 1,
		ConfigPath:  configPath,
		LockPath:    lockPath,
	}

	if c != nil {
		r.CacheDir = c.Path()
		size, err := c.Size()
		if err == nil {
			r.CacheSize = size
		}
	}

	if tm != nil {
		for _, name := range tm.KnownTools() {
			dest, _ := tm.Resolve(name)
			r.Tools = append(r.Tools, ToolInfo{
				Name:        name,
				Destination: dest,
				IsCustom:    tm.IsCustom(name),
			})
		}
	}

	return r, nil
}
