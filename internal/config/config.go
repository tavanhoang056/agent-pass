package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Account struct {
	Name      string `yaml:"name"`
	ConfigDir string `yaml:"config_dir"`
	Email     string `yaml:"email,omitempty"`
	TokenFile string `yaml:"token_file,omitempty"`
}

type AgentConfig struct {
	Active   string    `yaml:"active"`
	Accounts []Account `yaml:"accounts"`
}

type Config struct {
	Agents map[string]*AgentConfig `yaml:"agents"`
}

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".agpass"
	}
	return filepath.Join(home, ".agpass")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Agents: make(map[string]*AgentConfig)}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]*AgentConfig)
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}

func (c *Config) GetAgent(name string) *AgentConfig {
	if agent, ok := c.Agents[name]; ok {
		return agent
	}
	return nil
}

func (c *Config) SetActiveAccount(agentName, accountName string) error {
	agent := c.GetAgent(agentName)
	if agent == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}
	found := false
	for _, acc := range agent.Accounts {
		if acc.Name == accountName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("account '%s' not found for agent '%s'", accountName, agentName)
	}
	agent.Active = accountName
	return nil
}

func (c *Config) AddAccount(agentName string, account Account) {
	if c.Agents[agentName] == nil {
		c.Agents[agentName] = &AgentConfig{
			Active:   account.Name,
			Accounts: []Account{account},
		}
		return
	}
	for i, acc := range c.Agents[agentName].Accounts {
		if acc.Name == account.Name {
			c.Agents[agentName].Accounts[i] = account
			return
		}
	}
	c.Agents[agentName].Accounts = append(c.Agents[agentName].Accounts, account)
}

func (c *Config) ListAgents() []string {
	agents := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		agents = append(agents, name)
	}
	return agents
}