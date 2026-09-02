ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'recruiter', 'hiring_manager', 'viewer'));

ALTER TABLE candidates DROP COLUMN IF EXISTS assigned_to;

DROP TABLE IF EXISTS collaboration_comments;
DROP TABLE IF EXISTS interviews;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS company_ai_settings;
