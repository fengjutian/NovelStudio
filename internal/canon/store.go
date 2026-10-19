package canon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrConflict   = errors.New("canon version conflict")
	ErrInvalid    = errors.New("invalid canon change set")
	ErrUnknownRef = errors.New("canon change set references an unknown entity or event")
)

type Store interface {
	Version(context.Context, string, string) (int64, error)
	Commit(context.Context, ChangeSet) (Commit, error)
	State(context.Context, string, string) (map[string]map[string]any, error)
	RegisterDependencies(context.Context, ChapterDependencies) error
	AnalyzeImpact(context.Context, ImpactRequest) (ImpactResult, error)
	ListStale(context.Context, string, string) ([]StaleChapter, error)
}
type branchState struct {
	version  int64
	entities map[string]Entity
	events   map[string]Event
	values   map[string]map[string]any
}
type MemoryStore struct {
	mu       sync.RWMutex
	sequence atomic.Uint64
	branches map[string]*branchState
	dependencies map[string]ChapterDependencies
	stale map[string]StaleChapter
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{branches: map[string]*branchState{}, dependencies: map[string]ChapterDependencies{}, stale: map[string]StaleChapter{}} }
func key(p, b string) string       { return p + "\x00" + branch(b) }
func branch(v string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return "main"
}

func (s *MemoryStore) Version(_ context.Context, p, b string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if x := s.branches[key(p, b)]; x != nil {
		return x.version, nil
	}
	return 0, nil
}
func (s *MemoryStore) State(_ context.Context, p, b string) (map[string]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if x := s.branches[key(p, b)]; x != nil {
		return cloneState(x.values), nil
	}
	return map[string]map[string]any{}, nil
}

func (s *MemoryStore) Commit(_ context.Context, c ChangeSet) (Commit, error) {
	c.ProjectID = strings.TrimSpace(c.ProjectID)
	c.BranchID = branch(c.BranchID)
	c.ChapterVersionID = strings.TrimSpace(c.ChapterVersionID)
	c.ApprovedBy = strings.TrimSpace(c.ApprovedBy)
	if c.ProjectID == "" || c.ChapterVersionID == "" || c.ApprovedBy == "" {
		return Commit{}, fmt.Errorf("%w: projectId, chapterVersionId and approvedBy are required", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(c.ProjectID, c.BranchID)
	cur := s.branches[k]
	if cur == nil {
		cur = &branchState{entities: map[string]Entity{}, events: map[string]Event{}, values: map[string]map[string]any{}}
	}
	if c.BaseCanonVersion != cur.version {
		return Commit{}, fmt.Errorf("%w: expected %d, current %d", ErrConflict, c.BaseCanonVersion, cur.version)
	}
	if err := validate(cur, c); err != nil {
		return Commit{}, err
	}
	next := cloneBranch(cur)
	version := cur.version + 1
	for _, x := range c.Entities {
		x.ProjectID = c.ProjectID
		next.entities[x.ID] = x
	}
	for _, x := range c.Events {
		x.ProjectID = c.ProjectID
		x.BranchID = c.BranchID
		x.CanonCommitVersion = version
		next.events[x.ID] = x
	}
	for _, x := range c.Transitions {
		v := next.values[x.EntityID]
		if v == nil {
			v = map[string]any{}
			next.values[x.EntityID] = v
		}
		v[x.Field] = clone(x.After)
	}
	result := Commit{ID: fmt.Sprintf("ccm_%06d", s.sequence.Add(1)), ProjectID: c.ProjectID, BranchID: c.BranchID, Version: version, BaseVersion: cur.version, ChapterVersionID: c.ChapterVersionID, Message: strings.TrimSpace(c.Message), ApprovedBy: c.ApprovedBy, CreatedAt: time.Now().UTC()}
	next.version = version
	s.branches[k] = next
	return result, nil
}

func validate(cur *branchState, c ChangeSet) error {
	entities := map[string]bool{}
	for id := range cur.entities {
		entities[id] = true
	}
	seen := map[string]bool{}
	for _, x := range c.Entities {
		if x.ID == "" || x.Name == "" || x.Type == "" || seen[x.ID] {
			return fmt.Errorf("%w: invalid or duplicate entity", ErrInvalid)
		}
		seen[x.ID] = true
		entities[x.ID] = true
	}
	events := map[string]bool{}
	for id := range cur.events {
		events[id] = true
	}
	for _, x := range c.Events {
		if x.ID == "" || seen[x.ID] || x.StoryTime == "" || x.Description == "" || events[x.ID] || !evidence(x.Source, c.ChapterVersionID) {
			return fmt.Errorf("%w: invalid event", ErrInvalid)
		}
		if x.LocationID != "" && !entities[x.LocationID] {
			return ErrUnknownRef
		}
		for _, id := range x.ParticipantIDs {
			if !entities[id] {
				return ErrUnknownRef
			}
		}
		events[x.ID] = true
		seen[x.ID] = true
	}
	for _, x := range c.Facts {
		if x.ID == "" || seen[x.ID] || x.Predicate == "" || !entities[x.SubjectID] || !evidence(x.Source, c.ChapterVersionID) {
			return fmt.Errorf("%w: invalid fact", ErrInvalid)
		}
		seen[x.ID] = true
	}
	working := cloneState(cur.values)
	for _, x := range c.Transitions {
		if x.ID == "" || seen[x.ID] || x.Field == "" || x.StoryTime == "" || !entities[x.EntityID] || !evidence(x.Source, c.ChapterVersionID) {
			return fmt.Errorf("%w: invalid state transition", ErrInvalid)
		}
		seen[x.ID] = true
		if x.EventID != "" && !events[x.EventID] {
			return ErrUnknownRef
		}
		v := working[x.EntityID]
		if v == nil {
			v = map[string]any{}
			working[x.EntityID] = v
		}
		if !reflect.DeepEqual(v[x.Field], x.Before) {
			return fmt.Errorf("%w: stale before value for %s.%s", ErrConflict, x.EntityID, x.Field)
		}
		v[x.Field] = clone(x.After)
	}
	return nil
}
func evidence(e Evidence, v string) bool {
	return e.DocumentVersionID == v && e.StartOffset >= 0 && e.EndOffset > e.StartOffset
}
func clone(v any) any {
	raw, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(raw, &out)
	return out
}
func cloneState(v map[string]map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for id, fields := range v {
		x := map[string]any{}
		for k, item := range fields {
			x[k] = clone(item)
		}
		out[id] = x
	}
	return out
}
func cloneBranch(v *branchState) *branchState {
	out := &branchState{version: v.version, entities: map[string]Entity{}, events: map[string]Event{}, values: cloneState(v.values)}
	for id, x := range v.entities {
		out.entities[id] = x
	}
	for id, x := range v.events {
		out.events[id] = x
	}
	return out
}
