package quota

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-pass/internal/config"
)

// QuotaWindow represents a specific rate limit window (e.g. 5-hour, weekly, monthly)
type QuotaWindow struct {
	Name            string        `json:"name"`             // e.g. "Weekly Window", "5-Hour Window", "Monthly Allowance"
	UsedPercent     float64       `json:"used_percent"`     // 0-100%
	RemainingPercent float64      `json:"remaining_percent"`// 0-100%
	UsedCount       int           `json:"used_count,omitempty"`
	TotalCount      int           `json:"total_count,omitempty"`
	WindowDuration  time.Duration `json:"window_duration,omitempty"`
	ResetsIn        time.Duration `json:"resets_in"`
	ResetAt         time.Time     `json:"reset_at,omitempty"`
}

// AgentQuotaReport represents comprehensive, honest quota status for an agent account
type AgentQuotaReport struct {
	AgentName      string        `json:"agent"`
	AccountName    string        `json:"account_name"`
	IsActive       bool          `json:"is_active"`
	AccountEmail   string        `json:"email,omitempty"`
	AccountDisplayName string    `json:"display_name,omitempty"`
	PlanType       string        `json:"plan_type,omitempty"` // e.g. "ChatGPT Plus", "Pro", "Free"
	IsLiveAPI      bool          `json:"is_live_api"`        // true if fetched from live server
	StatusMessage  string        `json:"status_message"`     // e.g. "Allowed", "Rate Limited"
	CreditBalance  string        `json:"credit_balance,omitempty"`
	Windows        []QuotaWindow `json:"windows"`
	QuotaNotice    string        `json:"quota_notice,omitempty"` // For agents without public quota endpoint
	Error          string        `json:"error,omitempty"`
}

// CheckAgentQuota performs real live API fetching if supported, or checks local engine
func CheckAgentQuota(cfg *config.Config, agentName, accountName string) (*AgentQuotaReport, error) {
	acc := cfg.GetAccount(agentName, accountName)
	if acc == nil {
		return nil, fmt.Errorf("account '%s' not found for agent '%s'", accountName, agentName)
	}

	report := &AgentQuotaReport{
		AgentName:    agentName,
		AccountName:  accountName,
		AccountEmail: acc.Email,
		Windows:      []QuotaWindow{},
	}

	agentCfg := cfg.GetAgent(agentName)
	if agentCfg != nil {
		report.IsActive = (agentCfg.Active == accountName)
	}

	switch agentName {
	case "codex":
		return fetchCodexLiveQuota(acc, report)
	case "antigravity":
		return fetchAntigravityQuota(acc, report)
	default:
		return fetchGenericQuota(acc, report)
	}
}

// OpenAI Codex live API fetcher
type codexTokens struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

type codexAuthFile struct {
	Tokens codexTokens `json:"tokens"`
}

type whamRateLimitWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
	ResetAfterSeconds  int64   `json:"reset_after_seconds"`
	ResetAt            int64   `json:"reset_at"`
}

type whamUsageResponse struct {
	Email     string `json:"email"`
	PlanType  string `json:"plan_type"`
	RateLimit struct {
		Allowed         bool                 `json:"allowed"`
		LimitReached    bool                 `json:"limit_reached"`
		PrimaryWindow   *whamRateLimitWindow `json:"primary_window"`
		SecondaryWindow *whamRateLimitWindow `json:"secondary_window"`
	} `json:"rate_limit"`
	Credits struct {
		HasCredits bool   `json:"has_credits"`
		Balance    string `json:"balance"`
	} `json:"credits"`
}

func fetchCodexLiveQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	authPath := filepath.Join(acc.ConfigDir, "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		report.Error = fmt.Sprintf("Cannot read auth file at %s: %v", authPath, err)
		return report, nil
	}

	var auth codexAuthFile
	if err := json.Unmarshal(data, &auth); err != nil || auth.Tokens.AccessToken == "" {
		report.Error = "No valid access_token found in auth.json"
		return report, nil
	}

	// Extract display name / email from id_token if available
	if auth.Tokens.IDToken != "" {
		parts := strings.Split(auth.Tokens.IDToken, ".")
		if len(parts) >= 2 {
			payload := parts[1]
			for len(payload)%4 != 0 {
				payload += "="
			}
			if decoded, err := base64.URLEncoding.DecodeString(payload); err == nil {
				var claims struct {
					Email string `json:"email"`
					Name  string `json:"name"`
				}
				_ = json.Unmarshal(decoded, &claims)
				if claims.Email != "" {
					report.AccountEmail = claims.Email
				}
				if claims.Name != "" {
					report.AccountDisplayName = claims.Name
				}
			}
		}
	}

	// Call live OpenAI ChatGPT WHAM usage API
	req, err := http.NewRequest("GET", "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		report.Error = fmt.Sprintf("Failed to create request: %v", err)
		return report, nil
	}

	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("User-Agent", "codex-cli")

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		report.Error = fmt.Sprintf("Live API unreachable: %v", err)
		return report, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		report.Error = fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body))
		return report, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		report.Error = fmt.Sprintf("Failed to read API response: %v", err)
		return report, nil
	}

	var usage whamUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		report.Error = fmt.Sprintf("Failed to parse API JSON: %v", err)
		return report, nil
	}

	report.IsLiveAPI = true
	report.PlanType = strings.ToUpper(usage.PlanType)
	if usage.RateLimit.Allowed {
		report.StatusMessage = "Allowed (Normal)"
	} else {
		report.StatusMessage = "Rate Limited"
	}

	if usage.Credits.Balance != "" && usage.Credits.Balance != "0" {
		report.CreditBalance = "$" + usage.Credits.Balance
	}

	// Primary window (Weekly or Hourly)
	if usage.RateLimit.PrimaryWindow != nil {
		pw := usage.RateLimit.PrimaryWindow
		windowName := "Weekly Limit"
		if pw.LimitWindowSeconds <= 86400 {
			windowName = fmt.Sprintf("%dh Window", pw.LimitWindowSeconds/3600)
		} else {
			windowName = fmt.Sprintf("%dd Weekly Window", pw.LimitWindowSeconds/86400)
		}

		usedPct := pw.UsedPercent
		remPct := 100.0 - usedPct
		if remPct < 0 {
			remPct = 0
		}

		resetsIn := time.Duration(pw.ResetAfterSeconds) * time.Second
		resetAt := time.Unix(pw.ResetAt, 0)

		report.Windows = append(report.Windows, QuotaWindow{
			Name:             windowName,
			UsedPercent:      usedPct,
			RemainingPercent: remPct,
			WindowDuration:   time.Duration(pw.LimitWindowSeconds) * time.Second,
			ResetsIn:         resetsIn,
			ResetAt:          resetAt,
		})
	}

	// Secondary window if present (e.g. 5-hour rolling)
	if usage.RateLimit.SecondaryWindow != nil {
		sw := usage.RateLimit.SecondaryWindow
		windowName := fmt.Sprintf("%dh Rolling Limit", sw.LimitWindowSeconds/3600)
		usedPct := sw.UsedPercent
		remPct := 100.0 - usedPct
		if remPct < 0 {
			remPct = 0
		}

		report.Windows = append(report.Windows, QuotaWindow{
			Name:             windowName,
			UsedPercent:      usedPct,
			RemainingPercent: remPct,
			WindowDuration:   time.Duration(sw.LimitWindowSeconds) * time.Second,
			ResetsIn:         time.Duration(sw.ResetAfterSeconds) * time.Second,
			ResetAt:          time.Unix(sw.ResetAt, 0),
		})
	}

	return report, nil
}

func fetchAntigravityQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	report.IsLiveAPI = false
	report.PlanType = "Antigravity IDE Runtime"
	report.StatusMessage = "Active / Available"
	report.QuotaNotice = "Antigravity operates on Google Cloud internal session routing (Flash models: High Throughput / Reasoning models: Rolling Session Window). No public REST quota endpoint is exposed by the IDE daemon."

	// If the user configured custom limits in config.yaml, display them honestly
	if acc.TotalQuota > 0 {
		used := acc.UsedQuota
		rem := acc.TotalQuota - used
		if rem < 0 {
			rem = 0
		}
		remPct := float64(rem) / float64(acc.TotalQuota) * 100

		resetsIn := time.Until(acc.QuotaResetAt)
		if resetsIn < 0 {
			resetsIn = 0
		}

		report.Windows = append(report.Windows, QuotaWindow{
			Name:             "Tracked Quota",
			UsedPercent:      100 - remPct,
			RemainingPercent: remPct,
			UsedCount:        used,
			TotalCount:       acc.TotalQuota,
			ResetsIn:         resetsIn,
			ResetAt:          acc.QuotaResetAt,
		})
	}

	return report, nil
}

func fetchGenericQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	report.IsLiveAPI = false
	report.PlanType = "Local Configuration"
	report.StatusMessage = "Configured"

	if acc.TotalQuota > 0 {
		used := acc.UsedQuota
		rem := acc.TotalQuota - used
		if rem < 0 {
			rem = 0
		}
		remPct := float64(rem) / float64(acc.TotalQuota) * 100

		resetsIn := time.Until(acc.QuotaResetAt)
		if resetsIn < 0 {
			resetsIn = 0
		}

		report.Windows = append(report.Windows, QuotaWindow{
			Name:             "Allowance",
			UsedPercent:      100 - remPct,
			RemainingPercent: remPct,
			UsedCount:        used,
			TotalCount:       acc.TotalQuota,
			ResetsIn:         resetsIn,
			ResetAt:          acc.QuotaResetAt,
		})
	}

	return report, nil
}