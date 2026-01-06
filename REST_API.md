### REST API Documentation

This document describes the REST endpoints available in the CMS. All responses follow a common envelope unless noted:

- success: boolean
- message: string
- data: any
- error: { code, message, details? }
- meta: { page, limit, total, total_pages } when paginated

Base URL: `http://localhost:8080`

Authentication
- Use `Authorization: Bearer <access_token>` unless endpoint is explicitly public.
- Rate limiting applies to some `/auth` endpoints.

---

### Health
- GET `/health`
  - Public
  - 200: `{ status: "ok", message: "CMS API is running" }`

- GET `/csrf-token`
  - Public
  - 200: `{ csrf_token: string }`

---

### Documentation
- GET `/swagger/*` - Swagger UI (serves OpenAPI documentation)
- GET `/openapi.yaml` - OpenAPI specification file

---

### GraphQL
GraphQL API uses standard GraphQL protocol. Most operations require JWT authentication via `Authorization: Bearer <token>` header. Exceptions: `login` and `register` mutations are public.

Endpoints:
- POST `/graphql`
  - Body (json): `{ query: string, variables?: object, operationName?: string }`
  - Headers: `Authorization: Bearer <token>` (optional, required for most operations)
  - 200: `{ data?: any, errors?: array }`

- GET `/graphql`
  - GraphiQL playground (interactive GraphQL IDE)
  - Development only, provides visual query builder

- POST `/graphql/batch`
  - Body (json): Array of `{ query, variables?, operationName? }` (max 10 requests)
  - Headers: `Authorization: Bearer <token>` (optional)
  - 200: Array of GraphQL responses

#### Queries

Auth & User:
- `me: User` - Get current authenticated user (requires auth)
- `users(limit?: Int, offset?: Int): [User]` - List users (admin only)
- `user(id: ID!): User` - Get user by ID (admin only)

Roles:
- `roles: [Role]` - List all roles (admin only)
- `role(id: ID!): Role` - Get role by ID (admin only)

Content:
- `contentTypes: [ContentType]` - List all content types
- `contentType(id: ID!): ContentType` - Get content type by ID
- `contentEntries(contentTypeId: Int!, limit?: Int, offset?: Int): [ContentEntry]` - List entries by content type
- `contentEntry(id: ID!): ContentEntry` - Get entry by ID
- `contentRelations(fromContentId: Int!): [ContentRelation]` - Get relations for entry

Media:
- `mediaFiles(limit?: Int, offset?: Int, folder?: String): [MediaFile]` - List media files
- `mediaFile(id: ID!): MediaFile` - Get media file by ID
- `mediaFolders: [MediaFolder]` - List media folders
- `mediaStats: MediaStats` - Get media statistics

Search:
- `searchEntries(query?: String, contentTypeIds?: [Int], fields?: [String], status?: String, createdBy?: Int, tags?: [String], fromDate?: String, toDate?: String, page?: Int, limit?: Int, sortBy?: String, orderBy?: String): SearchResult` - Full-text search
- `advancedSearch(query?: String, contentTypeIds?: [Int], fields?: [String], status?: String, createdBy?: Int, tags?: [String], fromDate?: String, toDate?: String, filters?: JSON, page?: Int, limit?: Int, sortBy?: String, orderBy?: String): SearchResult` - Advanced search with filters
- `searchFacets(query?: String, contentTypeIds?: [Int]): JSON` - Get search facets
- `autocomplete(field: String!, prefix: String!, contentTypeId: Int!, limit?: Int): [String]` - Autocomplete suggestions

Workflow:
- `workflowHistory(entryId: Int!): [WorkflowHistory]` - Get workflow history for entry
- `workflowComments(entryId: Int!, includePrivate?: Boolean): [WorkflowComment]` - Get comments for entry
- `workflowAssignments(status?: String): [WorkflowAssignment]` - Get workflow assignments
- `workflowStats(contentTypeId: Int!): WorkflowStats` - Get workflow statistics

