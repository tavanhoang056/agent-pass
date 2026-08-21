package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"agent-pass/internal/config"
	"agent-pass/internal/quota"
	"agent-pass/internal/ui"
)

var (
	quotaJsonOutput bool
	quotaSetTotal   int
	quotaSetUsed    int
	quotaSetResetIn string
	quotaSetModel   string
)

var quotaCmd = &cobra.Command{
	Use:   "quota [agent]",
	Short: "Check remaining quota for agent accounts",
	Long:  "Inspect live API quota, rolling rate-limit windows (5h/weekly/monthly), and status for all or a specific AI coding agent.",
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

		var reports []*quota.AgentQuotaReport
		for _, name := range agentNames {
			agent := cfg.GetAgent(name)
			for _, acc := range agent.Accounts {
				rep, err := quota.CheckAgentQuota(cfg, name, acc.Name)
				if err != nil {
					reports = append(reports, &quota.AgentQuotaReport{
						AgentName:   name,
						AccountName: acc.Name,
						Error:       err.Error(),
					})
					continue
				}
				reports = append(reports, rep)
			}
		}

		if quotaJsonOutput {
			data, _ := json.MarshalIndent(reports, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Render UI
		header := ui.SectionHeader(ui.IconQuota, "Quota & Rate Limit Overview")
		output := "\n" + header + "\n\n"

		// Group reports by Agent
		agentGroups := make(map[string][]*quota.AgentQuotaReport)
		for _, rep := range reports {
			agentGroups[rep.AgentName] = append(agentGroups[rep.AgentName], rep)
		}

		for i, name := range agentNames {
			group := agentGroups[name]
			isLastAgent := i == len(agentNames)-1

			output += "  " + ui.AgentName.Render(name) + "\n"

			for j, rep := range group {
				isLastAcc := j == len(group)-1
				tree := ui.TreeBranch
				if isLastAcc {
					tree = ui.TreeLast
				}

				accLabel := ui.AccountInactive.Render(rep.AccountName)
				if rep.IsActive {
					accLabel = ui.AccountActive.Render(rep.AccountName) + ui.Muted.Render(" (active)")
				}

				// Badges
				badge := ""
				if rep.IsLiveAPI {
					badge = " " + lipgloss.NewStyle().
						Foreground(ui.ColorSuccess).
						Background(lipgloss.Color("#132A1C")).
						Padding(0, 1).
						Render("LIVE API")
				}
				if rep.PlanType != "" {
					badge += " " + lipgloss.NewStyle().
						Foreground(ui.ColorSecondary).
						Background(lipgloss.Color("#162B34")).
						Padding(0, 1).
						Render(rep.PlanType)
				}

				emailText := ""
				if rep.AccountEmail != "" && rep.AccountEmail != rep.AccountName {
					emailText = " " + ui.Muted.Render(fmt.Sprintf("<%s>", rep.AccountEmail))
				}

				output += fmt.Sprintf("  %s %s%s%s\n",
					ui.Muted.Render(tree),
					accLabel,
					badge,
					emailText,
				)

				pipe := ui.TreePipe
				if isLastAcc {
					pipe = ui.TreeSpace
				}

				if rep.Error != "" {
					output += fmt.Sprintf("  %s   %s\n",
						ui.Muted.Render(pipe),
						ui.Warning.Render("⚠ "+rep.Error),
					)
				} else if len(rep.Windows) > 0 {
					for _, win := range rep.Windows {
						bar := ui.ProgressBar(win.RemainingPercent, 14)
						pctStr := fmt.Sprintf("%.0f%% remaining", win.RemainingPercent)

						var pctStyle func(string) string
						switch {
						case win.RemainingPercent >= 70:
							pctStyle = func(s string) string { return ui.Success.Render(s) }
						case win.RemainingPercent >= 30:
							pctStyle = func(s string) string { return ui.Warning.Render(s) }
						default:
							pctStyle = func(s string) string { return ui.Danger.Render(s) }
						}

						resetStr := ""
						if win.ResetsIn > 0 {
							days := int(win.ResetsIn.Hours() / 24)
							hours := int(win.ResetsIn.Hours()) % 24
							mins := int(win.ResetsIn.Minutes()) % 60

							if days > 0 {
								resetStr = fmt.Sprintf(" · Resets in %dd %dh", days, hours)
							} else if hours > 0 {
								resetStr = fmt.Sprintf(" · Resets in %dh %dm", hours, mins)
							} else {
								resetStr = fmt.Sprintf(" · Resets in %dm", mins)
							}
						}

						countsStr := ""
						if win.TotalCount > 0 {
							countsStr = fmt.Sprintf(" (%d / %d requests)", win.TotalCount-win.UsedCount, win.TotalCount)
						}

						output += fmt.Sprintf("  %s   %s: %s  %s%s%s\n",
							ui.Muted.Render(pipe),
							ui.Subtitle.Render(win.Name),
							bar,
							pctStyle(pctStr),
							countsStr,
							ui.Muted.Render(resetStr),
						)
					}
				}

				if rep.CreditBalance != "" {
					output += fmt.Sprintf("  %s   %s %s\n",
						ui.Muted.Render(pipe),
						ui.Label.Render("Credit Balance:"),
						ui.Bright.Render(rep.CreditBalance),
					)
				}

				if rep.QuotaNotice != "" {
					output += fmt.Sprintf("  %s   %s\n",
						ui.Muted.Render(pipe),
						ui.Muted.Render("ℹ "+rep.QuotaNotice),
					)
				}
			}

			if !isLastAgent {
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
	Short: "Configure or update local tracking allowance for an account",
	Long:  "Manually configure the total requests, used requests, and reset duration for local tracking.",
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

		fmt.Print(ui.SuccessMessage(fmt.Sprintf("Updated quota tracking for %s/%s (%d/%d requests, resets in %v)",
			agentName, accountName, quotaSetTotal-quotaSetUsed, quotaSetTotal, resetDuration)))
		return nil
	},
}

func init() {
	quotaCmd.Flags().BoolVar(&quotaJsonOutput, "json", false, "Output quota in JSON format for automated scripts/AI agents")

	quotaSetCmd.Flags().IntVarP(&quotaSetTotal, "total", "t", 300, "Total request allowance (e.g. 300)")
	quotaSetCmd.Flags().IntVarP(&quotaSetUsed, "used", "u", 0, "Used requests (e.g. 50)")
	quotaSetCmd.Flags().StringVarP(&quotaSetResetIn, "reset", "r", "24h", "Time until reset (e.g. 4h, 24h, 30m)")
	quotaSetCmd.Flags().StringVarP(&quotaSetModel, "model", "m", "", "Associated model name")

	quotaCmd.AddCommand(quotaSetCmd)
	rootCmd.AddCommand(quotaCmd)
}