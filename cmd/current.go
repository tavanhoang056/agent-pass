package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/agent-pass/agent-pass/internal/config"
	"github.com/agent-pass/agent-pass/internal/ui"
)

var currentJsonOutput bool

var currentCmd = &cobra.Command{
	Use:   "current <agent>",
	Short: "Show active account for an agent",
	Long:  "Print the currently active account name for the specified agent.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]

		cfg, err := config.Load()
		if err != nil {
			if currentJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		agentCfg := cfg.GetAgent(agentName)
		if agentCfg == nil {
			if currentJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"error": fmt.Sprintf("agent '%s' not found", agentName)})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Agent '%s' not configured", agentName)))
			return nil
		}

		if currentJsonOutput {
			res, _ := json.Marshal(map[string]interface{}{
				"agent":          agentName,
				"active_account": agentCfg.Active,
			})
			fmt.Println(string(res))
			return nil
		}

		fmt.Println(agentCfg.Active)
		return nil
	},
}

func init() {
	currentCmd.Flags().BoolVar(&currentJsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(currentCmd)
}