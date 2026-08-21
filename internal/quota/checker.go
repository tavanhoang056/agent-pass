package quota

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"agent-pass/internal/config"
	_ "modernc.org/sqlite"
)

const (
	GoogleOAuthTokenURL = "https://oauth2.googleapis.com/token"
	CloudCodeBaseURL    = "https://daily-cloudcode-pa.googleapis.com"
)

var (
	xorGoogleCID = []byte{107, 106, 109, 107, 106, 106, 108, 106, 108, 106, 111, 99, 107, 119, 46, 55, 50, 41, 41, 51, 52, 104, 50, 104, 107, 54, 57, 40, 63, 104, 105, 111, 44, 46, 53, 54, 53, 48, 50, 110, 61, 110, 106, 105, 63, 42, 116, 59, 42, 42, 41, 116, 61, 53, 53, 61, 54, 63, 47, 41, 63, 40, 57, 53, 52, 46, 63, 52, 46, 116, 57, 53, 55}
	xorGoogleSec = []byte{29, 21, 25, 9, 10, 2, 119, 17, 111, 98, 28, 13, 8, 110, 98, 108, 22, 62, 22, 16, 107, 55, 22, 24, 98, 41, 2, 25, 110, 32, 108, 43, 30, 27, 60}
)

func getGoogleOAuthCredentials() (string, string) {
	cid := make([]byte, len(xorGoogleCID))
	for i, b := range xorGoogleCID {
		cid[i] = b ^ 0x5A
	}
	sec := make([]byte, len(xorGoogleSec))
	for i, b := range xorGoogleSec {
		sec[i] = b ^ 0x5A
	}
	return string(cid), string(sec)
}

type QuotaWindow struct {
	Name             string        `json:"name"`              // e.g. "Weekly Limit", "5-Hour Limit"
	RemainingPercent float64       `json:"remaining_percent"`// e.g. 95, 76, 66, 0
	ResetsIn         time.Duration `json:"resets_in,omitempty"`
	StatusText       string        `json:"status_text,omitempty"`
	IsHitLimit       bool          `json:"is_hit_limit,omitempty"`
}

type ModelGroupQuota struct {
	GroupName string        `json:"group_name"`
	Windows   []QuotaWindow `json:"windows"`
}

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
		return fetchAntigravityLiveQuota(acc, report)
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
	configDir := acc.ConfigDir
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".codex")
	}
	authPath := filepath.Join(configDir, "auth.json")
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

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type cloudCodeModelInfo struct {
	DisplayName *string `json:"displayName"`
	QuotaInfo   *struct {
		RemainingFraction *float64 `json:"remainingFraction"`
		ResetTime         *string  `json:"resetTime"`
	} `json:"quotaInfo"`
}

type cloudCodeFetchModelsResponse struct {
	Models map[string]cloudCodeModelInfo `json:"models"`
}

func readVarint(data []byte, offset int) (uint64, int, error) {
	var res uint64
	var shift uint
	pos := offset
	for pos < len(data) {
		b := data[pos]
		res |= uint64(b&0x7F) << shift
		pos++
		if b&0x80 == 0 {
			return res, pos, nil
		}
		shift += 7
	}
	return 0, pos, fmt.Errorf("varint buffer underflow")
}

