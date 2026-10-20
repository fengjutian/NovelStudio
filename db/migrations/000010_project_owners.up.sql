ALTER TABLE projects ADD COLUMN owner_id VARCHAR(40) NULL AFTER id;
UPDATE projects SET owner_id=(SELECT id FROM users ORDER BY created_at LIMIT 1) WHERE owner_id IS NULL;
ALTER TABLE projects ADD CONSTRAINT fk_projects_owner FOREIGN KEY(owner_id) REFERENCES users(id) ON DELETE RESTRICT;
CREATE INDEX idx_projects_owner_updated ON projects(owner_id, updated_at);
