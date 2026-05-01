package bridge

import "time"

// Payload is sent from the Go listener to GAS Web App.
type Payload struct {
	TriggeredAt     time.Time `json:"triggeredAt"`
	Host            string    `json:"host"`
	VscodeRunning   bool      `json:"vscodeRunning"`
	VscodeProcess   string    `json:"vscodeProcess"`
	SourceLogFile   string    `json:"sourceLogFile"`
	LatestUser      string    `json:"latestUser"`
	LatestAssistant string    `json:"latestAssistant"`
	RawExcerpt      string    `json:"rawExcerpt"`
}
