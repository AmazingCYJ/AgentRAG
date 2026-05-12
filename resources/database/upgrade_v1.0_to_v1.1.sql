-- AgentRAG PostgreSQL upgrade: v1.0 -> v1.1.
-- Aligns older repository-created tables with the current schema_pg.sql shape.

BEGIN;

ALTER TABLE agentrag_conversations ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_conversations ADD COLUMN IF NOT EXISTS create_time TIMESTAMP;
ALTER TABLE agentrag_conversations ADD COLUMN IF NOT EXISTS update_time TIMESTAMP;
ALTER TABLE agentrag_conversations ADD COLUMN IF NOT EXISTS deleted INTEGER NOT NULL DEFAULT 0;

UPDATE agentrag_conversations
SET id = conversation_id || '_' || user_id
WHERE id IS NULL OR id = '';

UPDATE agentrag_conversations
SET create_time = last_time
WHERE create_time IS NULL;

UPDATE agentrag_conversations
SET update_time = last_time
WHERE update_time IS NULL;

DO $$
DECLARE
    pk_columns TEXT[];
BEGIN
    SELECT array_agg(a.attname ORDER BY a.attnum)
    INTO pk_columns
    FROM pg_index i
    JOIN pg_attribute a
      ON a.attrelid = i.indrelid
     AND a.attnum = ANY(i.indkey)
    WHERE i.indrelid = 'agentrag_conversations'::regclass
      AND i.indisprimary;

    IF pk_columns = ARRAY['conversation_id'] THEN
        ALTER TABLE agentrag_conversations DROP CONSTRAINT agentrag_conversations_pkey;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'agentrag_conversations'::regclass
          AND contype = 'p'
    ) THEN
        ALTER TABLE agentrag_conversations ADD CONSTRAINT agentrag_conversations_pkey PRIMARY KEY (id);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_agentrag_conversations_conversation_user
