package qualityhistory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"novelstudio/internal/validation"
)

type Record struct { ID string `json:"id"`; ProjectID string `json:"projectId"`; DocumentID string `json:"documentId,omitempty"`; VersionID string `json:"versionId,omitempty"`; TextHash string `json:"textHash"`; Score int `json:"score"`; Verdict string `json:"verdict"`; GateStatus string `json:"gateStatus"`; Result validation.PipelineResult `json:"result"`; CreatedAt time.Time `json:"createdAt"` }
type Store interface { Save(context.Context,Record)error; List(context.Context,string,int)([]Record,error) }
type MemoryStore struct{mu sync.RWMutex;items []Record}
func NewMemoryStore()*MemoryStore{return &MemoryStore{}}
func(s *MemoryStore)Save(_ context.Context,item Record)error{s.mu.Lock();s.items=append(s.items,item);s.mu.Unlock();return nil}
func(s *MemoryStore)List(_ context.Context,projectID string,limit int)([]Record,error){s.mu.RLock();defer s.mu.RUnlock();if limit<1{limit=50};result:=[]Record{};for i:=len(s.items)-1;i>=0&&len(result)<limit;i--{if projectID==""||s.items[i].ProjectID==projectID{result=append(result,s.items[i])}};return result,nil}
func NewID()string{raw:=make([]byte,12);_,_=rand.Read(raw);return "val_"+hex.EncodeToString(raw)}
