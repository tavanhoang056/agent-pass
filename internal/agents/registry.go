package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Agent represents a supported AI coding agent
type Agent struct {
	Name        string
	DisplayName string
	Description string
	ConfigPaths map[string]string // OS -> config path
	AuthFile    string
}

// Registry of all supported agents
var Registry = map[string]*Agent{
	"antigravity": {
		Name:        "antigravity",
		DisplayName: "Antigravity",
		Description: "Google Antigravity AI coding agent",
		ConfigPaths: map[string]string{
			"windows": filepath.Join(os.Getenv("USERPROFILE"), ".gemini"),
			"linux":   filepath.Join(os.Getenv("HOME"), ".gemini"),
			"darwin":  filepath.Join(os.Getenv("HOME"), ".gemini"),
		},
		AuthFile: "auth.json",
	},
	"codex": {
		Name:        "codex",
		DisplayName: "OpenAI Codex",
		Description: "OpenAI Codex CLI coding agent",
		ConfigPaths: map[string]string{
			"windows": filepath.Join(os.Getenv("USERPROFILE"), ".codex"),
			"linux":   filepath.Join(os.Getenv("HOME"), ".codex"),
			"darwin":  filepath.Join(os.Getenv("HOME"), ".codex"),
		},
		AuthFile: "auth.json",
	},
}

// GetAgent returns an agent by name
func GetAgent(name string) (*Agent, error) {
	agent, ok := Registry[name]
	if !ok {
		return nil, fmt.Errorf("unsupported agent: %s", name)
	}
	return agent, nil
}

// GetConfigPath returns the config path for the current OS
func (a *Agent) GetConfigPath() string {
	if p, ok := a.ConfigPaths[runtime.GOOS]; ok {
		return p
	}
	return ""
}

// ListAgentNames returns all registered agent names
func ListAgentNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// SwitchAccount performs the actual account switch for an agent
func (a *Agent) SwitchAccount(fromAccount, toAccount, backupDir string) error {
	configPath := a.GetConfigPath()
	if configPath == "" {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Create backup directory
	backupPath := filepath.Join(backupDir, a.Name, fromAccount)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}

	authFilePath := filepath.Join(configPath, a.AuthFile)

	// Backup current auth file if it exists
	if _, err := os.Stat(authFilePath); err == nil {
		data, err := os.ReadFile(authFilePath)
		if err != nil {
			return fmt.Errorf("failed to read current auth: %w", err)
		}
		backupFile := filepath.Join(backupPath, a.AuthFile)
		if err := os.WriteFile(backupFile, data, 0644); err != nil {
			return fmt.Errorf("failed to backup auth: %w", err)
		}
	}

	// Restore target account's auth file
	restorePath := filepath.Join(backupDir, a.Name, toAccount, a.AuthFile)
	if _, err := os.Stat(restorePath); err == nil {
		data, err := os.ReadFile(restorePath)
		if err != nil {
			return fmt.Errorf("failed to read target auth: %w", err)
		}
		if err := os.MkdirAll(configPath, 0755); err != nil {
			return fmt.Errorf("failed to create config dir: %w", err)
		}
		if err := os.WriteFile(authFilePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write auth: %w", err)
		}
	}

	return nil
}
