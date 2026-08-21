package quota

import (
	"fmt"
	"time"

	"agent-pass/internal/config"
)

// QuotaInfo represents quota information for an account
type QuotaInfo struct {
	AgentName   string
	AccountName string
	Used        int
	Total       int
	ResetsIn    time.Duration
	Model       string
	Error       string
}

// Remaining returns remaining quota
func (q *QuotaInfo) Remaining() int {
	rem := q.Total - q.Used
	if rem < 0 {
		return 0
	}
	return rem
}

// Percent returns the remaining percentage
func (q *QuotaInfo) Percent() float64 {
	if q.Total <= 0 {
		return 0
	}
	return float64(q.Remaining()) / float64(q.Total) * 100
}

// ResetsInString returns human readable reset time
func (q *QuotaInfo) ResetsInString() string {
	if q.ResetsIn <= 0 {
		return "now"
	}
	h := int(q.ResetsIn.Hours())
	m := int(q.ResetsIn.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// CheckQuota checks and calculates quota for an agent account
func CheckQuota(cfg *config.Config, agentName, accountName string) (*QuotaInfo, error) {
	acc := cfg.GetAccount(agentName, accountName)
	if acc == nil {
		return nil, fmt.Errorf("account '%s' not found for agent '%s'", accountName, agentName)
	}

	total := acc.TotalQuota
	used := acc.UsedQuota
	resetAt := acc.QuotaResetAt
	model := acc.QuotaModel

	// Defaults if not explicitly configured
	if total <= 0 {
		if agentName == "antigravity" {
			total = 300
		} else if agentName == "codex" {
			total = 500
		} else {
			total = 200
		}
	}

	if resetAt.IsZero() {
		resetAt = time.Now().Add(24 * time.Hour)
	}

	// Automatic rolling window reset if reset time has passed
	if time.Now().After(resetAt) {
		used = 0
		resetAt = time.Now().Add(24 * time.Hour)
		_ = cfg.UpdateAccountQuota(agentName, accountName, total, used, 24*time.Hour, model)
		_ = cfg.Save()
	}

	resetsIn := time.Until(resetAt)
	if resetsIn < 0 {
		resetsIn = 0
	}

	return &QuotaInfo{
		AgentName:   agentName,
		AccountName: accountName,
		Used:        used,
		Total:       total,
		ResetsIn:    resetsIn,
		Model:       model,
	}, nil
}