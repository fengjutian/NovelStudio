CREATE TABLE IF NOT EXISTS story_memories (
    id VARCHAR(32) PRIMARY KEY, project_id VARCHAR(32) NOT NULL, memory_type VARCHAR(30) NOT NULL,
    name VARCHAR(200) NOT NULL, summary TEXT NOT NULL, status VARCHAR(30) NOT NULL,
    attributes JSON NULL, created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    CONSTRAINT fk_story_memories_project FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
    INDEX idx_story_memories_project_type(project_id,memory_type), INDEX idx_story_memories_updated(updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
