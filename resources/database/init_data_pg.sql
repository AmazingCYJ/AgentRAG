-- AgentRAG PostgreSQL bootstrap data.

BEGIN;

INSERT INTO agentrag_users (id, username, password, role, avatar, create_time, update_time)
VALUES ('u_admin', 'admin', 'admin123', 'admin', '', NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    password = EXCLUDED.password,
    role = EXCLUDED.role,
    avatar = EXCLUDED.avatar,
    update_time = NOW();

COMMIT;
