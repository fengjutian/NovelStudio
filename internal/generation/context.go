package generation

// ContextManifest records the immutable inputs used for one model call. It is
// intentionally ID-oriented so a historical run can be audited without
// copying the complete prompt into the database.
type ContextManifest struct {
	ProjectID                string              `json:"projectId,omitempty"`
	BranchID                 string              `json:"branchId,omitempty"`
	CanonVersion             int64               `json:"canonVersion"`
	ChapterNodeID            string              `json:"chapterNodeId,omitempty"`
	PreviousChapterVersionID string              `json:"previousChapterVersionId,omitempty"`
	EvidenceIDs              []string            `json:"evidenceIds"`
	EntityIDs                []string            `json:"entityIds,omitempty"`
	FactIDs                  []string            `json:"factIds,omitempty"`
	EventIDs                 []string            `json:"eventIds,omitempty"`
	PromptVersion            string              `json:"promptVersion,omitempty"`
	Model                    string              `json:"model,omitempty"`
	Truncations              []ContextTruncation `json:"truncations"`
}

type ContextTruncation struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
	Count  int    `json:"count,omitempty"`
}

func finalizeManifest(base ContextManifest, request Request, evidenceIDs []string, model, promptVersion string) ContextManifest {
	base.ProjectID = request.ProjectID
	if base.BranchID == "" {
		base.BranchID = "main"
	}
	base.EvidenceIDs = append([]string(nil), evidenceIDs...)
	base.Model = model
	base.PromptVersion = promptVersion
	if base.Truncations == nil {
		base.Truncations = []ContextTruncation{}
	}
	return base
}
