-- seed.sql — deterministic fixture for internal/engram/ tests.
-- Seeded at test setup time via INSERT statements into a t.TempDir() SQLite DB.
-- Matches actual engram.db observations schema (discovered via PRAGMA table_info):
-- id, sync_id, session_id, type, title, content, tool_name, project, scope,
-- topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at,
-- created_at, updated_at, deleted_at

CREATE TABLE IF NOT EXISTS observations (
    id              INTEGER PRIMARY KEY,
    sync_id         TEXT,
    session_id      TEXT    NOT NULL DEFAULT '',
    type            TEXT    NOT NULL DEFAULT '',
    title           TEXT    NOT NULL DEFAULT '',
    content         TEXT    NOT NULL DEFAULT '',
    tool_name       TEXT,
    project         TEXT,
    scope           TEXT    NOT NULL DEFAULT 'project',
    topic_key       TEXT,
    normalized_hash TEXT,
    revision_count  INTEGER NOT NULL DEFAULT 1,
    duplicate_count INTEGER NOT NULL DEFAULT 1,
    last_seen_at    TEXT,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    deleted_at      TEXT
);

-- 3 observations for project="atelier" (NOT deleted), sorted newest first by created_at
INSERT INTO observations (id, session_id, type, title, content, project, scope, topic_key, created_at, updated_at, deleted_at)
VALUES
(1, 'sess-1', 'architecture', 'Bootstrap decision',
 'First line of content for obs 1'||char(10)||'Second line here',
 'atelier', 'project', 'sdd/atelier-bootstrap/design',
 '2026-05-01 10:00:00', '2026-05-01 10:00:00', NULL),

(2, 'sess-2', 'bugfix', 'Fixed N+1 query',
 'What: Fixed the N+1 issue in the registry'||char(10)||'Why: Performance',
 'atelier', 'project', 'atelier/bugfix/n-plus-one',
 '2026-05-02 11:00:00', '2026-05-02 11:00:00', NULL),

(3, 'sess-3', 'decision', 'Use Bubble Tea',
 '',
 'atelier', 'project', NULL,
 '2026-05-03 12:00:00', '2026-05-03 12:00:00', NULL);

-- 2 observations for project="other" (NOT deleted)
INSERT INTO observations (id, session_id, type, title, content, project, scope, topic_key, created_at, updated_at, deleted_at)
VALUES
(4, 'sess-4', 'pattern', 'Other project entry 1',
 'Content for other project',
 'other', 'project', NULL,
 '2026-05-01 09:00:00', '2026-05-01 09:00:00', NULL),

(5, 'sess-5', 'decision', 'Other project entry 2',
 'More content for other',
 'other', 'project', NULL,
 '2026-05-02 09:00:00', '2026-05-02 09:00:00', NULL);

-- 1 soft-deleted observation for project="atelier"
INSERT INTO observations (id, session_id, type, title, content, project, scope, topic_key, created_at, updated_at, deleted_at)
VALUES
(6, 'sess-6', 'architecture', 'Deleted observation',
 'This entry is soft-deleted and must not be returned by ListByProject',
 'atelier', 'project', NULL,
 '2026-05-01 08:00:00', '2026-05-01 08:00:00', '2026-05-01 09:00:00');

-- 2 SDD archive-report observations for project="atelier"
-- topic_key LIKE 'sdd/%' AND topic_key LIKE '%/archive-report'
INSERT INTO observations (id, session_id, type, title, content, project, scope, topic_key, created_at, updated_at, deleted_at)
VALUES
(7, 'sess-7', 'architecture', 'Bootstrap archive',
 'Archive report content for bootstrap change',
 'atelier', 'project', 'sdd/atelier-bootstrap/archive-report',
 '2026-04-28 10:00:00', '2026-04-28 10:00:00', NULL),

(8, 'sess-8', 'architecture', 'Registry archive',
 'Archive report content for project-registry change',
 'atelier', 'project', 'sdd/atelier-project-registry/archive-report',
 '2026-04-30 10:00:00', '2026-04-30 10:00:00', NULL);

-- 5000-character content observation for GetByID full-content test
INSERT INTO observations (id, session_id, type, title, content, project, scope, topic_key, created_at, updated_at, deleted_at)
VALUES
(9, 'sess-9', 'decision', 'Long content observation',
 REPLACE(printf('%5000c', 'x'), ' ', 'x'),
 'atelier', 'project', 'atelier/decision/long',
 '2026-05-04 10:00:00', '2026-05-04 10:00:00', NULL);
