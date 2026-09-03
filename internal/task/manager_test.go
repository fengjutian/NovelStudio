package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"novelstudio/internal/task"
)

func TestQueueLimitsConcurrentTasks(t *testing.T) {
	manager := task.NewManager()
	manager.ConfigureQueue(2, 10)
	release := make(chan struct{})
	started := make(chan struct{}, 3)
	items := make([]task.Task, 0, 3)
	for i := 0; i < 3; i++ {
		items = append(items, manager.Create("p1", "WRITE", func(ctx context.Context, _ func(int, string)) (any, error) {
			started <- struct{}{}
			select {
			case <-release:
				return "ok", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}))
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not start")
		}
	}
	select {
	case <-started:
		t.Fatal("third task started before a worker was available")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	for _, item := range items {
		waitForStatus(t, manager, item.ID, task.StatusSuccess)
	}
}

func TestQueuedTaskCanBeCancelled(t *testing.T) {
	manager := task.NewManager()
	manager.ConfigureQueue(1, 2)
	release := make(chan struct{})
	first := manager.Create("p1", "WRITE", func(context.Context, func(int, string)) (any, error) {
		<-release
		return "ok", nil
	})
	secondRan := make(chan struct{}, 1)
	second := manager.Create("p1", "WRITE", func(context.Context, func(int, string)) (any, error) {
		secondRan <- struct{}{}
		return "unexpected", nil
	})
	if err := manager.Cancel(second.ID); err != nil {
		t.Fatal(err)
	}
	close(release)
	waitForStatus(t, manager, first.ID, task.StatusSuccess)
	waitForStatus(t, manager, second.ID, task.StatusCancelled)
	select {
	case <-secondRan:
		t.Fatal("cancelled queued task executed")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestFailedTaskCanBeRetried(t *testing.T) {
	manager := task.NewManager()
	attempts := 0
	item := manager.Create("p1", "WRITE", func(context.Context, func(int, string)) (any, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("temporary failure")
		}
		return "ok", nil
	})
	waitForStatus(t, manager, item.ID, task.StatusFailed)
	retried, err := manager.Retry(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, manager, retried.ID, task.StatusSuccess)
}

func TestTaskTimesOut(t *testing.T) {
	manager := task.NewManager()
	manager.SetTimeout(5 * time.Millisecond)
	item := manager.Create("p1", "SLOW", func(ctx context.Context, _ func(int, string)) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	failed := waitForStatus(t, manager, item.ID, task.StatusFailed)
	if !errors.Is(errors.New(failed.Error), context.DeadlineExceeded) && failed.Error != context.DeadlineExceeded.Error() {
		t.Fatalf("unexpected error: %s", failed.Error)
	}
}

func TestTaskCompletesAndReplaysEvents(t *testing.T) {
	manager := task.NewManager()
	item := manager.Create("p1", "VALIDATE", func(_ context.Context, progress func(int, string)) (any, error) {
		progress(50, "half")
		return map[string]string{"gate": "PASS"}, nil
	})
	completed := waitForStatus(t, manager, item.ID, task.StatusSuccess)
	if completed.Progress != 100 || completed.Result == nil {
		t.Fatalf("unexpected task: %#v", completed)
	}
	events, err := manager.EventsSince(item.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 4 || events[len(events)-1].Type != "task.completed" {
		t.Fatalf("unexpected events: %#v", events)
	}
	replayed, _ := manager.EventsSince(item.ID, events[len(events)-2].ID)
	if len(replayed) != 1 || replayed[0].Type != "task.completed" {
		t.Fatalf("unexpected replay: %#v", replayed)
	}
}

func TestTaskCancellationIsTerminal(t *testing.T) {
	manager := task.NewManager()
	started := make(chan struct{})
	item := manager.Create("p1", "VALIDATE", func(ctx context.Context, _ func(int, string)) (any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	<-started
	if err := manager.Cancel(item.ID); err != nil {
		t.Fatal(err)
	}
	cancelled := waitForStatus(t, manager, item.ID, task.StatusCancelled)
	time.Sleep(10 * time.Millisecond)
	after, _ := manager.Get(item.ID)
	if cancelled.Status != task.StatusCancelled || after.Status != task.StatusCancelled {
		t.Fatalf("cancelled task changed state: %#v", after)
	}
}

func waitForStatus(t *testing.T, manager *task.Manager, id string, status task.Status) task.Task {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		item, err := manager.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if item.Status == status {
			return item
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("task %s did not reach %s", id, status)
	return task.Task{}
}
