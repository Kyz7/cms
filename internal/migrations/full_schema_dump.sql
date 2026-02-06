-- ==========================================
-- ENUMS
-- ==========================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'workflow_status') THEN
        CREATE TYPE workflow_status AS ENUM (
            'draft',
            'in_review',
            'ready_for_approval',
            'approved',
            'published',
            'rejected'
        );
    END IF;
END
$$;

-- ==========================================
-- ROLES & USERS
-- ==========================================

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE,
    description TEXT,
    is_global BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_roles_deleted_at ON roles(deleted_at);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(100) UNIQUE,
    password VARCHAR(255),
    provider VARCHAR(50),
    status VARCHAR(20) DEFAULT 'active',
    role_id INTEGER REFERENCES roles(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    profile TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- ==========================================
-- PERMISSIONS
-- ==========================================

CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    module VARCHAR(50),
    action VARCHAR(50),
    field_scope VARCHAR(50),      -- "all", "seo_only", "custom" etc
    allowed_fields JSONB,         -- ["title", "slug"]
    denied_fields JSONB,          -- ["internal_notes"]
    content_type_ids JSONB,       -- [1, 2, 3]
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_permissions_deleted_at ON permissions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_role_module_action ON permissions(role_id, module, action);

-- ==========================================
-- PROJECTS
-- ==========================================

CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);

CREATE TABLE IF NOT EXISTS project_members (
    id SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES roles(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    invited_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_deleted_at ON project_members(deleted_at);

-- ==========================================
-- CONTENT SCHEMA (Types & Fields)
-- ==========================================

CREATE TABLE IF NOT EXISTS content_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    slug VARCHAR(100),
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE, -- nullable for global
    enable_seo BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_content_types_project_id ON content_types(project_id);
CREATE INDEX IF NOT EXISTS idx_content_types_deleted_at ON content_types(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_content_types_slug_project ON content_types(slug, project_id); 

CREATE TABLE IF NOT EXISTS content_fields (
    id SERIAL PRIMARY KEY,
    content_type_id INTEGER REFERENCES content_types(id) ON DELETE CASCADE,
    name VARCHAR(100),
    type VARCHAR(50),
    required BOOLEAN DEFAULT false,
    is_seo BOOLEAN DEFAULT false,
    "unique" BOOLEAN DEFAULT false, -- unique is a keyword, quoting it
    max_length INTEGER,
    min_length INTEGER,
    pattern VARCHAR(255),
    min_value DOUBLE PRECISION,
    max_value DOUBLE PRECISION,
    default_value VARCHAR(500),
    placeholder VARCHAR(255),
    help_text VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_content_fields_deleted_at ON content_fields(deleted_at);

-- ==========================================
-- CONTENT ENTRIES
-- ==========================================

CREATE TABLE IF NOT EXISTS content_entries (
    id SERIAL PRIMARY KEY,
    content_type_id INTEGER REFERENCES content_types(id) ON DELETE CASCADE,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    data JSONB,
    status workflow_status DEFAULT 'draft',
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_content_entries_project_id ON content_entries(project_id);
CREATE INDEX IF NOT EXISTS idx_content_entries_status ON content_entries(status);
CREATE INDEX IF NOT EXISTS idx_content_entries_created_by ON content_entries(created_by);
CREATE INDEX IF NOT EXISTS idx_content_entries_deleted_at ON content_entries(deleted_at);
-- GIN Indexes for JSON data and Search
CREATE INDEX IF NOT EXISTS idx_content_entries_data_gin ON content_entries USING GIN (data);

CREATE TABLE IF NOT EXISTS content_relations (
    id SERIAL PRIMARY KEY,
    from_content_id INTEGER NOT NULL, -- Logical FK to content_entries
    to_content_id INTEGER NOT NULL,   -- Logical FK to content_entries
    relation_type VARCHAR(50),
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_content_relations_project_id ON content_relations(project_id);
CREATE INDEX IF NOT EXISTS idx_content_relations_deleted_at ON content_relations(deleted_at);

-- ==========================================
-- WORKFLOW
-- ==========================================

CREATE TABLE IF NOT EXISTS workflow_transitions (
    id SERIAL PRIMARY KEY,
    from_status workflow_status,
    to_status workflow_status,
    required_role VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_workflow_transitions_deleted_at ON workflow_transitions(deleted_at);

CREATE TABLE IF NOT EXISTS workflow_histories (
    id SERIAL PRIMARY KEY,
    entry_id INTEGER REFERENCES content_entries(id) ON DELETE CASCADE,
    from_status workflow_status,
    to_status workflow_status,
    changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_workflow_histories_deleted_at ON workflow_histories(deleted_at);

CREATE TABLE IF NOT EXISTS workflow_comments (
    id SERIAL PRIMARY KEY,
    entry_id INTEGER REFERENCES content_entries(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    comment TEXT,
    is_private BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_workflow_comments_deleted_at ON workflow_comments(deleted_at);

CREATE TABLE IF NOT EXISTS workflow_assignments (
    id SERIAL PRIMARY KEY,
    entry_id INTEGER REFERENCES content_entries(id) ON DELETE CASCADE,
    assigned_to INTEGER REFERENCES users(id) ON DELETE CASCADE,
    assigned_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'pending',
    due_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_workflow_assignments_deleted_at ON workflow_assignments(deleted_at);

-- ==========================================
-- MEDIA
-- ==========================================

CREATE TABLE IF NOT EXISTS media_folders (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    path VARCHAR(255) UNIQUE,
    parent_id INTEGER REFERENCES media_folders(id) ON DELETE SET NULL,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_media_folders_project_id ON media_folders(project_id);
CREATE INDEX IF NOT EXISTS idx_media_folders_deleted_at ON media_folders(deleted_at);

CREATE TABLE IF NOT EXISTS media_files (
    id SERIAL PRIMARY KEY,
    file_name VARCHAR(255),
    url VARCHAR(500),
    type VARCHAR(100),
    size BIGINT,
    width INTEGER,
    height INTEGER,
    folder VARCHAR(255),
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    tags JSONB,
    alt VARCHAR(255),
    caption TEXT,
    uploaded_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_media_files_folder ON media_files(folder);
CREATE INDEX IF NOT EXISTS idx_media_files_type ON media_files(type);
CREATE INDEX IF NOT EXISTS idx_media_files_project_id ON media_files(project_id);
CREATE INDEX IF NOT EXISTS idx_media_files_uploaded_by ON media_files(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_media_files_deleted_at ON media_files(deleted_at);

-- ==========================================
-- AUTHENTICATION & TOKENS
-- ==========================================

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS reset_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_reset_tokens_token_hash ON reset_tokens(token_hash);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at);