SEO:
- `seoPreview(entryId: Int!): JSON` - Get SEO preview data

#### Mutations

Auth (Public):
- `login(email: String!, password: String!): AuthPayload` - Login user
- `register(name: String!, email: String!, password: String!): AuthPayload` - Register new user

Users (Admin only):
- `createUser(name: String!, email: String!, password: String!, roleId: Int!): User`
- `updateUser(id: ID!, name?: String, email?: String, password?: String, roleId?: Int, status?: UserStatus): User`
- `deleteUser(id: ID!): Boolean`

Roles (Admin only):
- `createRole(name: String!, description?: String): Role`
- `updateRole(id: ID!, name?: String, description?: String): Role`
- `deleteRole(id: ID!): Boolean`
- `duplicateRole(id: ID!, name?: String): Role`
- `assignRoleToUser(userId: ID!, roleId: ID!): Boolean`

Content Types:
- `createContentType(name: String!, slug: String!, enableSeo?: Boolean): ContentType`
- `updateContentType(id: ID!, name?: String, slug?: String, enableSeo?: Boolean): ContentType`
- `deleteContentType(id: ID!): Boolean`

Content Fields:
- `addContentField(contentTypeId: Int!, name: String!, type: String!, required?: Boolean, isSeo?: Boolean, unique?: Boolean, maxLength?: Int, minLength?: Int, pattern?: String, minValue?: Float, maxValue?: Float, defaultValue?: String, placeholder?: String, helpText?: String): ContentField`
- `updateContentField(id: ID!, name?: String, type?: String, required?: Boolean, isSeo?: Boolean, unique?: Boolean, maxLength?: Int, minLength?: Int, pattern?: String, minValue?: Float, maxValue?: Float, defaultValue?: String, placeholder?: String, helpText?: String): ContentField`
- `deleteContentField(id: ID!): Boolean`

Content Entries:
- `createContentEntry(contentTypeId: ID!, data: JSON!): ContentEntry`
- `updateContentEntry(id: ID!, data?: JSON, status?: WorkflowStatus): ContentEntry`
- `deleteContentEntry(id: ID!): Boolean`
- `translateContentEntry(id: ID!, targetLang: String!, sourceLang?: String, fields?: [String]): ContentEntry`

Content Relations:
- `createContentRelation(fromContentId: ID!, toContentId: Int!, relationType: String!): ContentRelation`
- `deleteContentRelation(id: ID!): Boolean`

Media:
- `createMediaFolder(name: String!, parentId?: Int): MediaFolder`

Workflow:
- `changeContentStatus(entryId: Int!, status: WorkflowStatus!, comment?: String): WorkflowHistory`
- `requestReview(entryId: Int!, comment?: String): WorkflowHistory`
- `approveEntry(entryId: Int!, comment?: String): WorkflowHistory`
- `rejectEntry(entryId: Int!, comment: String!): WorkflowHistory`
- `publishEntry(entryId: Int!, comment?: String): WorkflowHistory`
- `addWorkflowComment(entryId: Int!, comment: String!, isPrivate?: Boolean): WorkflowComment`
- `assignEntry(entryId: Int!, assignedTo: Int!, dueDate?: Time): WorkflowAssignment`

#### Types

