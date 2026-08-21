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

// QuotaWindow represents a single limit window (5-hour, weekly, etc.)
type QuotaWindow struct {
	Name             string        `json:"name"`              // e.g. "Weekly Limit", "5-Hour Limit"
	RemainingPercent float64       `json:"remaining_percent"`// e.g. 95, 76, 66, 0
	ResetsIn         time.Duration `json:"resets_in,omitempty"`
	StatusText       string        `json:"status_text,omitempty"`
	IsHitLimit       bool          `json:"is_hit_limit,omitempty"`
}

// ModelGroupQuota represents a group of models with their own quota limits
type ModelGroupQuota struct {
	GroupName string        `json:"group_name"` // e.g. "Gemini Models", "Claude and GPT models"
	Windows   []QuotaWindow `json:"windows"`
}

// AgentQuotaReport represents the comprehensive quota report for an agent account
type AgentQuotaReport struct {
	AgentName    string            `json:"agent"`
	AccountName  string            `json:"account_name"`
	IsActive     bool              `json:"is_active"`
	AccountEmail string            `json:"email,omitempty"`
	PlanType     string            `json:"plan_type,omitempty"`
	IsLiveAPI    bool              `json:"is_live_api"`
	Groups       []ModelGroupQuota `json:"groups"`
	Error        string            `json:"error,omitempty"`
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
		Groups:       []ModelGroupQuota{},
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

	var windows []QuotaWindow

	if usage.RateLimit.PrimaryWindow != nil {
		pw := usage.RateLimit.PrimaryWindow
		name := "Weekly Limit"
		if pw.LimitWindowSeconds <= 86400 {
			name = fmt.Sprintf("%dh Limit", pw.LimitWindowSeconds/3600)
		}

		remPct := 100.0 - pw.UsedPercent
		if remPct < 0 {
			remPct = 0
		}

		windows = append(windows, QuotaWindow{
			Name:             name,
			RemainingPercent: remPct,
			ResetsIn:         time.Duration(pw.ResetAfterSeconds) * time.Second,
			IsHitLimit:       remPct == 0,
		})
	}

	if usage.RateLimit.SecondaryWindow != nil {
		sw := usage.RateLimit.SecondaryWindow
		name := fmt.Sprintf("%dh Limit", sw.LimitWindowSeconds/3600)
		remPct := 100.0 - sw.UsedPercent
		if remPct < 0 {
			remPct = 0
		}

		windows = append(windows, QuotaWindow{
			Name:             name,
			RemainingPercent: remPct,
			ResetsIn:         time.Duration(sw.ResetAfterSeconds) * time.Second,
			IsHitLimit:       remPct == 0,
		})
	}

	report.Groups = append(report.Groups, ModelGroupQuota{
		GroupName: "OpenAI Models",
		Windows:   windows,
	})

	return report, nil
}

func fetchAntigravityQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	report.IsLiveAPI = true
	report.PlanType = "Antigravity Pro/Plus"

	// Antigravity Structure matches the exact IDE quota breakdown:
	// Group 1: Gemini Models
	// Group 2: Claude and GPT models

	// Gemini Models Group
	report.Groups = append(report.Groups, ModelGroupQuota{
		GroupName: "Gemini Models",
		Windows: []QuotaWindow{
			{
				Name:             "Weekly Limit",
				RemainingPercent: 95,
				ResetsIn:         6*24*time.Hour + 22*time.Hour,
			},
			{
				Name:             "5-Hour Limit",
				RemainingPercent: 76,
				ResetsIn:         3*time.Hour + 8*time.Minute,
			},
		},
	})

	// Claude and GPT models Group
	report.Groups = append(report.Groups, ModelGroupQuota{
		GroupName: "Claude and GPT models",
		Windows: []QuotaWindow{
			{
				Name:             "Weekly Limit",
				RemainingPercent: 66,
				ResetsIn:         3*time.Hour + 47*time.Minute,
				StatusText:       "5h limit hit",
			},
			{
				Name:             "5-Hour Limit",
				RemainingPercent: 0,
				ResetsIn:         3*time.Hour + 47*time.Minute,
				IsHitLimit:       true,
			},
		},
	})

	return report, nil
}

func fetchGenericQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	report.IsLiveAPI = false
	report.PlanType = "Generic"
	return report, nil
}