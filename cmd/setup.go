package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/agent-pass/agent-pass/internal/config"
	"github.com/agent-pass/agent-pass/internal/ui"
)

var setupPathCmd = &cobra.Command{
	Use:   "setup-path",
	Short: "Install agpass binary to user PATH",
	Long:  "Copies the agpass executable to ~/.agpass/bin and registers the directory in the user PATH environment variable.",
	RunE: func(cmd *cobra.Command, args []string) error {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to get current executable path: %v", err)))
			return nil
		}

		binDir := filepath.Join(config.ConfigDir(), "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to create bin dir: %v", err)))
			return nil
		}

		destFilename := "agpass"
		if runtime.GOOS == "windows" {
			destFilename = "agpass.exe"
		}
		destPath := filepath.Join(binDir, destFilename)

		// Copy executable
		srcFile, err := os.Open(execPath)
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to open source binary: %v", err)))
			return nil
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to create destination binary: %v", err)))
			return nil
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to copy binary: %v", err)))
			return nil
		}

		// Register to PATH
		if runtime.GOOS == "windows" {
			psScript := fmt.Sprintf(`
$binPath = "%s"
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -split ";" -notcontains $binPath) {
    [System.Environment]::SetEnvironmentVariable("Path", "$currentPath;$binPath", "User")
    Write-Host "ADDED"
} else {
    Write-Host "ALREADY_PRESENT"
}
`, binDir)

			out, err := exec.Command("powershell", "-NoProfile", "-Command", psScript).CombinedOutput()
			if err != nil {
				fmt.Print(ui.ErrorMessage(fmt.Sprintf("Failed to update PATH: %v", err)))
				return nil
			}

			status := strings.TrimSpace(string(out))
			if status == "ADDED" {
				fmt.Print(ui.SuccessMessage(fmt.Sprintf("Installed agpass to %s and added to user PATH!", binDir)))
				fmt.Println(ui.Muted.Render("  Please restart your terminal session for PATH changes to take effect."))
			} else {
				fmt.Print(ui.SuccessMessage(fmt.Sprintf("Updated binary at %s (PATH already configured).", destPath)))
			}
		} else {
			fmt.Print(ui.SuccessMessage(fmt.Sprintf("Installed binary to %s", destPath)))
			fmt.Println(ui.Muted.Render(fmt.Sprintf("  Ensure '%s' is in your PATH (e.g. export PATH=\"$PATH:%s\")", binDir, binDir)))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupPathCmd)
}