package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/agent-pass/agent-pass/internal/agents"
	"github.com/agent-pass/agent-pass/internal/config"
	"github.com/agent-pass/agent-pass/internal/ui"
)

var (
	addAccountName   string
	addEmail         string
	addConfigDir     string
	addJsonOutput    bool
	addNonInteractive bool
)

var addCmd = &cobra.Command{
	Use:   "add <agent> [account]",
	Short: "Add a new account for an agent",
	Long:  "Add a new account configuration for the specified AI agent. Supports interactive prompts or flags for headless execution.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]

		agentInfo, err := agents.GetAgent(agentName)
		if err != nil {
			if addJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": fmt.Sprintf("unknown agent: %s", agentName)})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Unknown agent: %s", agentName)))
			fmt.Printf("\n  Supported agents: ")
			for _, name := range agents.ListAgentNames() {
				fmt.Printf("%s ", ui.AgentName.Render(name))
			}
			fmt.Println()
			return nil
		}

		accountName := addAccountName
		if len(args) == 2 && accountName == "" {
			accountName = args[1]
		}

		email := addEmail
		configDir := addConfigDir
		if configDir == "" {
			configDir = agentInfo.GetConfigPath()
		}

		// Interactive mode if accountName is empty and not non-interactive
		if accountName == "" && !addNonInteractive {
			reader := bufio.NewReader(os.Stdin)

			header := ui.SectionHeader(ui.IconKey, fmt.Sprintf("Add Account · %s", agentInfo.DisplayName))
			fmt.Printf("\n%s\n\n", header)

			fmt.Print("  " + ui.Label.Render("Account name (e.g., work, personal): "))
			inputName, _ := reader.ReadString('\n')
			accountName = strings.TrimSpace(inputName)
			if accountName == "" {
				fmt.Print(ui.ErrorMessage("Account name cannot be empty"))
				return nil
			}

			fmt.Print("  " + ui.Label.Render("Email (optional): "))
			inputEmail, _ := reader.ReadString('\n')
			email = strings.TrimSpace(inputEmail)

			defaultDir := agentInfo.GetConfigPath()
			fmt.Printf("  "+ui.Label.Render("Config directory")+ui.Muted.Render(" [%s]")+": ", defaultDir)
			inputDir, _ := reader.ReadString('\n')
			inputDir = strings.TrimSpace(inputDir)
			if inputDir != "" {
				configDir = inputDir
			}
		}

		if accountName == "" {
			if addJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": "account name is required"})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage("Account name cannot be empty"))
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			if addJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		account := config.Account{
			Name:      accountName,
			ConfigDir: configDir,
			Email:     email,
		}

		cfg.AddAccount(agentName, account)

		if err := cfg.Save(); err != nil {
			if addJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"success": false, "error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to save config: %v", err)))
			return nil
		}

		if addJsonOutput {
			res, _ := json.Marshal(map[string]interface{}{
				"success": true,
				"agent":   agentName,
				"account": accountName,
				"message": fmt.Sprintf("Added account '%s' for %s", accountName, agentInfo.DisplayName),
			})
			fmt.Println(string(res))
			return nil
		}

		fmt.Print(ui.SuccessMessage(fmt.Sprintf("Added account '%s' for %s",
			accountName, agentInfo.DisplayName)))

		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addAccountName, "name", "n", "", "Account name (e.g., work, personal)")
	addCmd.Flags().StringVarP(&addEmail, "email", "e", "", "Email associated with the account")
	addCmd.Flags().StringVarP(&addConfigDir, "config-dir", "c", "", "Custom configuration directory for this profile")
	addCmd.Flags().BoolVar(&addJsonOutput, "json", false, "Output in JSON format")
	addCmd.Flags().BoolVar(&addNonInteractive, "non-interactive", false, "Disable interactive prompts")
	rootCmd.AddCommand(addCmd)
}