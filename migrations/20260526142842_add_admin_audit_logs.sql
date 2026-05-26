-- +goose Up
CREATE TABLE admin_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID REFERENCES admins(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    message TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_logs_admin_id_idx ON admin_audit_logs (admin_id);
CREATE INDEX admin_audit_logs_action_idx ON admin_audit_logs (action);
CREATE INDEX admin_audit_logs_entity_idx ON admin_audit_logs (entity_type, entity_id);
CREATE INDEX admin_audit_logs_created_at_idx ON admin_audit_logs (created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS admin_audit_logs_created_at_idx;
DROP INDEX IF EXISTS admin_audit_logs_entity_idx;
DROP INDEX IF EXISTS admin_audit_logs_action_idx;
DROP INDEX IF EXISTS admin_audit_logs_admin_id_idx;

DROP TABLE IF EXISTS admin_audit_logs;