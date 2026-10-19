package canon

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func dependencyKey(projectID, branchID, chapterVersionID string) string {
	return key(projectID, branch(branchID)) + "\x00" + chapterVersionID
}

func (s *MemoryStore) RegisterDependencies(_ context.Context, input ChapterDependencies) error {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.BranchID = branch(input.BranchID)
	input.ChapterVersionID = strings.TrimSpace(input.ChapterVersionID)
	if input.ProjectID == "" || input.ChapterVersionID == "" || input.CanonVersion < 0 {
		return fmt.Errorf("%w: projectId, chapterVersionId and non-negative canonVersion are required", ErrInvalid)
	}
	input.FactIDs = unique(input.FactIDs)
	input.EntityIDs = unique(input.EntityIDs)
	input.EventIDs = unique(input.EventIDs)
	input.DependsOnChapterVersionIDs = unique(input.DependsOnChapterVersionIDs)
	s.mu.Lock()
	s.dependencies[dependencyKey(input.ProjectID, input.BranchID, input.ChapterVersionID)] = input
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) AnalyzeImpact(_ context.Context, request ImpactRequest) (ImpactResult, error) {
	request.ProjectID = strings.TrimSpace(request.ProjectID)
	request.BranchID = branch(request.BranchID)
	if request.ProjectID == "" {
		return ImpactResult{}, fmt.Errorf("%w: projectId is required", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := analyzeDependencies(s.dependencies, request)
	if request.MarkStale {
		for _, item := range result.Affected {
			s.stale[dependencyKey(item.ProjectID, item.BranchID, item.ChapterVersionID)] = item
		}
	}
	return result, nil
}

func (s *MemoryStore) ListStale(_ context.Context, projectID, branchID string) ([]StaleChapter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	branchID = branch(branchID)
	items := []StaleChapter{}
	for _, item := range s.stale {
		if item.ProjectID == projectID && item.BranchID == branchID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) ResolveStale(_ context.Context, projectID, branchID, chapterVersionID string, input ResolveStaleInput) (StaleChapter, error) {
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.ResolvedBy = strings.TrimSpace(input.ResolvedBy)
	if input.Status != "REVIEWING" && input.Status != "VALID" && input.Status != "ACCEPTED_RISK" {
		return StaleChapter{}, fmt.Errorf("%w: invalid stale status", ErrInvalid)
	}
	if input.Status != "REVIEWING" && input.ResolvedBy == "" {
		return StaleChapter{}, fmt.Errorf("%w: resolvedBy is required", ErrInvalid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := dependencyKey(projectID, branchID, chapterVersionID)
	item, ok := s.stale[k]
	if !ok {
		return StaleChapter{}, ErrUnknownRef
	}
	item.Status, item.ResolvedBy, item.ResolutionNote = input.Status, input.ResolvedBy, strings.TrimSpace(input.Note)
	if input.Status == "REVIEWING" {
		item.ResolvedAt = nil
	} else {
		now := time.Now().UTC()
		item.ResolvedAt = &now
	}
	s.stale[k] = item
	return item, nil
}

func analyzeDependencies(all map[string]ChapterDependencies, request ImpactRequest) ImpactResult {
	facts, entities, events, chapters := stringSet(request.ChangedFactIDs), stringSet(request.ChangedEntityIDs), stringSet(request.ChangedEventIDs), stringSet(request.ChangedChapterVersionIDs)
	reasons := map[string]map[string]bool{}
	paths := map[string][][]string{}
	changed := true
	for changed {
		changed = false
		for _, dep := range all {
			if dep.ProjectID != request.ProjectID || branch(dep.BranchID) != request.BranchID || chapters[dep.ChapterVersionID] {
				continue
			}
			matched := []string{}
			for _, id := range dep.FactIDs {
				if facts[id] {
					matched = append(matched, "FACT:"+id)
					paths[dep.ChapterVersionID] = append(paths[dep.ChapterVersionID], []string{"FACT:" + id, dep.ChapterVersionID})
				}
			}
			for _, id := range dep.EntityIDs {
				if entities[id] {
					matched = append(matched, "ENTITY:"+id)
					paths[dep.ChapterVersionID] = append(paths[dep.ChapterVersionID], []string{"ENTITY:" + id, dep.ChapterVersionID})
				}
			}
			for _, id := range dep.EventIDs {
				if events[id] {
					matched = append(matched, "EVENT:"+id)
					paths[dep.ChapterVersionID] = append(paths[dep.ChapterVersionID], []string{"EVENT:" + id, dep.ChapterVersionID})
				}
			}
			for _, id := range dep.DependsOnChapterVersionIDs {
				if chapters[id] {
					matched = append(matched, "CHAPTER_VERSION:"+id)
					for _, path := range paths[id] {
						paths[dep.ChapterVersionID] = append(paths[dep.ChapterVersionID], append(append([]string{}, path...), dep.ChapterVersionID))
					}
				}
			}
			if len(matched) > 0 {
				chapters[dep.ChapterVersionID] = true
				changed = true
				if reasons[dep.ChapterVersionID] == nil {
					reasons[dep.ChapterVersionID] = map[string]bool{}
				}
				for _, r := range matched {
					reasons[dep.ChapterVersionID][r] = true
				}
			}
		}
	}
	now := time.Now().UTC()
	result := ImpactResult{}
	for chapterID, why := range reasons {
		values := make([]string, 0, len(why))
		for reason := range why {
			values = append(values, reason)
		}
		sort.Strings(values)
		result.Affected = append(result.Affected, StaleChapter{ProjectID: request.ProjectID, BranchID: request.BranchID, ChapterVersionID: chapterID, TriggerCanonVersion: request.TriggerCanonVersion, Reasons: values, Paths: paths[chapterID], Status: "STALE", CreatedAt: now})
	}
	sort.Slice(result.Affected, func(i, j int) bool { return result.Affected[i].ChapterVersionID < result.Affected[j].ChapterVersionID })
	return result
}

func unique(values []string) []string {
	set := stringSet(values)
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = true
		}
	}
	return result
}