- `User`: id, name, email, provider, status, roleId, role, profile, createdAt, updatedAt
- `Role`: id, name, description, permissions, createdAt, updatedAt
- `Permission`: id, roleId, module, action, fieldScope, allowedFields, deniedFields, contentTypeIds, createdAt, updatedAt
- `ContentType`: id, name, slug, enableSeo, fields, seoFields, createdAt, updatedAt
- `ContentField`: id, contentTypeId, name, type, required, isSeo, unique, maxLength, minLength, pattern, minValue, maxValue, defaultValue, placeholder, helpText, createdAt, updatedAt
- `ContentEntry`: id, contentTypeId, contentType, data (JSON), status, createdBy, updatedBy, creator, updater, createdAt, updatedAt, publishedAt
- `ContentRelation`: id, fromContentId, toContentId, relationType, createdAt, updatedAt
- `MediaFile`: id, fileName, url, type, size, width, height, folder, tags, alt, caption, uploadedBy, uploader, createdAt, updatedAt
- `MediaFolder`: id, name, path, parentId, createdBy, createdAt, updatedAt
- `MediaStats`: totalFiles, totalSize, byType, recentUploads, storageMode
- `WorkflowHistory`: id, entryId, entry, fromStatus, toStatus, changedBy, user, comment, createdAt, updatedAt
- `WorkflowComment`: id, entryId, entry, userId, user, comment, isPrivate, createdAt, updatedAt
- `WorkflowAssignment`: id, entryId, entry, assignedTo, user, assignedBy, assigner, status, dueDate, createdAt, updatedAt
- `WorkflowStats`: total, draft, in_review, ready_for_approval, approved, published, rejected
- `SearchResult`: entries, total, page, limit, totalPages, hasNextPage, hasPreviousPage, query
- `AuthPayload`: token, refreshToken, user

#### Enums

- `WorkflowStatus`: DRAFT, IN_REVIEW, READY_FOR_APPROVAL, APPROVED, PUBLISHED, REJECTED
- `UserStatus`: ACTIVE, INACTIVE, SUSPENDED

#### Scalar Types

- `Time`: RFC3339 formatted timestamp
- `JSON`: Arbitrary JSON object

---

### Auth
- POST `/auth/register`
  - Public
  - Body (json): `{ name, email, password }`
  - 201: `{ access_token, refresh_token, user }`

- POST `/auth/login`
  - Public (rate limited)
  - Body (json): `{ email, password }`
  - 200: `{ access_token, refresh_token, expires_in }`

- POST `/auth/refresh`
  - Public (rate limited)
  - Body (json): `{ user_id, refresh_token }`
  - 200: `{ access_token, refresh_token, user, expires_in }`

- POST `/auth/logout`
  - Requires JWT
  - 200: `{ message: "Logout successful", data: { user_id } }`

- GET `/auth/google/login`
  - Public
  - 302 to Google OAuth

- GET `/auth/google/callback`
  - Public
  - Handles OAuth callback

- POST `/auth/forgot-password`
  - Public
  - Body (json): `{ email }`
  - 200: generic success

- POST `/auth/reset-password`
  - Public
  - Body (json): `{ token, new_password }`
  - 200: success

---

### Users (Admin only)
All endpoints require: JWT + role `admin`.

- GET `/users`
  - 200: list of users (password omitted)

- POST `/users`
  - Body: `{ name, email, password, role_id? }`
  - 201: created user

- GET `/users/:id`
  - 200: user by id (password omitted)

- PUT `/users/:id`
  - Body: `{ name?, email?, role_id? }`
  - 200: updated user

- DELETE `/users/:id`
  - 204

---

### Roles (Admin only)
All endpoints require: JWT + role `admin`.

- GET `/roles`
  - 200: list roles with permissions

- POST `/roles`
  - Body: `{ name, description?, is_global?, permissions: [{ module, action, field_scope?, allowed_fields?, denied_fields?, content_type_ids? }] }`
  - 201: created role

- GET `/roles/:id`
  - 200: role detail

- PUT `/roles/:id`
  - Body: same shape as create
  - 200: updated role

- DELETE `/roles/:id`
  - 204

- POST `/roles/:id/duplicate`
  - Body: `{ name }`
  - 201: duplicated role

- POST `/roles/assign`
  - Body: `{ user_id, role_id }`
  - 200: user with assigned role

Role permissions

Each role can have one or more permission objects in the `permissions` array:

```json
{
  "module": "ContentEntry",
  "action": "update",
  "field_scope": "custom",
  "allowed_fields": ["title", "slug"],
  "denied_fields": ["internal_notes"],
  "content_type_ids": [1, 2]
}
```

