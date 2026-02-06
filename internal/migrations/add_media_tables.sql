-- Create media_folders table
CREATE TABLE IF NOT EXISTS media_folders (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    path VARCHAR(255),
    parent_id INTEGER REFERENCES media_folders(id) ON DELETE SET NULL,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_media_folders_path ON media_folders(path);
CREATE INDEX IF NOT EXISTS idx_media_folders_project_id ON media_folders(project_id);
CREATE INDEX IF NOT EXISTS idx_media_folders_deleted_at ON media_folders(deleted_at);

-- Create media_files table
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

CREATE INDEX IF NOT EXISTS idx_media_files_type ON media_files(type);
CREATE INDEX IF NOT EXISTS idx_media_files_folder ON media_files(folder);
CREATE INDEX IF NOT EXISTS idx_media_files_project_id ON media_files(project_id);
CREATE INDEX IF NOT EXISTS idx_media_files_uploaded_by ON media_files(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_media_files_deleted_at ON media_files(deleted_at);
