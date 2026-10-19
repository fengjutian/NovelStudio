package canon

import "time"

type EntityType string

const (
	EntityCharacter    EntityType = "CHARACTER"
	EntityLocation     EntityType = "LOCATION"
	EntityOrganization EntityType = "ORGANIZATION"
	EntityItem         EntityType = "ITEM"
	EntityAbility      EntityType = "ABILITY"
)

type Entity struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	Type      EntityType     `json:"type"`
	Name      string         `json:"name"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
type Evidence struct {
	DocumentVersionID string `json:"documentVersionId"`
	StartOffset       int    `json:"startOffset"`
	EndOffset         int    `json:"endOffset"`
}
type Fact struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"projectId"`
	SubjectID          string    `json:"subjectId"`
	Predicate          string    `json:"predicate"`
	Value              any       `json:"value"`
	ValidFrom          string    `json:"validFrom,omitempty"`
	ValidTo            string    `json:"validTo,omitempty"`
	KnownBy            []string  `json:"knownBy,omitempty"`
	RevealedAt         string    `json:"revealedToReaderAt,omitempty"`
	Certainty          string    `json:"certainty"`
	BranchID           string    `json:"branchId"`
	Source             Evidence  `json:"source"`
	RecordedAt         time.Time `json:"recordedAt"`
	CanonCommitVersion int64     `json:"canonCommitVersion"`
}
type Event struct {
	ID                 string   `json:"id"`
	ProjectID          string   `json:"projectId"`
	StoryTime          string   `json:"storyTime"`
	Duration           string   `json:"duration,omitempty"`
	LocationID         string   `json:"locationId,omitempty"`
	ParticipantIDs     []string `json:"participantIds,omitempty"`
	Description        string   `json:"description"`
	BranchID           string   `json:"branchId"`
	Source             Evidence `json:"source"`
	CanonCommitVersion int64    `json:"canonCommitVersion"`
}
type StateTransition struct {
	ID                 string   `json:"id"`
	ProjectID          string   `json:"projectId"`
	EntityID           string   `json:"entityId"`
	Field              string   `json:"field"`
	Before             any      `json:"before"`
	After              any      `json:"after"`
	StoryTime          string   `json:"storyTime"`
	EventID            string   `json:"eventId,omitempty"`
	Source             Evidence `json:"source"`
	CanonCommitVersion int64    `json:"canonCommitVersion"`
}
type ChangeSet struct {
	ProjectID        string            `json:"projectId"`
	BranchID         string            `json:"branchId"`
	BaseCanonVersion int64             `json:"baseCanonVersion"`
	ChapterVersionID string            `json:"chapterVersionId"`
	Message          string            `json:"message"`
	ApprovedBy       string            `json:"approvedBy"`
	Entities         []Entity          `json:"entities,omitempty"`
	Facts            []Fact            `json:"facts,omitempty"`
	Events           []Event           `json:"events,omitempty"`
	Transitions      []StateTransition `json:"transitions,omitempty"`
}
type Commit struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	BranchID         string    `json:"branchId"`
	Version          int64     `json:"version"`
	BaseVersion      int64     `json:"baseVersion"`
	ChapterVersionID string    `json:"chapterVersionId"`
	Message          string    `json:"message"`
	ApprovedBy       string    `json:"approvedBy"`
	CreatedAt        time.Time `json:"createdAt"`
}

type ChapterDependencies struct {
	ProjectID                  string   `json:"projectId"`
	BranchID                   string   `json:"branchId"`
	ChapterVersionID           string   `json:"chapterVersionId"`
	CanonVersion               int64    `json:"canonVersion"`
	FactIDs                    []string `json:"factIds,omitempty"`
	EntityIDs                  []string `json:"entityIds,omitempty"`
	EventIDs                   []string `json:"eventIds,omitempty"`
	DependsOnChapterVersionIDs []string `json:"dependsOnChapterVersionIds,omitempty"`
}

type ImpactRequest struct {
	ProjectID                string   `json:"projectId"`
	BranchID                 string   `json:"branchId"`
	TriggerCanonVersion      int64    `json:"triggerCanonVersion"`
	ChangedFactIDs           []string `json:"changedFactIds,omitempty"`
	ChangedEntityIDs         []string `json:"changedEntityIds,omitempty"`
	ChangedEventIDs          []string `json:"changedEventIds,omitempty"`
	ChangedChapterVersionIDs []string `json:"changedChapterVersionIds,omitempty"`
	MarkStale                bool     `json:"markStale"`
}

type StaleChapter struct {
	ProjectID           string     `json:"projectId"`
	BranchID            string     `json:"branchId"`
	ChapterVersionID    string     `json:"chapterVersionId"`
	TriggerCanonVersion int64      `json:"triggerCanonVersion"`
	Reasons             []string   `json:"reasons"`
	Paths               [][]string `json:"paths,omitempty"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"createdAt"`
	ResolvedAt          *time.Time `json:"resolvedAt,omitempty"`
	ResolvedBy          string     `json:"resolvedBy,omitempty"`
	ResolutionNote      string     `json:"resolutionNote,omitempty"`
}

type ImpactResult struct {
	Affected []StaleChapter `json:"affected"`
}

type ResolveStaleInput struct {
	Status     string `json:"status"`
	ResolvedBy string `json:"resolvedBy"`
	Note       string `json:"note,omitempty"`
}