ON agentrag_conversations (conversation_id, user_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_conversations_user_time
ON agentrag_conversations (user_id, last_time);

CREATE TABLE IF NOT EXISTS agentrag_conversation_summaries (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    last_message_id TEXT NOT NULL,
    content TEXT NOT NULL,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_agentrag_conversation_summaries_conv_user
ON agentrag_conversation_summaries (conversation_id, user_id);

ALTER TABLE agentrag_conversation_messages ADD COLUMN IF NOT EXISTS update_time TIMESTAMP;
ALTER TABLE agentrag_conversation_messages ADD COLUMN IF NOT EXISTS deleted INTEGER NOT NULL DEFAULT 0;
ALTER TABLE agentrag_conversation_messages ADD COLUMN IF NOT EXISTS thinking_content TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_conversation_messages ADD COLUMN IF NOT EXISTS thinking_duration INTEGER NULL;

UPDATE agentrag_conversation_messages
SET update_time = create_time
WHERE update_time IS NULL;

CREATE INDEX IF NOT EXISTS idx_agentrag_conversation_messages_conv_user_time
ON agentrag_conversation_messages (conversation_id, user_id, create_time);

CREATE TABLE IF NOT EXISTS agentrag_message_feedback (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    vote INTEGER NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    UNIQUE (message_id, user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_agentrag_message_feedback_message_user
ON agentrag_message_feedback (message_id, user_id);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'agentrag_conversation_messages'
          AND column_name = 'vote'
    ) THEN
        INSERT INTO agentrag_message_feedback
            (id, message_id, conversation_id, user_id, vote, reason, comment, create_time, update_time, deleted)
        SELECT id || '_' || user_id, id, conversation_id, user_id, vote, '', '', create_time, create_time, 0
        FROM agentrag_conversation_messages
        WHERE vote IS NOT NULL
        ON CONFLICT (message_id, user_id) DO NOTHING;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_agentrag_message_feedback_conversation_id
ON agentrag_message_feedback (conversation_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_message_feedback_user_id
ON agentrag_message_feedback (user_id);

ALTER TABLE agentrag_trace_runs ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_trace_runs ADD COLUMN IF NOT EXISTS extra_data TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_trace_runs ADD COLUMN IF NOT EXISTS create_time TIMESTAMP;
ALTER TABLE agentrag_trace_runs ADD COLUMN IF NOT EXISTS update_time TIMESTAMP;
ALTER TABLE agentrag_trace_runs ADD COLUMN IF NOT EXISTS deleted INTEGER NOT NULL DEFAULT 0;

UPDATE agentrag_trace_runs
SET id = 'run_' || trace_id
WHERE id IS NULL OR id = '';

UPDATE agentrag_trace_runs
SET create_time = start_time
WHERE create_time IS NULL;

UPDATE agentrag_trace_runs
SET update_time = end_time
WHERE update_time IS NULL;

DO $$
DECLARE
    pk_columns TEXT[];
BEGIN
    SELECT array_agg(a.attname ORDER BY a.attnum)
    INTO pk_columns
    FROM pg_index i
    JOIN pg_attribute a
      ON a.attrelid = i.indrelid
     AND a.attnum = ANY(i.indkey)
    WHERE i.indrelid = 'agentrag_trace_runs'::regclass
      AND i.indisprimary;

    IF pk_columns = ARRAY['trace_id'] THEN
        ALTER TABLE agentrag_trace_runs DROP CONSTRAINT agentrag_trace_runs_pkey;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'agentrag_trace_runs'::regclass
          AND contype = 'p'
    ) THEN
        ALTER TABLE agentrag_trace_runs ADD CONSTRAINT agentrag_trace_runs_pkey PRIMARY KEY (id);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_agentrag_trace_runs_trace_id
ON agentrag_trace_runs (trace_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_trace_runs_task_id
ON agentrag_trace_runs (task_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_trace_runs_user_id
ON agentrag_trace_runs (user_id);

ALTER TABLE agentrag_trace_nodes ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_trace_nodes ADD COLUMN IF NOT EXISTS extra_data TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_trace_nodes ADD COLUMN IF NOT EXISTS create_time TIMESTAMP;
ALTER TABLE agentrag_trace_nodes ADD COLUMN IF NOT EXISTS update_time TIMESTAMP;
ALTER TABLE agentrag_trace_nodes ADD COLUMN IF NOT EXISTS deleted INTEGER NOT NULL DEFAULT 0;

UPDATE agentrag_trace_nodes
SET id = trace_id || '_' || node_id
WHERE id IS NULL OR id = '';

UPDATE agentrag_trace_nodes
SET create_time = start_time
WHERE create_time IS NULL;

UPDATE agentrag_trace_nodes
SET update_time = end_time
WHERE update_time IS NULL;

DO $$
DECLARE
    pk_columns TEXT[];
BEGIN
    SELECT array_agg(a.attname ORDER BY a.attnum)
    INTO pk_columns
    FROM pg_index i
    JOIN pg_attribute a
      ON a.attrelid = i.indrelid
     AND a.attnum = ANY(i.indkey)
    WHERE i.indrelid = 'agentrag_trace_nodes'::regclass
      AND i.indisprimary;

    IF pk_columns = ARRAY['trace_id', 'node_id'] THEN
        ALTER TABLE agentrag_trace_nodes DROP CONSTRAINT agentrag_trace_nodes_pkey;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'agentrag_trace_nodes'::regclass
          AND contype = 'p'
    ) THEN
        ALTER TABLE agentrag_trace_nodes ADD CONSTRAINT agentrag_trace_nodes_pkey PRIMARY KEY (id);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_agentrag_trace_nodes_trace_node
ON agentrag_trace_nodes (trace_id, node_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_trace_nodes_trace_id
ON agentrag_trace_nodes (trace_id, start_time);

ALTER TABLE agentrag_knowledge_documents ADD COLUMN IF NOT EXISTS text_content TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_knowledge_chunks ADD COLUMN IF NOT EXISTS embedding_model TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_knowledge_chunks ADD COLUMN IF NOT EXISTS embedding_vector TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS agentrag_knowledge_vectors (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL DEFAULT '',
    doc_id TEXT NOT NULL DEFAULT '',
    collection_name TEXT NOT NULL DEFAULT '',
    chunk_id TEXT NOT NULL DEFAULT '',
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    embedding_model TEXT NOT NULL DEFAULT '',
    embedding_vector TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'agentrag_knowledge_chunk_logs'
          AND column_name = 'embedding_duration'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'agentrag_knowledge_chunk_logs'
          AND column_name = 'embed_duration'
    ) THEN
        ALTER TABLE agentrag_knowledge_chunk_logs RENAME COLUMN embedding_duration TO embed_duration;
    END IF;
END $$;

ALTER TABLE agentrag_knowledge_chunk_logs ADD COLUMN IF NOT EXISTS pipeline_name TEXT NOT NULL DEFAULT '';
ALTER TABLE agentrag_knowledge_chunk_logs ADD COLUMN IF NOT EXISTS embed_duration INTEGER NOT NULL DEFAULT 0;
ALTER TABLE agentrag_knowledge_chunk_logs ADD COLUMN IF NOT EXISTS persist_duration INTEGER NOT NULL DEFAULT 0;
ALTER TABLE agentrag_knowledge_chunk_logs ADD COLUMN IF NOT EXISTS other_duration INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_agentrag_sample_questions_update_time
ON agentrag_sample_questions (update_time DESC);

CREATE INDEX IF NOT EXISTS idx_agentrag_query_mappings_priority
ON agentrag_query_mappings (priority, create_time);

CREATE INDEX IF NOT EXISTS idx_agentrag_query_mappings_source_term
ON agentrag_query_mappings (source_term);

CREATE INDEX IF NOT EXISTS idx_agentrag_intent_nodes_parent_code
ON agentrag_intent_nodes (parent_code);

CREATE INDEX IF NOT EXISTS idx_agentrag_intent_nodes_intent_code
ON agentrag_intent_nodes (intent_code);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_bases_name
ON agentrag_knowledge_bases (name);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_documents_kb_id
ON agentrag_knowledge_documents (kb_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_documents_status
ON agentrag_knowledge_documents (status);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_chunks_doc_id
ON agentrag_knowledge_chunks (doc_id, chunk_index);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_chunks_kb_id
ON agentrag_knowledge_chunks (kb_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_vectors_collection
ON agentrag_knowledge_vectors (collection_name, embedding_model);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_vectors_chunk
ON agentrag_knowledge_vectors (chunk_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_chunk_logs_doc_id
ON agentrag_knowledge_chunk_logs (doc_id, create_time DESC);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_pipeline_nodes_pipeline_id
ON agentrag_ingestion_pipeline_nodes (pipeline_id, node_order);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_tasks_pipeline_id
ON agentrag_ingestion_tasks (pipeline_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_tasks_status
ON agentrag_ingestion_tasks (status, create_time DESC);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_task_nodes_task_id
ON agentrag_ingestion_task_nodes (task_id, node_order);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_task_nodes_pipeline_id
ON agentrag_ingestion_task_nodes (pipeline_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ck_agentrag_intent_nodes_top_k_positive'
          AND conrelid = 'agentrag_intent_nodes'::regclass
    ) THEN
        ALTER TABLE agentrag_intent_nodes
            ADD CONSTRAINT ck_agentrag_intent_nodes_top_k_positive
            CHECK (top_k IS NULL OR top_k > 0) NOT VALID;
    END IF;
END $$;

COMMIT;
