package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"agpass/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "agpass",
	Short: "AI Agent Account Switcher",
	Long:  "Seamless multi-account switcher for AI coding agents (Antigravity, Codex, etc.)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(ui.Banner())
		cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