- **module**:
  - Global:
    - `ContentEntry`: mengatur izin untuk entry konten global (create/read/update/delete/approve).
    - `ContentType`: mengatur izin untuk skema konten global (content types & fields).
    - `Media`: media library global (files & folders).
    - `SEO`: operasi terkait SEO (preview, SEO fields).
  - Project:
    - `ProjectContent`: entry konten di dalam project tertentu.
    - `ProjectMedia`: media di dalam project.
    - `ProjectSchema`: skema/tipe konten milik project.
    - `ProjectSettings`: konfigurasi project (nama, deskripsi, dsb).
    - `ProjectMembers`: manajemen member project (invite/read/update/remove).
    - `ProjectOwnership`: aksi khusus seperti transfer kepemilikan project.

- **action**:
  - Umum: `create`, `read`, `update`, `delete`.
  - Khusus konten/workflow: `approve`, `publish`.
  - Khusus project: `invite`, `remove`, `transfer` (kepemilikan), dll sesuai modul yang digunakan.

- **field_scope** (khusus untuk operasi tulis konten, mis. `ContentEntry` / `ProjectContent`):
  - `all`: boleh menulis semua field.
  - `seo_only`: hanya boleh menulis field SEO (mis. `meta_title`, `meta_description`, dsb).
  - `non_seo_only`: hanya boleh menulis field non-SEO (konten utama).
  - `custom`: gunakan `allowed_fields` / `denied_fields` untuk pengaturan granular.

- **allowed_fields** (opsional, dipakai jika `field_scope = "custom"`):
  - Daftar nama field yang **boleh** diubah (contoh: `["title", "slug"]`).
  - Jika diisi, hanya field-field ini yang akan diproses saat tulis; field lain akan diabaikan.

- **denied_fields** (opsional, dipakai jika `field_scope = "custom"`):
  - Daftar nama field yang **tidak boleh** diubah, meskipun ada di `allowed_fields`.
  - Berguna untuk memblokir field sensitif (mis. `["internal_notes", "cost_price"]`).

- **content_type_ids** (opsional, biasanya untuk `ContentEntry` / `ProjectContent`):
  - Daftar ID Content Type yang boleh diakses oleh permission ini (contoh: `[1, 2, 3]`).
  - Jika kosong atau tidak diisi, permission berlaku untuk **semua** content type.

---

### Projects
All endpoints require JWT. Project access is controlled by membership.

- POST `/projects`
  - Body: `{ name (required), description? }`
  - 201: created project (user becomes owner)

- GET `/projects`
  - 200: list of projects where user is a member

- GET `/projects/:id`
  - Requires: user must be project member
  - 200: project details

- PUT `/projects/:id`
  - Requires: project owner or admin role
  - Body: `{ name (required), description? }`
  - 200: updated project

- DELETE `/projects/:id`
  - Requires: project owner only
  - 200: success message

Project Members
- POST `/projects/:id/members`
  - Requires: project owner or admin role
  - Body: `{ user_id (required), role (required) }`
  - 201: added member

- GET `/projects/:id/members`
  - Requires: project member
  - 200: list of project members

- PUT `/projects/:id/members/:member_id`
  - Requires: project owner or admin role
  - Body: `{ role (required) }`
  - 200: updated member role

- DELETE `/projects/:id/members/:member_id`
  - Requires: project owner or admin role
  - 200: removed member

---

### Content
All endpoints under `/content` require JWT. Fine-grained access controlled by permissions via `middleware.PermissionProtected`.

Content Types
- POST `/content/types` (perm: `ContentEntry:create`)
  - Body: `{ name, slug }`
  - 201: content type

- GET `/content/types` (perm: `ContentEntry:read`)
  - 200: list content types with fields and SEO fields

- GET `/content/types/:id` (perm: `ContentEntry:read`)
  - 200: content type by id

- PUT `/content/types/:id` (perm: `ContentEntry:update`)
  - Body: `{ name, slug, enable_seo }`
  - 200: updated content type

