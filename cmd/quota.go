package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"agpass/internal/config"
	"agpass/internal/quota"
	"agpass/internal/ui"
)

var quotaJsonOutput bool

var quotaCmd = &cobra.Command{
	Use:   "quota [agent]",
	Short: "Check remaining quota for agent accounts",
	Long:  "Inspect model group quotas (Gemini, Claude & GPT, OpenAI) for configured AI agents.",
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

		type checkTask struct {
			agentName   string
			accountName string
		}

		var tasks []checkTask
		for _, name := range agentNames {
			agent := cfg.GetAgent(name)
			for _, acc := range agent.Accounts {
				tasks = append(tasks, checkTask{
					agentName:   name,
					accountName: acc.Name,
				})
			}
		}

		type checkResult struct {
			idx    int
			report *quota.AgentQuotaReport
		}

		reports := make([]*quota.AgentQuotaReport, len(tasks))
		resChan := make(chan checkResult, len(tasks))

		isTTY := (isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())) && !quotaJsonOutput

		// Start spinner if in interactive terminal
		doneSpinner := make(chan struct{})
		if isTTY {
			go func() {
				spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
				i := 0
				ticker := time.NewTicker(75 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-doneSpinner:
						fmt.Print("\r" + strings.Repeat(" ", 60) + "\r")
						return
					case <-ticker.C:
						spin := ui.Subtitle.Render(spinnerChars[i%len(spinnerChars)])
						msg := ui.Muted.Render(fmt.Sprintf(" Fetching quota metrics for %d account(s)...", len(tasks)))
						fmt.Printf("\r  %s%s", spin, msg)
						i++
					}
				}
			}()
		}

		var wg sync.WaitGroup
		for i, t := range tasks {
			wg.Add(1)
			go func(idx int, task checkTask) {
				defer wg.Done()
				rep, err := quota.CheckAgentQuota(cfg, task.agentName, task.accountName)
				if err != nil {
					resChan <- checkResult{
						idx: idx,
						report: &quota.AgentQuotaReport{
							AgentName:   task.agentName,
							AccountName: task.accountName,
							Error:       err.Error(),
						},
					}
					return
				}
				resChan <- checkResult{idx: idx, report: rep}
			}(i, t)
		}

		wg.Wait()
		close(resChan)

		if isTTY {
			close(doneSpinner)
			time.Sleep(10 * time.Millisecond)
		}

		for res := range resChan {
			reports[res.idx] = res.report
		}

		if quotaJsonOutput {
			data, _ := json.MarshalIndent(reports, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		header := ui.SectionHeader(ui.IconQuota, "Quota & Model Tiers Overview")
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
				indicator := ui.Muted.Render(ui.IconDotOpen)
				activeBadge := ""
				if rep.IsActive {
					accLabel = ui.AccountActive.Render(rep.AccountName)
					indicator = ui.Success.Render(ui.IconDot)
					activeBadge = " " + ui.Success.Render("← active")
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
					output += fmt.Sprintf("  %s %s %s%s%s\n",
						ui.Muted.Render(tree), indicator, accLabel, activeBadge, planBadge)
					output += fmt.Sprintf("  %s   %s\n",
						ui.Muted.Render(pipe), ui.Warning.Render("⚠ "+rep.Error))
				} else {
					output += fmt.Sprintf("  %s %s %s%s%s\n",
						ui.Muted.Render(tree), indicator, accLabel, activeBadge, planBadge)

					for _, g := range rep.Groups {
						output += fmt.Sprintf("  %s   %s\n",
							ui.Muted.Render(pipe),
							ui.Subtitle.Render("◆ "+g.GroupName+":"),
						)

						for _, win := range g.Windows {
							bar := ui.ProgressBar(win.RemainingPercent, 10)
							pctStr := fmt.Sprintf("%3.0f%%", win.RemainingPercent)

							var pctStyle func(string) string
							switch {
							case win.RemainingPercent >= 70:
								pctStyle = func(s string) string { return ui.Success.Render(s) }
							case win.RemainingPercent > 0:
								pctStyle = func(s string) string { return ui.Warning.Render(s) }
							default:
								pctStyle = func(s string) string { return ui.Danger.Render(s) }
							}

							desc := ""
							if win.ResetsIn > 0 {
								days := int(win.ResetsIn.Hours() / 24)
								hours := int(win.ResetsIn.Hours()) % 24
								mins := int(win.ResetsIn.Minutes()) % 60

								if days > 0 {
									desc = fmt.Sprintf(" · Resets in %dd %dh", days, hours)
								} else if hours > 0 {
									desc = fmt.Sprintf(" · Resets in %dh %dm", hours, mins)
								} else {
									desc = fmt.Sprintf(" · Resets in %dm", mins)
								}
							}

							limitNotice := ""
							if win.IsHitLimit {
								limitNotice = " " + ui.Danger.Render("(Limit Reached)")
							} else if win.StatusText != "" {
								limitNotice = " " + ui.Muted.Render("("+win.StatusText+")")
							}

							output += fmt.Sprintf("  %s     %-13s %s  %s%s%s\n",
								ui.Muted.Render(pipe),
								ui.Muted.Render(win.Name+":"),
								bar,
								pctStyle(pctStr),
								ui.Muted.Render(desc),
								limitNotice,
							)
						}
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