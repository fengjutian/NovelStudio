DROP TABLE IF EXISTS stale_chapter_history;
ALTER TABLE stale_chapters DROP COLUMN resolution_note, DROP COLUMN resolved_by, DROP COLUMN resolved_at, DROP COLUMN status, DROP COLUMN paths_json;