- DELETE `/content/types/:id` (perm: `ContentEntry:delete`)
  - 204 (blocked if entries exist)

Fields
- POST `/content/types/:content_type_id/fields` (perm: `ContentEntry:update`)
  - Body: `{ name, type, required, is_seo, unique?, max_length?, min_length?, pattern?, min_value?, max_value?, default_value?, placeholder?, help_text? }`
  - 201: field

- PUT `/content/:content_type_id/fields/:field_id` (perm: `ContentEntry:update`)
  - Body: same shape as create
  - 200: updated field with `validation_rules`

- DELETE `/content/:content_type_id/fields/:field_id` (perm: `ContentEntry:delete`)
  - 204

- GET `/content/fields/:field_id/validation` (perm: `ContentEntry:read`)
  - 200: field validation rules

Entries
- POST `/content/:content_type_id/entries` (perm: `ContentEntry:create`)
  - Accepts `application/json` or `multipart/form-data`
  - Media fields: supply `<field>_media_id` or upload file under `<field>`
  - 201: created entry

- POST `/content/:content_type_id/entries/json` (perm: `ContentEntry:create`)
  - JSON-only variant for media by id
  - 200: created entry

- GET `/content/:content_type_id/entries` (perm: `ContentEntry:read`)
  - Query: `status, created_by, from, to, page, limit`
  - 200: entries with `meta`

- GET `/content/entries` (perm: `ContentEntry:read`)
  - Query: `status, created_by, from, to, page, limit, content_type_id?`
  - 200: all entries across content types with `meta`

- GET `/content/entries/:entry_id` (perm: `ContentEntry:read`)
  - 200: entry by id

- PUT `/content/entries/:entry_id` (perm: `ContentEntry:update`)
  - JSON or multipart; respects field-level permissions
  - 200: updated entry (status set to `draft`)

- DELETE `/content/entries/:entry_id` (perm: `ContentEntry:delete`)
  - 204 (blocked if published)

Translation
- POST `/content/entries/:entry_id/translate` (perm: `ContentEntry:update`)
  - Body: `{ target_lang (required, or inferred from Accept-Language), source_lang?, fields[] }`
  - 200: entry with translated fields in `_i18n[target_lang]`

SEO & Preview
- GET `/content/entries/:entry_id/seo-preview` (perm: `SEO:read`)
  - 200: SEO preview payload

- POST `/content/entries/:entry_id/preview-token` (perm: `ContentEntry:read`)
  - 200: `{ token, expires_at, preview_url, frontend_preview_url? }`

- GET `/content/entries/:entry_id/preview`
  - Public (no auth required, but requires valid preview token)
  - Query: `token` (required, or `X-Preview-Token` header)
  - Validates preview token matches entry_id
  - 200: full entry data with ContentType preloaded
  - 400: missing token, 401: invalid token or token mismatch, 404: entry not found

Relations
- POST `/content/:from_content_id/relations` (perm: `ContentEntry:update`)
  - Body: `{ to_content_id, relation_type }`
  - 201: created relation

- GET `/content/:from_content_id/relations` (perm: `ContentEntry:read`)
  - 200: list relations

- DELETE `/content/relations/:relation_id` (perm: `ContentEntry:delete`)
  - 204

Docs & OpenAPI
- GET `/content/types/:id/api-reference` (perm: `ContentEntry:read`)
- GET `/content/types/:id/openapi` (perm: `ContentEntry:read`)
- GET `/content/types/:id/docs/markdown` (perm: `ContentEntry:read`)

---

### Media
All endpoints require JWT. Permissions for module `Media`.

- GET `/media/folders` (perm: `Media:read`)
- POST `/media/folders` (perm: `Media:create`)
- POST `/media/upload` (perm: `Media:create`)
  - FormData: `file`, optional `folder`, `alt`, `caption`, `tags` (JSON array)
- POST `/media/bulk-upload` (perm: `Media:create`)
  - FormData: `files[]`, optional `folder`
