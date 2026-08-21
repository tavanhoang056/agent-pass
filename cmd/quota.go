package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"agent-pass/internal/config"
	"agent-pass/internal/quota"
	"agent-pass/internal/ui"
)

var (
	quotaJsonOutput   bool
	quotaSetTotal     int
	quotaSetUsed      int
	quotaSetResetIn   string
	quotaSetModel     string
)

type AccountQuotaJSON struct {
	AccountName string  `json:"account_name"`
	IsActive    bool    `json:"is_active"`
	Used        int     `json:"used"`
	Total       int     `json:"total"`
	Remaining   int     `json:"remaining"`
	Percent     float64 `json:"percent_remaining"`
	ResetsIn    string  `json:"resets_in"`
	Model       string  `json:"model,omitempty"`
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
					q, err := quota.CheckQuota(cfg, name, acc.Name)
					if err != nil {
						accQuotas = append(accQuotas, AccountQuotaJSON{
							AccountName: acc.Name,
							IsActive:    acc.Name == agent.Active,
							Error:       err.Error(),
						})
						continue
					}
					accQuotas = append(accQuotas, AccountQuotaJSON{
						AccountName: acc.Name,
						IsActive:    acc.Name == agent.Active,
						Used:        q.Used,
						Total:       q.Total,
						Remaining:   q.Remaining(),
						Percent:     q.Percent(),
						ResetsIn:    q.ResetsInString(),
						Model:       q.Model,
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

				q, err := quota.CheckQuota(cfg, name, acc.Name)
				if err != nil {
					output += fmt.Sprintf("  %s %s  %s\n",
						ui.Muted.Render(tree),
						ui.AccountInactive.Render(acc.Name),
						ui.Muted.Render(fmt.Sprintf("(%v)", err)),
					)
					continue
				}

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

			if !isLast {
				output += "\n"
			}
		}

		fmt.Print(ui.BoxBorder.Render(output))
		fmt.Println()
		return nil
	},
}

var quotaSetCmd = &cobra.Command{
	Use:   "set <agent> <account>",
	Short: "Configure or update quota allowance for an account",
	Long:  "Manually set the total requests, used requests, and reset duration for an account.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		accountName := args[1]

		cfg, err := config.Load()
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		var resetDuration time.Duration
		if quotaSetResetIn != "" {
			d, err := time.ParseDuration(quotaSetResetIn)
			if err != nil {
				fmt.Print(ui.ErrorMessage(fmt.Sprintf("Invalid reset duration '%s' (e.g. 24h, 2h30m): %v", quotaSetResetIn, err)))
				return nil
			}
			resetDuration = d
		} else {
			resetDuration = 24 * time.Hour
		}

		if quotaSetTotal <= 0 {
			quotaSetTotal = 300
		}

		if err := cfg.UpdateAccountQuota(agentName, accountName, quotaSetTotal, quotaSetUsed, resetDuration, quotaSetModel); err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to update quota: %v", err)))
			return nil
		}

		if err := cfg.Save(); err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to save config: %v", err)))
			return nil
		}

		fmt.Print(ui.SuccessMessage(fmt.Sprintf("Updated quota for %s/%s (%d/%d requests, resets in %v)",
			agentName, accountName, quotaSetTotal-quotaSetUsed, quotaSetTotal, resetDuration)))
		return nil
	},
}

var quotaResetCmd = &cobra.Command{
	Use:   "reset <agent> <account>",
	Short: "Reset used quota to 0 for an account",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		accountName := args[1]

		cfg, err := config.Load()
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		acc := cfg.GetAccount(agentName, accountName)
		if acc == nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Account '%s' not found for agent '%s'", accountName, agentName)))
			return nil
		}

		_ = cfg.UpdateAccountQuota(agentName, accountName, acc.TotalQuota, 0, 24*time.Hour, acc.QuotaModel)
		_ = cfg.Save()

		fmt.Print(ui.SuccessMessage(fmt.Sprintf("Reset used quota to 0 for %s/%s (100%% available)", agentName, accountName)))
		return nil
	},
}

func init() {
	quotaCmd.Flags().BoolVar(&quotaJsonOutput, "json", false, "Output quota in JSON format for automated scripts/AI agents")

	quotaSetCmd.Flags().IntVarP(&quotaSetTotal, "total", "t", 300, "Total request allowance (e.g. 300)")
	quotaSetCmd.Flags().IntVarP(&quotaSetUsed, "used", "u", 0, "Used requests (e.g. 50)")
	quotaSetCmd.Flags().StringVarP(&quotaSetResetIn, "reset", "r", "24h", "Time until reset (e.g. 4h, 24h, 30m)")
	quotaSetCmd.Flags().StringVarP(&quotaSetModel, "model", "m", "", "Associated model name (e.g. Gemini 3.7 Flash)")

	quotaCmd.AddCommand(quotaSetCmd)
	quotaCmd.AddCommand(quotaResetCmd)
	rootCmd.AddCommand(quotaCmd)
}