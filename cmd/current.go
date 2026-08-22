package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"agpass/internal/config"
	"agpass/internal/ui"
)

var (
	currentJsonOutput bool
	currentRawOutput  bool
)

var currentCmd = &cobra.Command{
	Use:   "current [agent]",
	Short: "Show active account for an agent",
	Long:  "Print the currently active account name for the specified agent, or all agents if none is specified.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			if currentJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				fmt.Println(string(res))
				return nil
			}
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to load config: %v", err)))
			return nil
		}

		if len(cfg.Agents) == 0 {
			if currentJsonOutput {
				fmt.Println("{}")
				return nil
			}
			fmt.Print(ui.WarningMessage("No agents configured yet. Use 'agpass add <agent>' to add one."))
			return nil
		}

		isTTY := (isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())) && !currentRawOutput

		// When specific agent is provided (e.g. agpass current codex)
		if len(args) == 1 {
			agentName := args[0]
			agentCfg := cfg.GetAgent(agentName)
			if agentCfg == nil {
				if currentJsonOutput {
					res, _ := json.Marshal(map[string]interface{}{"error": fmt.Sprintf("agent '%s' not found", agentName)})
					fmt.Println(string(res))
					return nil
				}
				fmt.Print(ui.ErrorMessage(fmt.Sprintf("Agent '%s' not configured", agentName)))
				return nil
			}

			if currentJsonOutput {
				res, _ := json.Marshal(map[string]interface{}{
					"agent":          agentName,
					"active_account": agentCfg.Active,
				})
				fmt.Println(string(res))
				return nil
			}

			// Interactive TTY mode -> render styled card
			if isTTY {
				header := ui.SectionHeader(ui.IconAgent, fmt.Sprintf("Active Account · %s", agentName))
				output := "\n" + header + "\n\n"
				output += fmt.Sprintf("  %s %s  %s %s\n",
					ui.Muted.Render(ui.TreeLast),
					ui.AgentName.Render(fmt.Sprintf("%-12s", agentName)),
					ui.Success.Render(ui.IconDot),
					ui.AccountActive.Render(agentCfg.Active),
				)
				fmt.Print(ui.BoxBorder.Render(output))
				fmt.Println()
				return nil
			}

			// Script / Raw / Pipe mode -> raw plain string
			fmt.Println(agentCfg.Active)
			return nil
		}

		// When no agent is specified: show active account for all configured agents
		agentNames := cfg.ListAgents()
		sort.Strings(agentNames)

		if currentJsonOutput {
			activeMap := make(map[string]string)
			for _, name := range agentNames {
				agentCfg := cfg.GetAgent(name)
				if agentCfg != nil && agentCfg.Active != "" {
					activeMap[name] = agentCfg.Active
				}
			}
			data, _ := json.MarshalIndent(activeMap, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Interactive TTY mode -> render full styled overview box
		if isTTY {
			header := ui.SectionHeader(ui.IconAgent, "Active Agent Accounts")
			output := "\n" + header + "\n\n"

			for i, name := range agentNames {
				agentCfg := cfg.GetAgent(name)
				isLast := i == len(agentNames)-1
				tree := ui.TreeBranch
				if isLast {
					tree = ui.TreeLast
				}

				activeName := "none"
				if agentCfg != nil && agentCfg.Active != "" {
					activeName = agentCfg.Active
				}

				output += fmt.Sprintf("  %s %s  %s %s\n",
					ui.Muted.Render(tree),
					ui.AgentName.Render(fmt.Sprintf("%-12s", name)),
					ui.Success.Render(ui.IconDot),
					ui.AccountActive.Render(activeName),
				)
			}

			fmt.Print(ui.BoxBorder.Render(output))
			fmt.Println()
			return nil
		}

		// Script / Pipe mode -> key: value lines
		for _, name := range agentNames {
			agentCfg := cfg.GetAgent(name)
			if agentCfg != nil && agentCfg.Active != "" {
				fmt.Printf("%s: %s\n", name, agentCfg.Active)
			}
		}
		return nil
	},
}

func init() {
	currentCmd.Flags().BoolVar(&currentJsonOutput, "json", false, "Output in JSON format")
	currentCmd.Flags().BoolVarP(&currentRawOutput, "raw", "r", false, "Output raw account name for scripts")
	rootCmd.AddCommand(currentCmd)
}