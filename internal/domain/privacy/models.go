package privacy

import "time"

type Consent struct {
	Purpose       string    `json:"purpose"`
	PolicyVersion string    `json:"policy_version"`
	Granted       bool      `json:"granted"`
	Source        string    `json:"source"`
	CreatedAt     time.Time `json:"created_at"`
}

type Request struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Details   string    `json:"details"`
	DueAt     time.Time `json:"due_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Center struct {
	Consents       []Consent `json:"consents"`
	Requests       []Request `json:"requests"`
	EmailMarketing bool      `json:"email_marketing"`
}