- GET `/media` (perm: `Media:read`) with pagination and filters: `type, folder, search, page, limit`
- GET `/media/search` (perm: `Media:read`) query: `q, page, limit`
- GET `/media/stats` (perm: `Media:read`)
- GET `/media/:id` (perm: `Media:read`)
- PUT `/media/:id` (perm: `Media:update`) body: `{ alt, caption, folder, tags[] }`
- DELETE `/media/:id` (perm: `Media:delete`)

---

### Workflow
All endpoints require JWT. Permissions under `ContentEntry` as indicated.

- POST `/workflow/entries/:entry_id/status` (perm: `ContentEntry:update`)
  - Body: `{ status, comment }`
- POST `/workflow/entries/:entry_id/request-review` (perm: `ContentEntry:update`)
  - Body: `{ comment? }`
- POST `/workflow/entries/:entry_id/approve` (perm: `ContentEntry:approve`)
  - Body: `{ comment? }`
- POST `/workflow/entries/:entry_id/reject` (perm: `ContentEntry:approve`)
  - Body: `{ comment }` (required)
- POST `/workflow/entries/:entry_id/publish` (perm: `ContentEntry:approve`)
  - Body: `{ comment? }`
- GET `/workflow/entries/:entry_id/history` (perm: `ContentEntry:read`)
- POST `/workflow/entries/:entry_id/comments` (perm: `ContentEntry:read`) body: `{ comment, is_private }`
- GET `/workflow/entries/:entry_id/comments` (perm: `ContentEntry:read`) query: `include_private`
- POST `/workflow/entries/:entry_id/assign` (perm: `ContentEntry:approve`) body: `{ assigned_to, due_date? }`
- GET `/workflow/assignments` (perm: `ContentEntry:read`) query: `status`
- PUT `/workflow/assignments/:assignment_id/complete` (perm: `ContentEntry:update`)
- GET `/workflow/content-types/:content_type_id/entries` (perm: `ContentEntry:read`) query: `status`
- GET `/workflow/content-types/:content_type_id/stats` (perm: `ContentEntry:read`)

---

### Search
All endpoints require JWT. Permissions `ContentEntry:read`.

- GET `/search/entries`
  - Query: `q, status, page, limit, sort_by, order_by, from, to, content_type_ids (comma), fields (comma), tags (comma), created_by`
  - 200: entries with meta

- POST `/search/advanced`
  - Body: `{ query, content_type_ids[], fields[], status, created_by, tags[], from_date, to_date, filters{}, page, limit, sort_by, order_by }`
  - 200: entries with meta

- GET `/search/facets`
  - Query: `q, content_type_ids (comma)`
  - 200: facet aggregates

- GET `/search/autocomplete`
  - Query: `field, prefix, content_type_id, limit`
  - 200: `{ suggestions[] }`

- GET `/search/entries/:entry_id/related`
  - Query: `type`
  - 200: related entries

- POST `/search/bulk`
  - Body: `{ query, content_type_ids[], limit, group_by_type }`
  - 200: grouped or flat results

- GET `/search/export`
  - Query: `q, status, sort_by, order_by, format=json|csv`
  - 200: file download (json implemented)

- GET `/search/stats` (perm: `ContentEntry:read`)
  - 200: `{ total_searches, unique_queries, avg_results_per_search, popular_queries[], zero_result_queries[] }`

- GET `/search/suggestions` (perm: `ContentEntry:read`)
  - Query: `q` (required)
  - 200: `{ suggestions[], query }`

---

### Static Files
- GET `/uploads/*` serves uploaded files (no auth).

---

### Error Codes
- UNAUTHORIZED, FORBIDDEN, NOT_FOUND, CONFLICT, VALIDATION_ERROR, INTERNAL_ERROR

---

### Notes
- Field-level permissions filter writes; unknown fields are ignored or rejected depending on endpoint.
- Published entries must be unpublished before edits/deletes.
- Rate limits: login (5/15m), register (5/1m via group limiter), refresh (3/5m).
