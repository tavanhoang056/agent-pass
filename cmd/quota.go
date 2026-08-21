package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/agent-pass/agent-pass/internal/config"
	"github.com/agent-pass/agent-pass/internal/quota"
	"github.com/agent-pass/agent-pass/internal/ui"
)

var quotaJsonOutput bool

type AccountQuotaJSON struct {
	AccountName string  `json:"account_name"`
	IsActive    bool    `json:"is_active"`
	Used        int     `json:"used"`
	Total       int     `json:"total"`
	Remaining   int     `json:"remaining"`
	Percent     float64 `json:"percent_remaining"`
	ResetsIn    string  `json:"resets_in"`
	Error       string  `json:"error,omitempty"`
}

type AgentQuotaJSON struct {
	Agent    string             `json:"agent"`
	Accounts []AccountQuotaJSON `json:"accounts"`
}

var quotaCmd = &cobra.Command{
	Use:   "quota [agent]",
	Short: "Check remaining quota for agent accounts",
	Long:  "Display quota usage, remaining requests, and reset timers for all or a specific AI agent.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			if quotaJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		if len(cfg.Agents) == 0 {
			if quotaJsonOutput {
				fmt.Println("[]")
				return nil
			}
			fmt.Print(ui.WarningMessage("No agents configured yet. Use 'agpass add <agent>' to add one."))
			return nil
		}

		agentNames := cfg.ListAgents()
		sort.Strings(agentNames)

		// Filter by agent if specified
		if len(args) > 0 {
			filterAgent := args[0]
			if cfg.GetAgent(filterAgent) == nil {
				if quotaJsonOutput {
					res, _ := json.Marshal(map[string]interface{}{"error": fmt.Sprintf("agent '%s' not found", filterAgent)})
					fmt.Println(string(res))
					return nil
				}
				fmt.Print(ui.ErrorMessage(fmt.Sprintf("Agent '%s' not found", filterAgent)))
				return nil
			}
			agentNames = []string{filterAgent}
		}

		if quotaJsonOutput {
			var results []AgentQuotaJSON
			for _, name := range agentNames {
				agent := cfg.GetAgent(name)
				var accQuotas []AccountQuotaJSON
				for _, acc := range agent.Accounts {
					q, _ := quota.CheckQuota(name, acc.Name)
					accQuotas = append(accQuotas, AccountQuotaJSON{
						AccountName: acc.Name,
						IsActive:    acc.Name == agent.Active,
						Used:        q.Used,
						Total:       q.Total,
						Remaining:   q.Remaining(),
						Percent:     q.Percent(),
						ResetsIn:    q.ResetsInString(),
						Error:       q.Error,
					})
				}
				results = append(results, AgentQuotaJSON{
					Agent:    name,
					Accounts: accQuotas,
				})
			}
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		header := ui.SectionHeader(ui.IconQuota, "Quota Overview")
		output := "\n" + header + "\n\n"

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

				q, _ := quota.CheckQuota(name, acc.Name)

				if q.Error != "" {
					output += fmt.Sprintf("  %s %s  %s\n",
						ui.Muted.Render(tree),
						ui.AccountInactive.Render(acc.Name),
						ui.Muted.Render("(quota API pending)"),
					)
				} else {
					percent := q.Percent()
					bar := ui.ProgressBar(percent, 12)
					percentStr := fmt.Sprintf("%.0f%% remaining", percent)

					var percentStyle func(string) string
					switch {
					case percent >= 70:
						percentStyle = func(s string) string { return ui.Success.Render(s) }
					case percent >= 30:
						percentStyle = func(s string) string { return ui.Warning.Render(s) }
					default:
						percentStyle = func(s string) string { return ui.Danger.Render(s) }
					}

					accName := ui.AccountInactive.Render(acc.Name)
					if acc.Name == agent.Active {
						accName = ui.AccountActive.Render(acc.Name)
					}

					output += fmt.Sprintf("  %s %s  %s  %s\n",
						ui.Muted.Render(tree),
						accName,
						bar,
						percentStyle(percentStr),
					)

					detailTree := ui.TreeSpace
					if !isLastAcc {
						detailTree = ui.TreePipe
					}
					output += fmt.Sprintf("  %s %s\n",
						ui.Muted.Render(detailTree),
						ui.Muted.Render(fmt.Sprintf("  %d / %d requests · Resets in %s",
							q.Remaining(), q.Total, q.ResetsInString())),
					)
				}
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
	quotaCmd.Flags().BoolVar(&quotaJsonOutput, "json", false, "Output quota in JSON format for automated scripts/AI agents")
	rootCmd.AddCommand(quotaCmd)
}