-- Add projects table
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Add project_members table
CREATE TABLE IF NOT EXISTS project_members (
    id SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    invited_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    UNIQUE(project_id, user_id)
);

-- Add indexes for project_members
CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);

-- Add project_id to content_types (nullable for global content types)
ALTER TABLE content_types ADD COLUMN IF NOT EXISTS project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE;

-- Add index for content_types project_id
CREATE INDEX IF NOT EXISTS idx_content_types_project_id ON content_types(project_id);

-- Add project_id to content_entries (nullable for global entries)
ALTER TABLE content_entries ADD COLUMN IF NOT EXISTS project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE;

-- Add index for content_entries project_id
CREATE INDEX IF NOT EXISTS idx_content_entries_project_id ON content_entries(project_id);

-- Update unique constraints for content_types to allow same slug in different projects
-- First, drop the existing unique index if it exists
DROP INDEX IF EXISTS content_types_slug_key;
DROP INDEX IF EXISTS content_types_name_key;

-- Create new unique index that includes project_id (NULL values are considered distinct)
CREATE UNIQUE INDEX IF NOT EXISTS idx_content_types_slug_project ON content_types(slug, project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_content_types_name_project ON content_types(name, project_id);

