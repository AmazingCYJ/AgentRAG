-- AgentRAG PostgreSQL schema.
-- Keep table names and columns aligned with internal/platform/db repositories.

BEGIN;

CREATE TABLE IF NOT EXISTS agentrag_users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role TEXT NOT NULL,
    avatar TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS agentrag_sample_questions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    question TEXT NOT NULL,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_sample_questions_update_time
ON agentrag_sample_questions (update_time DESC);

CREATE TABLE IF NOT EXISTS agentrag_query_mappings (
    id TEXT PRIMARY KEY,
    source_term TEXT NOT NULL,
    target_term TEXT NOT NULL,
    match_type INTEGER NOT NULL,
    priority INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL,
    remark TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_query_mappings_priority
ON agentrag_query_mappings (priority, create_time);

CREATE INDEX IF NOT EXISTS idx_agentrag_query_mappings_source_term
ON agentrag_query_mappings (source_term);

CREATE TABLE IF NOT EXISTS agentrag_intent_nodes (
    id TEXT NOT NULL PRIMARY KEY,
    kb_id TEXT,
    intent_code TEXT NOT NULL,
    name TEXT NOT NULL,
    level INTEGER NOT NULL,
    parent_code TEXT,
    description TEXT,
    examples TEXT,
    collection_name TEXT,
    top_k INTEGER,
    mcp_tool_id TEXT,
    kind INTEGER NOT NULL DEFAULT 0,
    prompt_snippet TEXT,
    prompt_template TEXT,
    param_prompt_template TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT ck_agentrag_intent_nodes_top_k_positive CHECK (top_k IS NULL OR top_k > 0)
);

CREATE INDEX IF NOT EXISTS idx_agentrag_intent_nodes_parent_code
ON agentrag_intent_nodes (parent_code);

CREATE INDEX IF NOT EXISTS idx_agentrag_intent_nodes_intent_code
ON agentrag_intent_nodes (intent_code);

CREATE TABLE IF NOT EXISTS agentrag_conversations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    last_time TIMESTAMP,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    UNIQUE (conversation_id, user_id)
);

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

CREATE TABLE IF NOT EXISTS agentrag_conversation_messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    thinking_content TEXT NOT NULL DEFAULT '',
    thinking_duration INTEGER NULL,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0
);

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

CREATE INDEX IF NOT EXISTS idx_agentrag_message_feedback_conversation_id
ON agentrag_message_feedback (conversation_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_message_feedback_user_id
ON agentrag_message_feedback (user_id);

CREATE TABLE IF NOT EXISTS agentrag_trace_runs (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL UNIQUE,
    trace_name TEXT NOT NULL DEFAULT '',
    entry_method TEXT NOT NULL DEFAULT '',
    conversation_id TEXT NOT NULL DEFAULT '',
    task_id TEXT NOT NULL DEFAULT '',
    user_name TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    extra_data TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_agentrag_trace_runs_task_id
ON agentrag_trace_runs (task_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_trace_runs_user_id
ON agentrag_trace_runs (user_id);

CREATE TABLE IF NOT EXISTS agentrag_trace_nodes (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    parent_node_id TEXT NOT NULL DEFAULT '',
    depth INTEGER NOT NULL DEFAULT 0,
    node_type TEXT NOT NULL DEFAULT '',
    node_name TEXT NOT NULL DEFAULT '',
    class_name TEXT NOT NULL DEFAULT '',
    method_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    extra_data TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL,
    deleted INTEGER NOT NULL DEFAULT 0,
    UNIQUE (trace_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_agentrag_trace_nodes_trace_id
ON agentrag_trace_nodes (trace_id, start_time);

CREATE TABLE IF NOT EXISTS agentrag_knowledge_bases (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    embedding_model TEXT NOT NULL,
    collection_name TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    document_count INTEGER NOT NULL DEFAULT 0,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_bases_name
ON agentrag_knowledge_bases (name);

CREATE TABLE IF NOT EXISTS agentrag_knowledge_documents (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL,
    doc_name TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT '',
    source_location TEXT NOT NULL DEFAULT '',
    text_content TEXT NOT NULL DEFAULT '',
    schedule_enabled INTEGER NOT NULL DEFAULT 0,
    schedule_cron TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    file_url TEXT NOT NULL DEFAULT '',
    file_type TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    process_mode TEXT NOT NULL DEFAULT '',
    chunk_strategy TEXT NOT NULL DEFAULT '',
    chunk_config TEXT NOT NULL DEFAULT '',
    pipeline_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_documents_kb_id
ON agentrag_knowledge_documents (kb_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_documents_status
ON agentrag_knowledge_documents (status);

CREATE TABLE IF NOT EXISTS agentrag_knowledge_chunks (
    id TEXT PRIMARY KEY,
    kb_id TEXT NOT NULL DEFAULT '',
    doc_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL DEFAULT 0,
    content TEXT NOT NULL DEFAULT '',
    content_hash TEXT NOT NULL DEFAULT '',
    char_count INTEGER NOT NULL DEFAULT 0,
    token_count INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_chunks_doc_id
ON agentrag_knowledge_chunks (doc_id, chunk_index);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_chunks_kb_id
ON agentrag_knowledge_chunks (kb_id);

CREATE TABLE IF NOT EXISTS agentrag_knowledge_chunk_logs (
    id TEXT PRIMARY KEY,
    doc_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT '',
    process_mode TEXT NOT NULL DEFAULT '',
    chunk_strategy TEXT NOT NULL DEFAULT '',
    pipeline_id TEXT NOT NULL DEFAULT '',
    pipeline_name TEXT NOT NULL DEFAULT '',
    extract_duration INTEGER NOT NULL DEFAULT 0,
    chunk_duration INTEGER NOT NULL DEFAULT 0,
    embed_duration INTEGER NOT NULL DEFAULT 0,
    persist_duration INTEGER NOT NULL DEFAULT 0,
    other_duration INTEGER NOT NULL DEFAULT 0,
    total_duration INTEGER NOT NULL DEFAULT 0,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    create_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_knowledge_chunk_logs_doc_id
ON agentrag_knowledge_chunk_logs (doc_id, create_time DESC);

CREATE TABLE IF NOT EXISTS agentrag_ingestion_pipelines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS agentrag_ingestion_pipeline_nodes (
    id INTEGER NOT NULL,
    pipeline_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    settings TEXT NOT NULL DEFAULT '',
    condition_json TEXT NOT NULL DEFAULT '',
    next_node_id TEXT NOT NULL DEFAULT '',
    node_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pipeline_id, id)
);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_pipeline_nodes_pipeline_id
ON agentrag_ingestion_pipeline_nodes (pipeline_id, node_order);

CREATE TABLE IF NOT EXISTS agentrag_ingestion_tasks (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT '',
    source_location TEXT NOT NULL DEFAULT '',
    source_file_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    chunk_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    logs TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_tasks_pipeline_id
ON agentrag_ingestion_tasks (pipeline_id);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_tasks_status
ON agentrag_ingestion_tasks (status, create_time DESC);

CREATE TABLE IF NOT EXISTS agentrag_ingestion_task_nodes (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    node_order INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    output TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMP NOT NULL,
    update_time TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_task_nodes_task_id
ON agentrag_ingestion_task_nodes (task_id, node_order);

CREATE INDEX IF NOT EXISTS idx_agentrag_ingestion_task_nodes_pipeline_id
ON agentrag_ingestion_task_nodes (pipeline_id);

COMMIT;
