package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRunning   Status = "RUNNING"
	StatusSuccess   Status = "SUCCESS"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"projectId"`
	Type      string     `json:"type"`
	Status    Status     `json:"status"`
	Progress  int        `json:"progress"`
	Message   string     `json:"message"`
	Result    any        `json:"result,omitempty"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

type Event struct {
	ID        uint64    `json:"id"`
	TaskID    string    `json:"taskId"`
	Type      string    `json:"type"`
	Progress  int       `json:"progress"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type Executor func(context.Context, func(int, string)) (any, error)

type Manager struct {
	mu          sync.RWMutex
	sequence    atomic.Uint64
	eventSeq    atomic.Uint64
	tasks       map[string]Task
	events      map[string][]Event
	cancels     map[string]context.CancelFunc
	listeners   map[string]map[uint64]chan Event
	listenerSeq atomic.Uint64
}

func NewManager() *Manager {
	return &Manager{tasks: make(map[string]Task), events: make(map[string][]Event), cancels: make(map[string]context.CancelFunc), listeners: make(map[string]map[uint64]chan Event)}
}

func (m *Manager) Create(projectID, taskType string, executor Executor) Task {
	now := time.Now().UTC()
	item := Task{ID: fmt.Sprintf("tsk_%06d", m.sequence.Add(1)), ProjectID: projectID, Type: taskType, Status: StatusPending, Progress: 0, Message: "任务已创建", CreatedAt: now}
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.tasks[item.ID] = item
	m.cancels[item.ID] = cancel
	m.mu.Unlock()
	m.publish(item.ID, "task.created", 0, item.Message)
	go m.execute(ctx, item.ID, executor)
	return item
}

func (m *Manager) Get(id string) (Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return item, nil
}

func (m *Manager) List(projectID string) []Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]Task, 0)
	for _, item := range m.tasks {
		if projectID == "" || item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	item, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotFound
	}
	if terminal(item.Status) {
		m.mu.Unlock()
		return nil
	}
	cancel := m.cancels[id]
	item.Status, item.Message = StatusCancelled, "任务已取消"
	item.Progress = min(item.Progress, 99)
	now := time.Now().UTC()
	item.EndedAt = &now
	m.tasks[id] = item
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.publish(id, "task.cancelled", item.Progress, item.Message)
	return nil
}

func (m *Manager) EventsSince(id string, after uint64) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.tasks[id]; !ok {
		return nil, ErrNotFound
	}
	var items []Event
	for _, event := range m.events[id] {
		if event.ID > after {
			items = append(items, event)
		}
	}
	return items, nil
}

func (m *Manager) Subscribe(id string) (<-chan Event, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[id]; !ok {
		return nil, nil, ErrNotFound
	}
	listenerID := m.listenerSeq.Add(1)
	channel := make(chan Event, 16)
	if m.listeners[id] == nil {
		m.listeners[id] = make(map[uint64]chan Event)
	}
	m.listeners[id][listenerID] = channel
	remove := func() {
		m.mu.Lock()
		if listeners := m.listeners[id]; listeners != nil {
			delete(listeners, listenerID)
		}
		m.mu.Unlock()
	}
	return channel, remove, nil
}

func (m *Manager) execute(ctx context.Context, id string, executor Executor) {
	now := time.Now().UTC()
	if !m.transition(id, StatusPending, func(item *Task) {
		item.Status, item.StartedAt, item.Progress, item.Message = StatusRunning, &now, 1, "任务开始执行"
	}) {
		return
	}
	m.publish(id, "task.started", 1, "任务开始执行")
	result, err := executor(ctx, func(progress int, message string) {
		if m.transition(id, StatusRunning, func(item *Task) {
			item.Progress, item.Message = max(1, min(progress, 99)), message
		}) {
			m.publish(id, "task.progress", progress, message)
		}
	})
	ended := time.Now().UTC()
	if err != nil {
		if !m.transition(id, StatusRunning, func(item *Task) {
			item.Status, item.Error, item.Message, item.EndedAt = StatusFailed, err.Error(), "任务执行失败", &ended
		}) {
			return
		}
		item, _ := m.Get(id)
		m.publish(id, "task.failed", item.Progress, err.Error())
		return
	}
	if !m.transition(id, StatusRunning, func(item *Task) {
		item.Status, item.Result, item.Progress, item.Message, item.EndedAt = StatusSuccess, result, 100, "任务执行完成", &ended
	}) {
		return
	}
	m.publish(id, "task.completed", 100, "任务执行完成")
}

func (m *Manager) transition(id string, from Status, update func(*Task)) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.tasks[id]
	if !ok || item.Status != from {
		return false
	}
	update(&item)
	m.tasks[id] = item
	return true
}

func (m *Manager) publish(id, eventType string, progress int, message string) {
	event := Event{ID: m.eventSeq.Add(1), TaskID: id, Type: eventType, Progress: progress, Message: message, CreatedAt: time.Now().UTC()}
	m.mu.Lock()
	m.events[id] = append(m.events[id], event)
	for _, listener := range m.listeners[id] {
		select {
		case listener <- event:
		default:
		}
	}
	m.mu.Unlock()
}

func terminal(status Status) bool {
	return status == StatusSuccess || status == StatusFailed || status == StatusCancelled
}
