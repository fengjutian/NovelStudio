CREATE TABLE IF NOT EXISTS knowledge_files (
 id VARCHAR(32) PRIMARY KEY, project_id VARCHAR(32) NOT NULL, name VARCHAR(255) NOT NULL,
 extension VARCHAR(20) NOT NULL, mime_type VARCHAR(120) NOT NULL, size_bytes BIGINT NOT NULL,
 status VARCHAR(30) NOT NULL, storage_path VARCHAR(700) NOT NULL, source_id VARCHAR(32) NULL,
 created_at DATETIME(6) NOT NULL,
 CONSTRAINT fk_knowledge_files_project FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
 CONSTRAINT fk_knowledge_files_source FOREIGN KEY(source_id) REFERENCES knowledge_sources(id) ON DELETE SET NULL,
 INDEX idx_knowledge_files_project_created(project_id,created_at), INDEX idx_knowledge_files_type(extension)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
