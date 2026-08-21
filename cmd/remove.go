package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"agent-pass/internal/config"
	"agent-pass/internal/ui"
)

var removeJsonOutput bool

var removeCmd = &cobra.Command{
	Use:   "remove <agent> <account>",
	Short: "Remove an account configuration",
	Long:  "Remove an existing account profile from an agent.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		accountName := args[1]

		cfg, err := config.Load()
		if err != nil {
			if removeJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		agent := cfg.GetAgent(agentName)
		if agent == nil {
			if removeJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": fmt.Sprintf("agent '%s' not found", agentName)})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Agent '%s' not found", agentName)))
			return nil
		}

		foundIndex := -1
		for i, acc := range agent.Accounts {
			if acc.Name == accountName {
				foundIndex = i
				break
			}
		}

		if foundIndex == -1 {
			if removeJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": fmt.Sprintf("account '%s' not found", accountName)})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Account '%s' not found for agent '%s'", accountName, agentName)))
			return nil
		}

		// Remove account
		agent.Accounts = append(agent.Accounts[:foundIndex], agent.Accounts[foundIndex+1:]...)

		// If removed account was active, switch active to first available
		if agent.Active == accountName {
			if len(agent.Accounts) > 0 {
				agent.Active = agent.Accounts[0].Name
			} else {
				agent.Active = ""
			}
		}

		if err := cfg.Save(); err != nil {
			if removeJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to save config: %v", err)))
			return nil
		}

		if removeJsonOutput {
			res, _ := json.Marshal(map[string]interface{}{
				"success": true,
				"agent":   agentName,
				"removed": accountName,
				"active":  agent.Active,
				"message": fmt.Sprintf("Removed account '%s' from %s", accountName, agentName),
			})
			fmt.Println(string(res))
			return nil
		}

		fmt.Print(ui.SuccessMessage(fmt.Sprintf("Removed account '%s' from %s", accountName, agentName)))
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVar(&removeJsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(removeCmd)
}