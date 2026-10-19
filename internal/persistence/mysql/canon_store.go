package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"novelstudio/internal/canon"
)

type CanonStore struct{ DB *sql.DB }

var _ canon.Store = CanonStore{}

func (s CanonStore) Version(ctx context.Context, projectID, branchID string) (int64, error) {
	var version int64
	err := s.DB.QueryRowContext(ctx, `SELECT current_version FROM canon_branches WHERE project_id=? AND branch_id=?`, projectID, canonBranch(branchID)).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return version, err
}

func (s CanonStore) State(ctx context.Context, projectID, branchID string) (map[string]map[string]any, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT entity_id,field_path,after_json FROM state_transitions st JOIN (SELECT entity_id e,field_path f,MAX(canon_commit_version) v FROM state_transitions WHERE project_id=? AND branch_id=? GROUP BY entity_id,field_path) latest ON latest.e=st.entity_id AND latest.f=st.field_path AND latest.v=st.canon_commit_version WHERE st.project_id=? AND st.branch_id=?`, projectID, canonBranch(branchID), projectID, canonBranch(branchID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]any{}
	for rows.Next() {
		var entityID, field string
		var raw []byte
		if err := rows.Scan(&entityID, &field, &raw); err != nil {
			return nil, err
		}
		if result[entityID] == nil {
			result[entityID] = map[string]any{}
		}
		var value any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &value)
		}
		result[entityID][field] = value
	}
	return result, rows.Err()
}

func (s CanonStore) Commit(ctx context.Context, change canon.ChangeSet) (canon.Commit, error) {
	change.ProjectID = strings.TrimSpace(change.ProjectID)
	change.BranchID = canonBranch(change.BranchID)
	change.ChapterVersionID = strings.TrimSpace(change.ChapterVersionID)
	change.ApprovedBy = strings.TrimSpace(change.ApprovedBy)
	if change.ProjectID == "" || change.ChapterVersionID == "" || change.ApprovedBy == "" {
		return canon.Commit{}, fmt.Errorf("%w: projectId, chapterVersionId and approvedBy are required", canon.ErrInvalid)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return canon.Commit{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err = tx.ExecContext(ctx, `INSERT INTO canon_branches(project_id,branch_id,current_version,updated_at) VALUES(?,?,0,?) ON DUPLICATE KEY UPDATE project_id=project_id`, change.ProjectID, change.BranchID, now); err != nil {
		return canon.Commit{}, err
	}
	var current int64
	if err = tx.QueryRowContext(ctx, `SELECT current_version FROM canon_branches WHERE project_id=? AND branch_id=? FOR UPDATE`, change.ProjectID, change.BranchID).Scan(&current); err != nil {
		return canon.Commit{}, err
	}
	if current != change.BaseCanonVersion {
		return canon.Commit{}, fmt.Errorf("%w: expected %d, current %d", canon.ErrConflict, change.BaseCanonVersion, current)
	}
	var chapterProject string
	if err = tx.QueryRowContext(ctx, `SELECT d.project_id FROM document_versions v JOIN documents d ON d.id=v.document_id WHERE v.id=?`, change.ChapterVersionID).Scan(&chapterProject); err != nil || chapterProject != change.ProjectID {
		return canon.Commit{}, fmt.Errorf("%w: chapter version is not part of project", canon.ErrInvalid)
	}
	known := map[string]bool{}
	seen := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM story_entities WHERE project_id=?`, change.ProjectID)
	if err != nil {
		return canon.Commit{}, err
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		known[id] = true
	}
	rows.Close()
	for _, e := range change.Entities {
		if e.ID == "" || e.Name == "" || e.Type == "" || known[e.ID] || seen[e.ID] {
			return canon.Commit{}, fmt.Errorf("%w: invalid or duplicate entity", canon.ErrInvalid)
		}
		known[e.ID] = true
		seen[e.ID] = true
	}
	events := map[string]bool{}
	rows, err = tx.QueryContext(ctx, `SELECT id FROM story_events_v2 WHERE project_id=? AND branch_id=?`, change.ProjectID, change.BranchID)
	if err != nil {
		return canon.Commit{}, err
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		events[id] = true
	}
	rows.Close()
	validEvidence := func(e canon.Evidence) bool {
		return e.DocumentVersionID == change.ChapterVersionID && e.StartOffset >= 0 && e.EndOffset > e.StartOffset
	}
	for _, e := range change.Events {
		if e.ID == "" || seen[e.ID] || e.StoryTime == "" || e.Description == "" || events[e.ID] || !validEvidence(e.Source) {
			return canon.Commit{}, fmt.Errorf("%w: invalid event", canon.ErrInvalid)
		}
		if e.LocationID != "" && !known[e.LocationID] {
			return canon.Commit{}, canon.ErrUnknownRef
		}
		for _, id := range e.ParticipantIDs {
			if !known[id] {
				return canon.Commit{}, canon.ErrUnknownRef
			}
		}
		events[e.ID] = true
		seen[e.ID] = true
	}
	for _, f := range change.Facts {
		if f.ID == "" || seen[f.ID] || f.Predicate == "" || !known[f.SubjectID] || !validEvidence(f.Source) {
			return canon.Commit{}, fmt.Errorf("%w: invalid fact", canon.ErrInvalid)
		}
		seen[f.ID] = true
	}
	state, err := stateTx(ctx, tx, change.ProjectID, change.BranchID)
	if err != nil {
		return canon.Commit{}, err
	}
	for _, tr := range change.Transitions {
		if tr.ID == "" || seen[tr.ID] || tr.Field == "" || tr.StoryTime == "" || !known[tr.EntityID] || !validEvidence(tr.Source) {
			return canon.Commit{}, fmt.Errorf("%w: invalid transition", canon.ErrInvalid)
		}
		seen[tr.ID] = true
		if tr.EventID != "" && !events[tr.EventID] {
			return canon.Commit{}, canon.ErrUnknownRef
		}
		if state[tr.EntityID] == nil {
			state[tr.EntityID] = map[string]any{}
		}
		if !jsonEqual(state[tr.EntityID][tr.Field], tr.Before) {
			return canon.Commit{}, fmt.Errorf("%w: stale before value for %s.%s", canon.ErrConflict, tr.EntityID, tr.Field)
		}
		state[tr.EntityID][tr.Field] = tr.After
	}
	version := current + 1
	commit := canon.Commit{ID: newID("ccm"), ProjectID: change.ProjectID, BranchID: change.BranchID, Version: version, BaseVersion: current, ChapterVersionID: change.ChapterVersionID, Message: strings.TrimSpace(change.Message), ApprovedBy: change.ApprovedBy, CreatedAt: now}
	for _, e := range change.Entities {
		raw, _ := json.Marshal(e.Metadata)
		if _, err = tx.ExecContext(ctx, `INSERT INTO story_entities(id,project_id,entity_type,name,metadata,created_at) VALUES(?,?,?,?,?,?)`, e.ID, change.ProjectID, e.Type, e.Name, raw, now); err != nil {
			return canon.Commit{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO canon_commits(id,project_id,branch_id,version,base_version,chapter_version_id,message,approved_by,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, commit.ID, commit.ProjectID, commit.BranchID, commit.Version, commit.BaseVersion, commit.ChapterVersionID, commit.Message, commit.ApprovedBy, commit.CreatedAt); err != nil {
		return canon.Commit{}, err
	}
	for _, e := range change.Events {
		if _, err = tx.ExecContext(ctx, `INSERT INTO story_events_v2(id,project_id,branch_id,story_time,duration,location_id,description,source_version_id,start_offset,end_offset,canon_commit_version) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, e.ID, change.ProjectID, change.BranchID, e.StoryTime, nullable(e.Duration), nullable(e.LocationID), e.Description, e.Source.DocumentVersionID, e.Source.StartOffset, e.Source.EndOffset, version); err != nil {
			return canon.Commit{}, err
		}
		for _, id := range e.ParticipantIDs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO event_participants(event_id,entity_id) VALUES(?,?)`, e.ID, id); err != nil {
				return canon.Commit{}, err
			}
		}
	}
	for _, f := range change.Facts {
		value, _ := json.Marshal(f.Value)
		knownBy, _ := json.Marshal(f.KnownBy)
		certainty := f.Certainty
		if certainty == "" {
			certainty = "CONFIRMED"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO canon_facts_v2(id,project_id,branch_id,subject_id,predicate,value_json,valid_from,valid_to,known_by_json,revealed_to_reader_at,certainty,source_version_id,start_offset,end_offset,canon_commit_version,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, f.ID, change.ProjectID, change.BranchID, f.SubjectID, f.Predicate, value, nullable(f.ValidFrom), nullable(f.ValidTo), knownBy, nullable(f.RevealedAt), certainty, f.Source.DocumentVersionID, f.Source.StartOffset, f.Source.EndOffset, version, now); err != nil {
			return canon.Commit{}, err
		}
	}
	for _, tr := range change.Transitions {
		before, _ := json.Marshal(tr.Before)
		after, _ := json.Marshal(tr.After)
		if _, err = tx.ExecContext(ctx, `INSERT INTO state_transitions(id,project_id,branch_id,entity_id,field_path,before_json,after_json,story_time,event_id,source_version_id,start_offset,end_offset,canon_commit_version) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, tr.ID, change.ProjectID, change.BranchID, tr.EntityID, tr.Field, before, after, tr.StoryTime, nullable(tr.EventID), tr.Source.DocumentVersionID, tr.Source.StartOffset, tr.Source.EndOffset, version); err != nil {
			return canon.Commit{}, err
		}
	}
	payload, _ := json.Marshal(map[string]any{"projectId": change.ProjectID, "branchId": change.BranchID, "canonVersion": version, "chapterVersionId": change.ChapterVersionID})
	if _, err = tx.ExecContext(ctx, `INSERT INTO outbox_events(id,project_id,event_type,aggregate_id,payload,status,available_at,created_at) VALUES(?,?,?,?,?,'PENDING',?,?)`, newID("out"), change.ProjectID, "CANON_COMMITTED", commit.ID, payload, now, now); err != nil {
		return canon.Commit{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE canon_branches SET current_version=?,updated_at=? WHERE project_id=? AND branch_id=?`, version, now, change.ProjectID, change.BranchID); err != nil {
		return canon.Commit{}, err
	}
	if err = tx.Commit(); err != nil {
		return canon.Commit{}, err
	}
	return commit, nil
}

func stateTx(ctx context.Context, tx *sql.Tx, p, b string) (map[string]map[string]any, error) {
	rows, err := tx.QueryContext(ctx, `SELECT entity_id,field_path,after_json FROM state_transitions WHERE project_id=? AND branch_id=? ORDER BY canon_commit_version`, p, b)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]map[string]any{}
	for rows.Next() {
		var id, f string
		var raw []byte
		if err := rows.Scan(&id, &f, &raw); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = map[string]any{}
		}
		var v any
		_ = json.Unmarshal(raw, &v)
		out[id][f] = v
	}
	return out, rows.Err()
}
func jsonEqual(a, b any) bool {
	normalize := func(v any) any { raw, _ := json.Marshal(v); var x any; _ = json.Unmarshal(raw, &x); return x }
	return reflect.DeepEqual(normalize(a), normalize(b))
}
func canonBranch(v string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return "main"
}
