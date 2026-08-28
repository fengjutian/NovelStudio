DROP TABLE IF EXISTS validation_results;
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS knowledge_sources;
ALTER TABLE documents DROP FOREIGN KEY fk_documents_current_version;
DROP TABLE IF EXISTS document_versions;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS content_nodes;
DROP TABLE IF EXISTS projects;
