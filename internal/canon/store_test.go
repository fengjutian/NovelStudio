package canon

import (
	"context"
	"errors"
	"testing"
)

func TestCommitIsAtomicAndVersioned(t *testing.T) {
	s := NewMemoryStore()
	c := ChangeSet{ProjectID: "p1", ChapterVersionID: "v1", ApprovedBy: "author", Entities: []Entity{{ID: "c1", Type: EntityCharacter, Name: "林川"}}, Events: []Event{{ID: "e1", StoryTime: "day-1", Description: "受伤", ParticipantIDs: []string{"c1"}, Source: Evidence{DocumentVersionID: "v1", StartOffset: 1, EndOffset: 3}}}, Transitions: []StateTransition{{ID: "s1", EntityID: "c1", Field: "health", Before: nil, After: "injured", StoryTime: "day-1", EventID: "e1", Source: Evidence{DocumentVersionID: "v1", StartOffset: 1, EndOffset: 3}}}}
	got, err := s.Commit(context.Background(), c)
	if err != nil || got.Version != 1 {
		t.Fatalf("commit=%#v err=%v", got, err)
	}
	c.BaseCanonVersion = 0
	c.ChapterVersionID = "v2"
	c.Entities = nil
	c.Events = nil
	c.Transitions = []StateTransition{{ID: "s2", EntityID: "c1", Field: "health", Before: "injured", After: "well", StoryTime: "day-2", Source: Evidence{DocumentVersionID: "v2", StartOffset: 0, EndOffset: 2}}}
	if _, err = s.Commit(context.Background(), c); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict: %v", err)
	}
	state, _ := s.State(context.Background(), "p1", "main")
	if state["c1"]["health"] != "injured" {
		t.Fatalf("partial write: %#v", state)
	}
}
func TestRejectsWrongBefore(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Commit(context.Background(), ChangeSet{ProjectID: "p", ChapterVersionID: "v", ApprovedBy: "a", Entities: []Entity{{ID: "c", Type: EntityCharacter, Name: "C"}}, Transitions: []StateTransition{{ID: "s", EntityID: "c", Field: "place", Before: "north", After: "south", StoryTime: "t", Source: Evidence{DocumentVersionID: "v", StartOffset: 0, EndOffset: 1}}}})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict: %v", err)
	}
	v, _ := s.Version(context.Background(), "p", "main")
	if v != 0 {
		t.Fatalf("version=%d", v)
	}
}
func TestRejectsUnknownParticipant(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Commit(context.Background(), ChangeSet{ProjectID: "p", ChapterVersionID: "v", ApprovedBy: "a", Events: []Event{{ID: "e", StoryTime: "t", Description: "x", ParticipantIDs: []string{"missing"}, Source: Evidence{DocumentVersionID: "v", StartOffset: 0, EndOffset: 1}}}})
	if !errors.Is(err, ErrUnknownRef) {
		t.Fatalf("want unknown ref: %v", err)
	}
}
