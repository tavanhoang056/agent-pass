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
	Use:   "remove <agent> [account]",
	Short: "Remove an account or an entire agent configuration",
	Long:  "Remove an existing account profile from an agent, or remove the entire agent if no account is specified.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]

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

		// If only agentName is passed, remove the entire agent
		if len(args) == 1 {
			delete(cfg.Agents, agentName)
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
					"message": fmt.Sprintf("Removed entire agent '%s'", agentName),
				})
				fmt.Println(string(res))
				return nil
			}

			fmt.Print(ui.SuccessMessage(fmt.Sprintf("Removed entire agent '%s'", agentName)))
			return nil
		}

		accountName := args[1]
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

		agent.Accounts = append(agent.Accounts[:foundIndex], agent.Accounts[foundIndex+1:]...)

		if len(agent.Accounts) == 0 {
			delete(cfg.Agents, agentName)
		} else if agent.Active == accountName {
			agent.Active = agent.Accounts[0].Name
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