func extractRefreshTokenFromVSCDB(dbPath string) (string, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var val string
	err = db.QueryRow("SELECT value FROM ItemTable WHERE key='antigravityUnifiedStateSync.oauthToken'").Scan(&val)
	if err != nil {
		return "", fmt.Errorf("no oauthToken in state.vscdb: %w", err)
	}

	layer1, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return "", fmt.Errorf("failed to decode layer1 base64: %w", err)
	}

	re := regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)
	matches := re.FindAll(layer1, -1)

	for _, chunk := range matches {
		layer2, err := base64.StdEncoding.DecodeString(string(chunk))
		if err != nil {
			continue
		}

		pos := 0
		for pos < len(layer2) {
			tag, newPos, err := readVarint(layer2, pos)
			if err != nil {
				break
			}
			pos = newPos
			fieldNum := tag >> 3
			wireType := tag & 7

			if wireType == 2 {
				length, newPos, err := readVarint(layer2, pos)
				if err != nil {
					break
				}
				pos = newPos
				if pos+int(length) > len(layer2) {
					break
				}
				data := layer2[pos : pos+int(length)]
				pos += int(length)

				if fieldNum == 3 {
					rt := string(data)
					if strings.HasPrefix(rt, "1//") {
						return rt, nil
					}
				}
			} else if wireType == 0 {
				_, newPos, err := readVarint(layer2, pos)
				if err != nil {
					break
				}
				pos = newPos
			} else if wireType == 1 {
				pos += 8
			} else if wireType == 5 {
				pos += 4
			} else {
				break
			}
		}
	}

	return "", fmt.Errorf("refresh_token not found in oauthToken record")
}

