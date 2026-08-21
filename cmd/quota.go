package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"agent-pass/internal/config"
	"agent-pass/internal/quota"
	"agent-pass/internal/ui"
)

var quotaJsonOutput bool

var quotaCmd = &cobra.Command{
	Use:   "quota [agent]",
	Short: "Check remaining quota for agent accounts",
	Long:  "Inspect remaining quota and reset countdowns for configured AI coding agents.",
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

		// Concise, clean UI Rendering
		header := ui.SectionHeader(ui.IconQuota, "Quota Overview")
		output := "\n" + header + "\n\n"

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
				indicator := ui.IconDotOpen
				if rep.IsActive {
					accLabel = ui.AccountActive.Render(rep.AccountName)
					indicator = ui.Success.Render(ui.IconDot)
				}

				planBadge := ""
				if rep.PlanType != "" {
					planBadge = " " + ui.Accent.Render("["+rep.PlanType+"]")
				}

				pipe := ui.TreePipe
				if isLastAcc {
					pipe = ui.TreeSpace
				}

				if rep.Error != "" {
					output += fmt.Sprintf("  %s %s %s%s\n",
						ui.Muted.Render(tree), indicator, accLabel, planBadge)
					output += fmt.Sprintf("  %s   %s\n",
						ui.Muted.Render(pipe), ui.Warning.Render("⚠ "+rep.Error))
				} else if len(rep.Windows) == 1 && rep.AgentName == "antigravity" {
					// Single-line Antigravity format
					bar := ui.ProgressBar(100, 10)
					output += fmt.Sprintf("  %s %s %s  %s  %s\n",
						ui.Muted.Render(tree),
						indicator,
						accLabel,
						bar,
						ui.Success.Render("100% · Active"),
					)
				} else if len(rep.Windows) > 0 {
					output += fmt.Sprintf("  %s %s %s%s\n",
						ui.Muted.Render(tree), indicator, accLabel, planBadge)

					for _, win := range rep.Windows {
						bar := ui.ProgressBar(win.RemainingPercent, 10)
						pctStr := fmt.Sprintf("%.0f%%", win.RemainingPercent)

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

						output += fmt.Sprintf("  %s   %s: %s  %s%s\n",
							ui.Muted.Render(pipe),
							ui.Muted.Render(win.Name),
							bar,
							pctStyle(pctStr),
							ui.Muted.Render(resetStr),
						)
					}
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

func init() {
	quotaCmd.Flags().BoolVar(&quotaJsonOutput, "json", false, "Output quota in JSON format for automated scripts/AI agents")
	rootCmd.AddCommand(quotaCmd)
}