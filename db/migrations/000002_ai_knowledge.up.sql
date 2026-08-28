CREATE TABLE IF NOT EXISTS ai_runs (
    id VARCHAR(32) PRIMARY KEY, project_id VARCHAR(32) NULL, task_id VARCHAR(32) NULL,
    role VARCHAR(32) NOT NULL, provider VARCHAR(80) NOT NULL, model VARCHAR(120) NOT NULL,
    prompt_version VARCHAR(40) NOT NULL, request_id VARCHAR(255) NOT NULL DEFAULT '',
    input_tokens INT NOT NULL DEFAULT 0, output_tokens INT NOT NULL DEFAULT 0,
    latency_ms BIGINT NOT NULL DEFAULT 0, status VARCHAR(20) NOT NULL, error TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL, INDEX idx_ai_runs_project_created(project_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS knowledge_facts (
    id VARCHAR(32) PRIMARY KEY, project_id VARCHAR(32) NOT NULL, subject VARCHAR(255) NOT NULL,
    predicate VARCHAR(255) NOT NULL, object TEXT NOT NULL, source_version_id VARCHAR(32) NULL,
    confidence DECIMAL(5,4) NOT NULL, status ENUM('PROPOSED','CONFIRMED','REJECTED','SUPERSEDED') NOT NULL DEFAULT 'PROPOSED',
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    CONSTRAINT fk_facts_project FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
    INDEX idx_facts_project_subject(project_id, subject)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