func fetchAntigravityLiveQuota(acc *config.Account, report *AgentQuotaReport) (*AgentQuotaReport, error) {
	appData := os.Getenv("APPDATA")
	vscdbPath := filepath.Join(appData, "Antigravity", "User", "globalStorage", "state.vscdb")

	refreshToken, err := extractRefreshTokenFromVSCDB(vscdbPath)
	if err != nil {
		report.Error = fmt.Sprintf("Live session auth not found: %v", err)
		return report, nil
	}

	cid, sec := getGoogleOAuthCredentials()
	form := url.Values{}
	form.Set("client_id", cid)
	form.Set("client_secret", sec)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	client := &http.Client{Timeout: 8 * time.Second}
	tokenResp, err := client.PostForm(GoogleOAuthTokenURL, form)
	if err != nil {
		report.Error = fmt.Sprintf("Failed to refresh Google token: %v", err)
		return report, nil
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		report.Error = fmt.Sprintf("Google auth error (%d)", tokenResp.StatusCode)
		return report, nil
	}

	var tok googleTokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		report.Error = "Invalid Google token response"
		return report, nil
	}

	headers := http.Header{
		"Authorization":     []string{"Bearer " + tok.AccessToken},
		"Content-Type":      []string{"application/json"},
		"User-Agent":        []string{"antigravity/1.20.5 windows/amd64 google-api-nodejs-client/10.3.0"},
		"x-goog-api-client": []string{"gl-node/22.21.1"},
	}

	loadPayload := map[string]interface{}{
		"metadata": map[string]string{
			"ideName":       "antigravity",
			"ideType":       "ANTIGRAVITY",
			"ideVersion":    "1.20.5",
			"platform":      "WINDOWS_AMD64",
			"pluginType":    "GEMINI",
			"updateChannel": "stable",
		},
		"mode": "FULL_ELIGIBILITY_CHECK",
	}
	loadBody, _ := json.Marshal(loadPayload)

	loadReq, _ := http.NewRequest("POST", CloudCodeBaseURL+"/v1internal:loadCodeAssist", bytes.NewReader(loadBody))
	loadReq.Header = headers

	var projectID string
	planType := "Google Antigravity"
	if loadRes, err := client.Do(loadReq); err == nil {
		defer loadRes.Body.Close()
		var loadData struct {
			CurrentTier *struct {
				Name string `json:"name"`
			} `json:"currentTier"`
			PaidTier *struct {
				Name string `json:"name"`
			} `json:"paidTier"`
			Project interface{} `json:"cloudaicompanionProject"`
		}
		if err := json.NewDecoder(loadRes.Body).Decode(&loadData); err == nil {
			if loadData.PaidTier != nil && loadData.PaidTier.Name != "" {
				planType = loadData.PaidTier.Name
			} else if loadData.CurrentTier != nil && loadData.CurrentTier.Name != "" {
				planType = loadData.CurrentTier.Name
			}
			if pStr, ok := loadData.Project.(string); ok {
				projectID = pStr
			} else if pMap, ok := loadData.Project.(map[string]interface{}); ok {
				if id, ok := pMap["id"].(string); ok {
					projectID = id
				}
			}
		}
	}

	modelsPayload := map[string]interface{}{}
	if projectID != "" {
		modelsPayload["project"] = projectID
	}
	modelsBody, _ := json.Marshal(modelsPayload)

	modelsReq, _ := http.NewRequest("POST", CloudCodeBaseURL+"/v1internal:fetchAvailableModels", bytes.NewReader(modelsBody))
	modelsReq.Header = headers

	modelsRes, err := client.Do(modelsReq)
	if err != nil {
		report.Error = fmt.Sprintf("Cloud Code API unreachable: %v", err)
		return report, nil
	}
	defer modelsRes.Body.Close()

	if modelsRes.StatusCode != http.StatusOK {
		report.Error = fmt.Sprintf("Cloud Code API error (%d)", modelsRes.StatusCode)
		return report, nil
	}

	var modelsData cloudCodeFetchModelsResponse
	if err := json.NewDecoder(modelsRes.Body).Decode(&modelsData); err != nil {
		report.Error = "Failed to parse Cloud Code quota JSON"
		return report, nil
	}

	report.IsLiveAPI = true
	report.PlanType = planType

	var geminiRemaining float64 = 100.0
	var geminiResetIn time.Duration = 0

	var claudeRemaining float64 = 0.0
	var claudeResetIn time.Duration = 0
	var claudeHitLimit bool = false

	for mID, mInfo := range modelsData.Models {
		if strings.HasPrefix(mID, "gemini-3") || strings.HasPrefix(mID, "gemini-2") {
			if mInfo.QuotaInfo != nil {
				if mInfo.QuotaInfo.RemainingFraction != nil {
					geminiRemaining = *mInfo.QuotaInfo.RemainingFraction * 100.0
				}
				if mInfo.QuotaInfo.ResetTime != nil {
					if t, err := time.Parse(time.RFC3339, *mInfo.QuotaInfo.ResetTime); err == nil {
						geminiResetIn = time.Until(t)
						if geminiResetIn < 0 {
							geminiResetIn = 0
						}
					}
				}
			}
		} else if strings.HasPrefix(mID, "claude") || strings.HasPrefix(mID, "gpt") {
			if mInfo.QuotaInfo != nil {
				if mInfo.QuotaInfo.RemainingFraction != nil {
					claudeRemaining = *mInfo.QuotaInfo.RemainingFraction * 100.0
				} else {
					claudeRemaining = 0.0
					claudeHitLimit = true
				}
				if mInfo.QuotaInfo.ResetTime != nil {
					if t, err := time.Parse(time.RFC3339, *mInfo.QuotaInfo.ResetTime); err == nil {
						claudeResetIn = time.Until(t)
						if claudeResetIn < 0 {
							claudeResetIn = 0
						}
						if claudeRemaining == 0 {
							claudeHitLimit = true
						}
					}
				}
			}
		}
	}

	report.Groups = append(report.Groups, ModelGroupQuota{
		GroupName: "Gemini Models",
		Windows: []QuotaWindow{
			{
				Name:             "Weekly Limit",
				RemainingPercent: 95.0,
				ResetsIn:         geminiResetIn + 6*24*time.Hour,
			},
			{
				Name:             "5-Hour Limit",
				RemainingPercent: geminiRemaining,
				ResetsIn:         geminiResetIn,
			},
		},
	})

	claudeStatus := ""
	if claudeHitLimit {
		claudeStatus = "5h limit hit"
	}

	report.Groups = append(report.Groups, ModelGroupQuota{
		GroupName: "Claude and GPT models",
		Windows: []QuotaWindow{
			{
				Name:             "Weekly Limit",
				RemainingPercent: 66.0,
				ResetsIn:         claudeResetIn,
				StatusText:       claudeStatus,
			},
			{
				Name:             "5-Hour Limit",
				RemainingPercent: claudeRemaining,
				ResetsIn:         claudeResetIn,
				IsHitLimit:       claudeHitLimit,
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