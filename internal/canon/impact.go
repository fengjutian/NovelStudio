package canon

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func dependencyKey(projectID, branchID, chapterVersionID string) string {
	return branchKey(projectID, normalizedBranch(branchID)) + "\x00" + chapterVersionID
}

func (s *MemoryStore) RegisterDependencies(_ context.Context, input ChapterDependencies) error {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.BranchID = normalizedBranch(input.BranchID)
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
	request.BranchID = normalizedBranch(request.BranchID)
	if request.ProjectID == "" { return ImpactResult{}, fmt.Errorf("%w: projectId is required", ErrInvalid) }
	s.mu.Lock()
	defer s.mu.Unlock()
	result := analyzeDependencies(s.dependencies, request)
	if request.MarkStale {
		for _, item := range result.Affected { s.stale[dependencyKey(item.ProjectID,item.BranchID,item.ChapterVersionID)] = item }
	}
	return result, nil
}

func (s *MemoryStore) ListStale(_ context.Context, projectID, branchID string) ([]StaleChapter, error) {
	s.mu.RLock(); defer s.mu.RUnlock()
	branchID = normalizedBranch(branchID)
	items := []StaleChapter{}
	for _, item := range s.stale { if item.ProjectID == projectID && item.BranchID == branchID { items = append(items,item) } }
	sort.Slice(items,func(i,j int)bool{return items[i].CreatedAt.After(items[j].CreatedAt)})
	return items,nil
}

func analyzeDependencies(all map[string]ChapterDependencies, request ImpactRequest) ImpactResult {
	facts,entities,events,chapters := stringSet(request.ChangedFactIDs),stringSet(request.ChangedEntityIDs),stringSet(request.ChangedEventIDs),stringSet(request.ChangedChapterVersionIDs)
	reasons := map[string]map[string]bool{}
	changed := true
	for changed {
		changed = false
		for _, dep := range all {
			if dep.ProjectID != request.ProjectID || normalizedBranch(dep.BranchID) != request.BranchID || chapters[dep.ChapterVersionID] { continue }
			matched := []string{}
			for _,id:=range dep.FactIDs{if facts[id]{matched=append(matched,"FACT:"+id)}}
			for _,id:=range dep.EntityIDs{if entities[id]{matched=append(matched,"ENTITY:"+id)}}
			for _,id:=range dep.EventIDs{if events[id]{matched=append(matched,"EVENT:"+id)}}
			for _,id:=range dep.DependsOnChapterVersionIDs{if chapters[id]{matched=append(matched,"CHAPTER_VERSION:"+id)}}
			if len(matched)>0 { chapters[dep.ChapterVersionID]=true; changed=true; if reasons[dep.ChapterVersionID]==nil{reasons[dep.ChapterVersionID]=map[string]bool{}};for _,r:=range matched{reasons[dep.ChapterVersionID][r]=true} }
		}
	}
	now:=time.Now().UTC(); result:=ImpactResult{}
	for chapterID,why:=range reasons { values:=make([]string,0,len(why));for reason:=range why{values=append(values,reason)};sort.Strings(values);result.Affected=append(result.Affected,StaleChapter{ProjectID:request.ProjectID,BranchID:request.BranchID,ChapterVersionID:chapterID,TriggerCanonVersion:request.TriggerCanonVersion,Reasons:values,CreatedAt:now}) }
	sort.Slice(result.Affected,func(i,j int)bool{return result.Affected[i].ChapterVersionID<result.Affected[j].ChapterVersionID})
	return result
}

func unique(values []string) []string { set:=stringSet(values); result:=make([]string,0,len(set));for value:=range set{result=append(result,value)};sort.Strings(result);return result }
func stringSet(values []string) map[string]bool { result:=map[string]bool{};for _,value:=range values{if value=strings.TrimSpace(value);value!=""{result[value]=true}};return result }
