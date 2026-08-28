package validation

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityMajor    Severity = "MAJOR"
	SeverityMinor    Severity = "MINOR"
)

type Issue struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Severity       Severity `json:"severity"`
	Claim          string   `json:"claim"`
	Explanation    string   `json:"explanation"`
	SuggestedFix   string   `json:"suggestedFix"`
	EvidenceIDs    []string `json:"evidenceIds"`
	Confidence     float64  `json:"confidence"`
	StartCharacter int      `json:"startCharacter"`
	EndCharacter   int      `json:"endCharacter"`
}

type Result struct {
	Score      int            `json:"score"`
	Verdict    string         `json:"verdict"`
	Dimensions map[string]int `json:"dimensions"`
	Issues     []Issue        `json:"issues"`
}
