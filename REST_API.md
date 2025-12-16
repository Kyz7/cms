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
  - Body: `{ name, description?, permissions: [{ module, action, field_scope?, allowed_fields?, denied_fields?, content_type_ids? }] }`
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

- PUT `/content/fields/:field_id` (perm: `ContentEntry:update`)
  - Body: same shape as create
  - 200: updated field with `validation_rules`

- DELETE `/content/fields/:field_id` (perm: `ContentEntry:delete`)
  - 204

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

- GET `/content/entries/:entry_id` (perm: `ContentEntry:read`)
  - 200: entry by id

- PUT `/content/entries/:entry_id` (perm: `ContentEntry:update`)
  - JSON or multipart; respects field-level permissions
  - 200: updated entry (status set to `draft`)

- DELETE `/content/entries/:entry_id` (perm: `ContentEntry:delete`)
  - 204 (blocked if published)

SEO
- GET `/content/entries/:entry_id/seo-preview` (perm: `SEO:read`)
  - 200: SEO preview payload

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
