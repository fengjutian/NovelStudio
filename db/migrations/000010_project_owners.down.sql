ALTER TABLE projects DROP FOREIGN KEY fk_projects_owner;
DROP INDEX idx_projects_owner_updated ON projects;
ALTER TABLE projects DROP COLUMN owner_id;
