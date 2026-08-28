package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"novelstudio/internal/task"
)

type TaskRepository struct{ DB *sql.DB }

var _ task.Repository = TaskRepository{}

func (r TaskRepository) SaveTask(ctx context.Context, item task.Task) error {
	var result any
	if item.Result != nil {
		raw, err := json.Marshal(item.Result)
		if err != nil {
			return err
		}
		result = raw
	}
	_, err := r.DB.ExecContext(ctx, `INSERT INTO ai_tasks(id,project_id,task_type,status,progress,message,result,error,created_at,started_at,ended_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),progress=VALUES(progress),message=VALUES(message),result=VALUES(result),error=VALUES(error),started_at=VALUES(started_at),ended_at=VALUES(ended_at)`, item.ID, item.ProjectID, item.Type, item.Status, item.Progress, item.Message, result, item.Error, item.CreatedAt, item.StartedAt, item.EndedAt)
	return err
}
func (r TaskRepository) AppendEvent(ctx context.Context, event task.Event) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO task_events(task_id,event_id,event_type,progress,message,created_at) VALUES(?,?,?,?,?,?)`, event.TaskID, event.ID, event.Type, event.Progress, event.Message, event.CreatedAt)
	return err
}
func (r TaskRepository) GetTask(ctx context.Context, id string) (task.Task, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id,project_id,task_type,status,progress,message,result,error,created_at,started_at,ended_at FROM ai_tasks WHERE id=?`, id)
	item, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	return item, err
}
func (r TaskRepository) ListTasks(ctx context.Context, projectID string) ([]task.Task, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,project_id,task_type,status,progress,message,result,error,created_at,started_at,ended_at FROM ai_tasks WHERE (?='' OR project_id=?) ORDER BY created_at DESC`, projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []task.Task{}
	for rows.Next() {
		item, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r TaskRepository) EventsSince(ctx context.Context, taskID string, after uint64) ([]task.Event, error) {
	if _, err := r.GetTask(ctx, taskID); err != nil {
		return nil, err
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT event_id,task_id,event_type,progress,message,created_at FROM task_events WHERE task_id=? AND event_id>? ORDER BY event_id`, taskID, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []task.Event{}
	for rows.Next() {
		var item task.Event
		if err := rows.Scan(&item.ID, &item.TaskID, &item.Type, &item.Progress, &item.Message, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r TaskRepository) RecoverInterrupted(ctx context.Context, now time.Time) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE ai_tasks SET status='FAILED',message='服务重启导致任务中断',error='PROCESS_INTERRUPTED',ended_at=? WHERE status IN ('PENDING','RUNNING')`, now)
	return err
}

type scanner interface{ Scan(...any) error }

func scanTask(row scanner) (task.Task, error) {
	var item task.Task
	var raw []byte
	var started, ended sql.NullTime
	err := row.Scan(&item.ID, &item.ProjectID, &item.Type, &item.Status, &item.Progress, &item.Message, &raw, &item.Error, &item.CreatedAt, &started, &ended)
	if started.Valid {
		item.StartedAt = &started.Time
	}
	if ended.Valid {
		item.EndedAt = &ended.Time
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &item.Result)
	}
	return item, err
}
