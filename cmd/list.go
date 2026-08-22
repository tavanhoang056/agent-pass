package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"agpass/internal/config"
	"agpass/internal/ui"
)

var listJsonOutput bool

type ListAccountOutput struct {
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	Email     string `json:"email,omitempty"`
	ConfigDir string `json:"config_dir"`
}

type ListAgentOutput struct {
	Agent    string              `json:"agent"`
	Active   string              `json:"active_account"`
	Accounts []ListAccountOutput `json:"accounts"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured agents and accounts",
	Long:  "List all configured AI coding agents and their associated accounts. Supports table/tree UI and JSON output.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			if listJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		if listJsonOutput {
			var agentsOut []ListAgentOutput
			agentNames := cfg.ListAgents()
			sort.Strings(agentNames)

			for _, name := range agentNames {
				agent := cfg.GetAgent(name)
				var accs []ListAccountOutput
				for _, acc := range agent.Accounts {
					accs = append(accs, ListAccountOutput{
						Name:      acc.Name,
						IsActive:  acc.Name == agent.Active,
						Email:     acc.Email,
						ConfigDir: acc.ConfigDir,
					})
				}
				agentsOut = append(agentsOut, ListAgentOutput{
					Agent:    name,
					Active:   agent.Active,
					Accounts: accs,
				})
			}
			data, _ := json.MarshalIndent(agentsOut, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(cfg.Agents) == 0 {
			fmt.Print(ui.WarningMessage("No agents configured yet. Use 'agpass add <agent>' to add one."))
			return nil
		}

		header := ui.SectionHeader(ui.IconAgent, "Configured Agents")
		output := "\n" + header + "\n\n"

		agentNames := cfg.ListAgents()
		sort.Strings(agentNames)

		for i, name := range agentNames {
			agent := cfg.GetAgent(name)
			isLast := i == len(agentNames)-1

			output += "  " + ui.AgentName.Render(name) + "\n"

			for j, acc := range agent.Accounts {
				isLastAcc := j == len(agent.Accounts)-1
				tree := ui.TreeBranch
				if isLastAcc {
					tree = ui.TreeLast
				}

				accStyle := ui.AccountInactive
				suffix := ""
				indicator := ui.IconDotOpen

				if acc.Name == agent.Active {
					accStyle = ui.AccountActive
					suffix = ui.Muted.Render("  ← active")
					indicator = ui.Success.Render(ui.IconDot)
				}

				output += fmt.Sprintf("  %s %s %s%s\n",
					ui.Muted.Render(tree),
					indicator,
					accStyle.Render(acc.Name),
					suffix,
				)
			}

			if !isLast {
				output += "\n"
			}
		}

		fmt.Print(ui.BoxBorder.Render(output))
		fmt.Println()
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJsonOutput, "json", false, "Output list in JSON format for automated scripts/AI agents")
	rootCmd.AddCommand(listCmd)
}