package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"agent-pass/internal/agents"
	"agent-pass/internal/config"
	"agent-pass/internal/ui"
)

var (
	switchTargetAccount string
	switchJsonOutput    bool
)

type SwitchResult struct {
	Success     bool   `json:"success"`
	Agent       string `json:"agent"`
	PreviousAcc string `json:"previous_account"`
	CurrentAcc  string `json:"current_account"`
	Message     string `json:"message"`
}

var switchCmd = &cobra.Command{
	Use:   "switch <agent> [account]",
	Short: "Switch account for an AI agent (interactive or headless)",
	Long: `Switch to a different account for the specified AI agent.
Can be run interactively (with arrow keys) or in headless mode by providing the account name or using --to / --account flag.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]

		agentInfo, err := agents.GetAgent(agentName)
		if err != nil {
			if switchJsonOutput {
				res, _ := json.Marshal(SwitchResult{Success: false, Agent: agentName, Message: fmt.Sprintf("unknown agent: %s", agentName)})
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

		cfg, err := config.Load()
		if err != nil {
			if switchJsonOutput {
				res, _ := json.Marshal(SwitchResult{Success: false, Agent: agentName, Message: fmt.Sprintf("failed to load config: %v", err)})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		agentCfg := cfg.GetAgent(agentName)
		if agentCfg == nil || len(agentCfg.Accounts) == 0 {
			msg := fmt.Sprintf("No accounts configured for %s. Use 'agpass add %s' to add one.", agentInfo.DisplayName, agentName)
			if switchJsonOutput {
				res, _ := json.Marshal(SwitchResult{Success: false, Agent: agentName, Message: msg})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.WarningMessage(msg))
			return nil
		}

		// Determine target account from args or flag
		targetAccount := switchTargetAccount
		if len(args) == 2 && targetAccount == "" {
			targetAccount = args[1]
		}

		// If target account is specified directly, run in headless mode
		if targetAccount != "" {
			matchedAccount := ""
			for _, acc := range agentCfg.Accounts {
				if acc.Name == targetAccount {
					matchedAccount = acc.Name
					break
				}
			}
			if matchedAccount == "" {
				lowerTarget := strings.ToLower(targetAccount)
				for _, acc := range agentCfg.Accounts {
					if strings.Contains(strings.ToLower(acc.Name), lowerTarget) {
						matchedAccount = acc.Name
						break
					}
				}
			}

			if matchedAccount == "" {
				msg := fmt.Sprintf("Account '%s' not found for agent '%s'", targetAccount, agentName)
				if switchJsonOutput {
					res, _ := json.Marshal(SwitchResult{Success: false, Agent: agentName, Message: msg})
					fmt.Println(string(res))
					return nil
				}
				fmt.Print(ui.ErrorMessage(msg))
				return nil
			}

			prevAccount := agentCfg.Active
			backupDir := config.ConfigDir()
			if err := agentInfo.SwitchAccount(agentCfg.Active, matchedAccount, backupDir); err != nil {
				msg := fmt.Sprintf("Switch failed: %v", err)
				if switchJsonOutput {
					res, _ := json.Marshal(SwitchResult{Success: false, Agent: agentName, PreviousAcc: prevAccount, Message: msg})
					fmt.Println(string(res))
					return nil
				}
				fmt.Print(ui.ErrorMessage(msg))
				return nil
			}

			_ = cfg.SetActiveAccount(agentName, matchedAccount)
			_ = cfg.Save()

			if switchJsonOutput {
				res, _ := json.Marshal(SwitchResult{
					Success:     true,
					Agent:       agentName,
					PreviousAcc: prevAccount,
					CurrentAcc:  matchedAccount,
					Message:     fmt.Sprintf("Successfully switched %s to %s", agentInfo.DisplayName, matchedAccount),
				})
				fmt.Println(string(res))
				return nil
			}

			fmt.Print(ui.SuccessMessage(fmt.Sprintf("Switched %s to %s", agentInfo.DisplayName, matchedAccount)))
			return nil
		}

		// Interactive Mode (Bubbletea TUI)
		if len(agentCfg.Accounts) == 1 {
			fmt.Print(ui.WarningMessage(fmt.Sprintf("Only one account configured for %s. Nothing to switch.", agentInfo.DisplayName)))
			return nil
		}

		items := make([]ui.AccountItem, len(agentCfg.Accounts))
		for i, acc := range agentCfg.Accounts {
			items[i] = ui.AccountItem{
				Name:     acc.Name,
				IsActive: acc.Name == agentCfg.Active,
			}
		}

		selector := ui.NewSelector(agentInfo.DisplayName, items)
		p := tea.NewProgram(selector)
		result, err := p.Run()
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Selector error: %v", err)))
			return nil
		}

		model := result.(ui.SelectorModel)
		if model.Quitting {
			fmt.Print(ui.Muted.Render("\n  Cancelled.\n"))
			return nil
		}

		selected := model.GetSelectedAccount()
		if selected == "" || selected == agentCfg.Active {
			fmt.Print(ui.Muted.Render("\n  No change.\n"))
			return nil
		}

		backupDir := config.ConfigDir()
		if err := agentInfo.SwitchAccount(agentCfg.Active, selected, backupDir); err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Switch failed: %v", err)))
			return nil
		}

		_ = cfg.SetActiveAccount(agentName, selected)
		_ = cfg.Save()

		fmt.Print(ui.SuccessMessage(fmt.Sprintf("Switched %s to %s", agentInfo.DisplayName, selected)))
		return nil
	},
}

func init() {
	switchCmd.Flags().StringVarP(&switchTargetAccount, "to", "t", "", "Target account name to switch to (headless mode)")
	switchCmd.Flags().StringVarP(&switchTargetAccount, "account", "a", "", "Target account name to switch to (alias for --to)")
	switchCmd.Flags().BoolVar(&switchJsonOutput, "json", false, "Output result in JSON format for automated scripts/AI agents")
	rootCmd.AddCommand(switchCmd)
}