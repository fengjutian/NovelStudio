package mysql

import (
	"context"
	"database/sql"

	"novelstudio/internal/airun"
)

type AIRunRecorder struct{ DB *sql.DB }

var _ airun.Recorder = AIRunRecorder{}

func (r AIRunRecorder) Record(ctx context.Context, run airun.Run) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO ai_runs(id,project_id,task_id,role,provider,model,prompt_version,request_id,input_tokens,output_tokens,latency_ms,status,error,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, run.ID, nullable(run.ProjectID), nullable(run.TaskID), run.Role, run.Provider, run.Model, run.PromptVersion, run.RequestID, run.InputTokens, run.OutputTokens, run.LatencyMs, run.Status, run.Error, run.CreatedAt)
	return err
}
func (r AIRunRecorder) List(ctx context.Context, projectID string, limit int) ([]airun.Run, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT id,COALESCE(project_id,''),COALESCE(task_id,''),role,provider,model,prompt_version,request_id,input_tokens,output_tokens,latency_ms,status,error,created_at FROM ai_runs WHERE (?='' OR project_id=?) ORDER BY created_at DESC LIMIT ?`, projectID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]airun.Run, 0)
	for rows.Next() {
		var item airun.Run
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.TaskID, &item.Role, &item.Provider, &item.Model, &item.PromptVersion, &item.RequestID, &item.InputTokens, &item.OutputTokens, &item.LatencyMs, &item.Status, &item.Error, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
