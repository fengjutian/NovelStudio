ALTER TABLE content_nodes DROP FOREIGN KEY fk_content_nodes_document;
ALTER TABLE content_nodes DROP COLUMN document_id;
ALTER TABLE validation_results DROP COLUMN version_id;
ALTER TABLE validation_results DROP COLUMN document_id;
