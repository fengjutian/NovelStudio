package document

import "strings"

type DiffLine struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	OldLine int    `json:"oldLine,omitempty"`
	NewLine int    `json:"newLine,omitempty"`
}

type DiffResult struct {
	FromVersionID string     `json:"fromVersionId"`
	ToVersionID   string     `json:"toVersionId"`
	Lines         []DiffLine `json:"lines"`
	Added         int        `json:"added"`
	Deleted       int        `json:"deleted"`
}

func Diff(from, to Version) DiffResult {
	a, b := splitLines(from.Content), splitLines(to.Content)
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	result := DiffResult{FromVersionID: from.ID, ToVersionID: to.ID, Lines: []DiffLine{}}
	for i, j := 0, 0; i < len(a) || j < len(b); {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			result.Lines = append(result.Lines, DiffLine{Type: "UNCHANGED", Content: a[i], OldLine: i + 1, NewLine: j + 1})
			i++
			j++
		case j < len(b) && (i == len(a) || dp[i][j+1] > dp[i+1][j]):
			result.Lines = append(result.Lines, DiffLine{Type: "ADDED", Content: b[j], NewLine: j + 1})
			result.Added++
			j++
		default:
			result.Lines = append(result.Lines, DiffLine{Type: "DELETED", Content: a[i], OldLine: i + 1})
			result.Deleted++
			i++
		}
	}
	return result
}

func splitLines(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
}
