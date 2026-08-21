package quota

import (
	"fmt"
	"time"
)

// QuotaInfo represents quota information for an account
type QuotaInfo struct {
	AgentName   string
	AccountName string
	Used        int
	Total       int
	ResetsIn    time.Duration
	Error       string
}

// Remaining returns remaining quota
func (q *QuotaInfo) Remaining() int {
	return q.Total - q.Used
}

// Percent returns the remaining percentage
func (q *QuotaInfo) Percent() float64 {
	if q.Total == 0 {
		return 0
	}
	return float64(q.Remaining()) / float64(q.Total) * 100
}

// ResetsInString returns human readable reset time
func (q *QuotaInfo) ResetsInString() string {
	if q.ResetsIn <= 0 {
		return "unknown"
	}
	h := int(q.ResetsIn.Hours())
	m := int(q.ResetsIn.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// CheckQuota checks quota for an agent account
// For v1 this is a placeholder that reads from config
// Future versions will call actual agent APIs
func CheckQuota(agentName, accountName string) (*QuotaInfo, error) {
	// TODO: Implement actual API calls per agent
	// For now, return placeholder data indicating API integration needed
	return &QuotaInfo{
		AgentName:   agentName,
		AccountName: accountName,
		Used:        0,
		Total:       0,
		ResetsIn:    0,
		Error:       "API integration pending - use 'agpass quota --set' to configure manually",
	}, nil
}
