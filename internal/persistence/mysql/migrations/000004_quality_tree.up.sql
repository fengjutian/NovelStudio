ALTER TABLE validation_results ADD COLUMN document_id VARCHAR(32) NULL AFTER project_id;
ALTER TABLE validation_results ADD COLUMN version_id VARCHAR(32) NULL AFTER document_id;
ALTER TABLE content_nodes ADD COLUMN document_id VARCHAR(32) NULL AFTER parent_id;
ALTER TABLE content_nodes ADD CONSTRAINT fk_content_nodes_document FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE SET NULL;
