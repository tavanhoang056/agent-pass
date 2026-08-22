package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"agpass/internal/ui"
)

const SkillContent = `---
name: agpass
description: Manage, check quotas, and switch user accounts/profiles for AI coding agents (Antigravity, Codex, etc.).
---

# agpass - AI Agent Account & Quota Switcher

Use ` + "`agpass`" + ` to manage, inspect, and switch accounts/credentials for AI coding agents.

## When to use
- When the current agent account has exhausted its quota or hit a rate limit.
- When switching between work, personal, or different client project profiles.
- When you need to inspect available accounts and remaining request allowances.

## CLI Commands for AI Agents (Headless)

### 1. List configured accounts
` + "```bash" + `
agpass list --json
` + "```" + `

### 2. Check current active account
` + "```bash" + `
agpass current antigravity
` + "```" + `

### 3. Check remaining quota
` + "```bash" + `
agpass quota antigravity --json
` + "```" + `

### 4. Switch account (Non-interactive)
` + "```bash" + `
agpass switch antigravity --to <account_name> --json
# or
agpass switch codex <account_name> --json
` + "```" + `

### 5. Add a new account
` + "```bash" + `
agpass add antigravity --name work --email user@example.com --non-interactive --json
` + "```" + `
`

var (
	skillGlobalFlag bool
)

var installSkillCmd = &cobra.Command{
	Use:   "install-skill",
	Short: "Install AI Agent skill for Antigravity and other agents",
	Long:  "Installs the agpass SKILL.md into global or workspace agent configuration roots so AI agents can discover and use agpass autonomously.",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to get user home directory: %v", err)))
			return nil
		}

		var targetPaths []string

		// Global Antigravity skill path
		globalSkillDir := filepath.Join(home, ".gemini", "config", "skills", "agpass")
		targetPaths = append(targetPaths, filepath.Join(globalSkillDir, "SKILL.md"))

		// Workspace skill path
		workspaceSkillDir := filepath.Join(".agents", "skills", "agpass")
		targetPaths = append(targetPaths, filepath.Join(workspaceSkillDir, "SKILL.md"))

		for _, path := range targetPaths {
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				continue
			}
			if err := os.WriteFile(path, []byte(SkillContent), 0644); err != nil {
				fmt.Print(ui.WarningMessage(fmt.Sprintf("Could not write skill to %s: %v", path, err)))
			} else {
				fmt.Print(ui.SuccessMessage(fmt.Sprintf("Installed agent skill to %s", path)))
			}
		}

		fmt.Println(ui.Accent.Render("\n  AI coding agents can now discover and run 'agpass' commands autonomously!"))
		return nil
	},
}

func init() {
	installSkillCmd.Flags().BoolVarP(&skillGlobalFlag, "global", "g", true, "Install skill to global agent config directory")
	rootCmd.AddCommand(installSkillCmd)
}