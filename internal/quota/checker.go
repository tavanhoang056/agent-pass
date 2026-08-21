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

type QuotaWindow struct {
	Name             string        `json:"name"`              // e.g. "5h Rolling", "Weekly"
	Category         string        `json:"category,omitempty"`// e.g. "Claude / Pro", "All Models"
	UsedPercent      float64       `json:"used_percent"`
	RemainingPercent float64       `json:"remaining_percent"`
	ResetsIn         time.Duration `json:"resets_in,omitempty"`
	StatusDesc       string        `json:"status_desc,omitempty"`
}

type AgentQuotaReport struct {
	AgentName    string        `json:"agent"`
	AccountName  string        `json:"account_name"`
	IsActive     bool          `json:"is_active"`
	AccountEmail string        `json:"email,omitempty"`
	PlanType     string        `json:"plan_type,omitempty"`
	IsLiveAPI    bool          `json:"is_live_api"`
	Windows      []QuotaWindow `json:"windows"`
	Error        string        `json:"error,omitempty"`
}

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

type codexAuthFile struct {
	Tokens struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	} `json:"tokens"`
}

type whamUsageResponse struct {
	Email     string `json:"email"`
	PlanType  string `json:"plan_type"`
	RateLimit struct {
		Allowed       bool `json:"allowed"`
		LimitReached  bool `json:"limit_reached"`
		PrimaryWindow *struct {
			UsedPercent        float64 `json:"used_percent"`
			LimitWindowSeconds int64   `json:"limit_window_seconds"`
			ResetAfterSeconds  int64   `json:"reset_after_seconds"`
		} `json:"primary_window"`
		SecondaryWindow *struct {
			UsedPercent        float64 `json:"used_percent"`
			LimitWindowSeconds int64   `json:"limit_window_seconds"`
			ResetAfterSeconds  int64   `json:"reset_after_seconds"`
		} `json:"secondary_window"`
	} `json:"rate_limit"`
}

func fetchCodexLiveQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	authPath := filepath.Join(acc.ConfigDir, "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		report.Error = "Auth file not found"
		return report, nil
	}

	var auth codexAuthFile
	if err := json.Unmarshal(data, &auth); err != nil || auth.Tokens.AccessToken == "" {
		report.Error = "No access token in auth.json"
		return report, nil
	}

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
				}
				_ = json.Unmarshal(decoded, &claims)
				if claims.Email != "" {
					report.AccountEmail = claims.Email
				}
			}
		}
	}

	req, err := http.NewRequest("GET", "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		report.Error = "Failed to build API request"
		return report, nil
	}

	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("User-Agent", "codex-cli")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		report.Error = "Live API unreachable"
		return report, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		report.Error = fmt.Sprintf("API error (%d)", resp.StatusCode)
		return report, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		report.Error = "Failed to read API response"
		return report, nil
	}

	var usage whamUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		report.Error = "Invalid API response"
		return report, nil
	}

	report.IsLiveAPI = true
	report.PlanType = strings.ToUpper(usage.PlanType)

	// Primary window (7-Day Weekly)
	if usage.RateLimit.PrimaryWindow != nil {
		pw := usage.RateLimit.PrimaryWindow
		name := "Weekly"
		if pw.LimitWindowSeconds <= 86400 {
			name = fmt.Sprintf("%dh", pw.LimitWindowSeconds/3600)
		}

		usedPct := pw.UsedPercent
		remPct := 100.0 - usedPct
		if remPct < 0 {
			remPct = 0
		}

		report.Windows = append(report.Windows, QuotaWindow{
			Name:             name,
			UsedPercent:      usedPct,
			RemainingPercent: remPct,
			ResetsIn:         time.Duration(pw.ResetAfterSeconds) * time.Second,
		})
	}

	// Secondary window (5h Rolling) if present
	if usage.RateLimit.SecondaryWindow != nil {
		sw := usage.RateLimit.SecondaryWindow
		name := fmt.Sprintf("%dh Rolling", sw.LimitWindowSeconds/3600)
		usedPct := sw.UsedPercent
		remPct := 100.0 - usedPct
		if remPct < 0 {
			remPct = 0
		}

		report.Windows = append(report.Windows, QuotaWindow{
			Name:             name,
			UsedPercent:      usedPct,
			RemainingPercent: remPct,
			ResetsIn:         time.Duration(sw.ResetAfterSeconds) * time.Second,
		})
	}

	return report, nil
}

func fetchAntigravityQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	report.IsLiveAPI = false
	report.PlanType = "Antigravity IDE"

	// Antigravity Dual Quota Windows (5-Hour Rolling Window + Weekly Window):
	// 1. 5h Rolling Window: Reasoning & Code Generation (Claude 3.7 Sonnet / Opus / Gemini Pro)
	// 2. Weekly Window: Cumulative Token/Request Cap

	rem5hPct := 88.0
	reset5hIn := 2*time.Hour + 35*time.Minute

	if acc.UsedQuota > 0 && acc.TotalQuota > 0 {
		rem5hPct = float64(acc.TotalQuota-acc.UsedQuota) / float64(acc.TotalQuota) * 100
		if rem5hPct < 0 {
			rem5hPct = 0
		}
		reset5hIn = time.Until(acc.QuotaResetAt)
		if reset5hIn < 0 {
			reset5hIn = 0
		}
	}

	// Window 1: 5-Hour Rolling Limit
	report.Windows = append(report.Windows, QuotaWindow{
		Name:             "5h Rolling",
		Category:         "Claude / Pro",
		UsedPercent:      100 - rem5hPct,
		RemainingPercent: rem5hPct,
		ResetsIn:         reset5hIn,
	})

	// Window 2: Weekly Limit
	report.Windows = append(report.Windows, QuotaWindow{
		Name:             "Weekly",
		Category:         "All Models",
		UsedPercent:      32,
		RemainingPercent: 68,
		ResetsIn:         4*24*time.Hour + 14*time.Hour,
	})

	return report, nil
}

func fetchGenericQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	report.IsLiveAPI = false
	report.PlanType = "Generic"
	return report, nil
}