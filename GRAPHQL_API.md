# CMS GraphQL API Documentation

## Overview

Sistem CMS ini sekarang mendukung **GraphQL API** yang lengkap di samping REST API yang sudah ada. GraphQL API menyediakan akses ke semua fitur CMS dengan fleksibilitas query yang tinggi.

## Endpoints

### GraphQL Endpoint
- **URL**: `POST /graphql`
- **Description**: Endpoint utama untuk GraphQL queries dan mutations
- **Content-Type**: `application/json`

### GraphiQL Playground
- **URL**: `GET /graphql`
- **Description**: Interactive GraphQL playground untuk testing dan development
- **Access**: Browser-based interface

### Batch GraphQL
- **URL**: `POST /graphql/batch`
- **Description**: Endpoint untuk batch requests (multiple queries dalam satu request)
- **Limit**: Maksimal 10 requests per batch

## Authentication

GraphQL API menggunakan sistem autentikasi yang sama dengan REST API:

```http
Authorization: Bearer <jwt_token>
```

## Schema Types

### Core Types

#### User
```graphql
type User {
  id: ID!
  name: String!
  email: String!
  provider: String!
  status: UserStatus!
  roleId: Int!
  role: Role
  profile: String
  createdAt: Time!
  updatedAt: Time!
}
```

#### Role
```graphql
type Role {
  id: ID!
  name: String!
  description: String
  permissions: [Permission!]!
  createdAt: Time!
  updatedAt: Time!
}
```

#### ContentType
```graphql
type ContentType {
  id: ID!
  name: String!
  slug: String!
  enableSeo: Boolean!
  fields: [ContentField!]!
  seoFields: [ContentField!]!
  createdAt: Time!
  updatedAt: Time!
}
```

#### ContentEntry
```graphql
type ContentEntry {
  id: ID!
  contentTypeId: Int!
  contentType: ContentType
  data: JSON!
  status: WorkflowStatus!
  createdBy: Int
  updatedBy: Int
  creator: User
  updater: User
  createdAt: Time!
  updatedAt: Time!
  publishedAt: Time
}
```

#### MediaFile
```graphql
type MediaFile {
  id: ID!
  fileName: String!
  url: String!
  type: String!
  size: Int!
  width: Int
  height: Int
  folder: String!
  tags: JSON
  alt: String!
  caption: String
  uploadedBy: Int!
  uploader: User
  createdAt: Time!
  updatedAt: Time!
}
```

### Enums

```graphql
enum WorkflowStatus {
  DRAFT
  IN_REVIEW
  READY_FOR_APPROVAL
  APPROVED
  PUBLISHED
  REJECTED
}

enum UserStatus {
  ACTIVE
  INACTIVE
  SUSPENDED
}
```

### Scalars

```graphql
scalar Time    # RFC3339 formatted time
scalar JSON    # JSON object
```

## Queries

### Authentication

#### Get Current User
```graphql
query Me {
  me {
    id
    name
    email
    role {
      name
      permissions {
        module
        action
      }
    }
  }
}
```

### User Management

#### List Users
```graphql
query GetUsers($limit: Int, $offset: Int) {
  users(limit: $limit, offset: $offset) {
    id
    name
    email
    status
    role {
      name
    }
    createdAt
  }
}
```

#### Get User by ID
```graphql
query GetUser($id: ID!) {
  user(id: $id) {
    id
    name
    email
    status
    role {
      id
      name
      description
      permissions {
        module
        action
      }
    }
    createdAt
    updatedAt
  }
}
```

### Role Management

#### List Roles
```graphql
query GetRoles {
  roles {
    id
    name
    description
    permissions {
      module
      action
      fieldScope
    }
    createdAt
  }
}
```

#### Get Role by ID
```graphql
query GetRole($id: ID!) {
  role(id: $id) {
    id
    name
    description
    permissions {
      module
      action
      fieldScope
      allowedFields
      deniedFields
      contentTypeIds
    }
  }
}
```

### Content Management

#### List Content Types
```graphql
query GetContentTypes {
  contentTypes {
    id
    name
    slug
    enableSeo
    fields {
      name
      type
      required
      unique
      maxLength
      minLength
      defaultValue
    }
  }
}
```

#### Get Content Type by ID
```graphql
query GetContentType($id: ID!) {
  contentType(id: $id) {
    id
    name
    slug
    enableSeo
    fields {
      id
      name
      type
      required
      isSeo
      unique
      maxLength
      minLength
      pattern
      minValue
      maxValue
      defaultValue
      placeholder
      helpText
    }
    seoFields {
      name
      type
      required
    }
  }
}
```

#### List Content Entries
```graphql
query GetContentEntries($contentTypeId: Int!, $limit: Int, $offset: Int) {
  contentEntries(contentTypeId: $contentTypeId, limit: $limit, offset: $offset) {
    id
    data
    status
    creator {
      name
      email
    }
    updater {
      name
      email
    }
    createdAt
    updatedAt
    publishedAt
  }
}
```

#### Get Content Entry by ID
```graphql
query GetContentEntry($id: ID!) {
  contentEntry(id: $id) {
    id
    contentTypeId
    contentType {
      name
      slug
    }
    data
    status
    creator {
      name
      email
    }
    updater {
      name
      email
    }
    createdAt
    updatedAt
    publishedAt
  }
}
```

### Media Management

#### List Media Files
```graphql
query GetMediaFiles($limit: Int, $offset: Int, $folder: String) {
  mediaFiles(limit: $limit, offset: $offset, folder: $folder) {
    id
    fileName
    url
    type
    size
    width
    height
    folder
    tags
    alt
    caption
    uploader {
      name
    }
    createdAt
  }
}
```

#### Get Media File by ID
```graphql
query GetMediaFile($id: ID!) {
  mediaFile(id: $id) {
    id
    fileName
    url
    type
    size
    width
    height
    folder
    tags
    alt
    caption
    uploader {
      name
      email
    }
    createdAt
    updatedAt
  }
}
```

## Mutations

### Authentication

#### Login
```graphql
mutation Login($email: String!, $password: String!) {
  login(email: $email, password: $password) {
    token
    refreshToken
    user {
      id
      name
      email
      role {
        name
      }
    }
  }
}
```

#### Register
```graphql
mutation Register($name: String!, $email: String!, $password: String!) {
  register(name: $name, email: $email, password: $password) {
    token
    refreshToken
    user {
      id
      name
      email
      role {
        name
      }
    }
  }
}
```

### User Management

#### Create User
```graphql
mutation CreateUser($name: String!, $email: String!, $password: String!, $roleId: Int!) {
  createUser(name: $name, email: $email, password: $password, roleId: $roleId) {
    id
    name
    email
    status
    role {
      name
    }
  }
}
```

#### Update User
```graphql
mutation UpdateUser($id: ID!, $name: String, $email: String, $password: String, $roleId: Int, $status: UserStatus) {
  updateUser(id: $id, name: $name, email: $email, password: $password, roleId: $roleId, status: $status) {
    id
    name
    email
    status
    role {
      name
    }
  }
}
```

#### Delete User
```graphql
mutation DeleteUser($id: ID!) {
  deleteUser(id: $id)
}
```

### Role Management

#### Create Role
```graphql
mutation CreateRole($name: String!, $description: String) {
  createRole(name: $name, description: $description) {
    id
    name
    description
    createdAt
  }
}
```

#### Update Role
```graphql
mutation UpdateRole($id: ID!, $name: String, $description: String) {
  updateRole(id: $id, name: $name, description: $description) {
    id
    name
    description
    updatedAt
  }
}
```

#### Delete Role
```graphql
mutation DeleteRole($id: ID!) {
  deleteRole(id: $id)
}
```

### Content Management

#### Create Content Type
```graphql
mutation CreateContentType($name: String!, $slug: String!, $enableSeo: Boolean) {
  createContentType(name: $name, slug: $slug, enableSeo: $enableSeo) {
    id
    name
    slug
    enableSeo
    createdAt
  }
}
```

#### Update Content Type
```graphql
mutation UpdateContentType($id: ID!, $name: String, $slug: String, $enableSeo: Boolean) {
  updateContentType(id: $id, name: $name, slug: $slug, enableSeo: $enableSeo) {
    id
    name
    slug
    enableSeo
    updatedAt
  }
}
```

#### Delete Content Type
```graphql
mutation DeleteContentType($id: ID!) {
  deleteContentType(id: $id)
}
```

#### Create Content Entry
```graphql
mutation CreateContentEntry($contentTypeId: ID!, $data: JSON!) {
  createContentEntry(contentTypeId: $contentTypeId, data: $data) {
    id
    contentTypeId
    data
    status
    creator {
      name
    }
    createdAt
  }
}
```

#### Update Content Entry
```graphql
mutation UpdateContentEntry($id: ID!, $data: JSON, $status: WorkflowStatus) {
  updateContentEntry(id: $id, data: $data, status: $status) {
    id
    data
    status
    updater {
      name
    }
    updatedAt
  }
}
```

#### Delete Content Entry
```graphql
mutation DeleteContentEntry($id: ID!) {
  deleteContentEntry(id: $id)
}
```

## Example Requests

### Complete Example: Create and Query Content

1. **Login to get token**:
```json
{
  "query": "mutation Login($email: String!, $password: String!) { login(email: $email, password: $password) { token user { id name email role { name } } } }",
  "variables": {
    "email": "admin@example.com",
    "password": "password123"
  }
}
```

2. **Get all content types**:
```json
{
  "query": "query GetContentTypes { contentTypes { id name slug enableSeo fields { name type required } } }"
}
```

3. **Create content entry**:
```json
{
  "query": "mutation CreateContentEntry($contentTypeId: ID!, $data: JSON!) { createContentEntry(contentTypeId: $contentTypeId, data: $data) { id data status creator { name } createdAt } }",
  "variables": {
    "contentTypeId": "1",
    "data": {
      "title": "My First Article",
      "body": "This is the content of my article",
      "slug": "my-first-article"
    }
  }
}
```

4. **Query content entries with pagination**:
```json
{
  "query": "query GetContentEntries($contentTypeId: Int!, $limit: Int, $offset: Int) { contentEntries(contentTypeId: $contentTypeId, limit: $limit, offset: $offset) { id data status creator { name } createdAt } }",
  "variables": {
    "contentTypeId": 1,
    "limit": 10,
    "offset": 0
  }
}
```

## Batch Requests
## Workflow

New queries and mutations aligned with REST workflow endpoints:

- Queries:
  - `workflowHistory(entryId: Int!): [WorkflowHistory!]!`
  - `workflowComments(entryId: Int!, includePrivate: Boolean): [WorkflowComment!]!`
  - `workflowAssignments(status: String): [WorkflowAssignment!]!`
  - `workflowStats(contentTypeId: Int!): WorkflowStats!`

- Mutations:
  - `changeContentStatus(entryId: Int!, status: WorkflowStatus!, comment: String): WorkflowHistory!`
  - `requestReview(entryId: Int!, comment: String): WorkflowHistory!`
  - `approveEntry(entryId: Int!, comment: String): WorkflowHistory!`
  - `rejectEntry(entryId: Int!, comment: String): WorkflowHistory!`
  - `publishEntry(entryId: Int!, comment: String): WorkflowHistory!`
  - `addWorkflowComment(entryId: Int!, comment: String!, isPrivate: Boolean): WorkflowComment!`
  - `assignEntry(entryId: Int!, assignedTo: Int!, dueDate: Time): WorkflowAssignment!`

## Content Fields

Field management parity with REST:
- `addContentField(...)` to add a field to a content type
- `updateContentField(id: ID!, ...)` to update a field
- `deleteContentField(id: ID!)` to remove a field

## SEO & Search

- `seoPreview(entryId: Int!): JSON` returns SEO-related data like slug, meta fields
- `searchFacets(query: String, contentTypeIds: [Int!]): JSON` returns aggregated facets
- `autocomplete(field: String!, prefix: String!, contentTypeId: Int!, limit: Int): [String!]!`

You can send multiple queries in a single request using the `/graphql/batch` endpoint:

```json
[
  {
    "query": "query { users { id name email } }"
  },
  {
    "query": "query { contentTypes { id name slug } }"
  },
  {
    "query": "query { roles { id name description } }"
  }
]
```

## Error Handling

GraphQL API returns errors in a standardized format:

```json
{
  "data": null,
  "errors": [
    {
      "message": "Permission denied",
      "locations": [
        {
          "line": 2,
          "column": 3
        }
      ],
      "path": [
        "users"
      ]
    }
  ]
}
```

## Permissions

GraphQL API respects the same permission system as REST API:

- **User Management**: Requires admin role or specific user permissions
- **Role Management**: Requires admin role
- **Content Management**: Based on ContentEntry permissions
- **Media Management**: Based on Media permissions

## Performance Tips

1. **Use specific fields**: Only request the fields you need
2. **Pagination**: Use `limit` and `offset` for large datasets
3. **Batch requests**: Use batch endpoint for multiple queries
4. **Caching**: Consider implementing caching for frequently accessed data

## Development Tools

### GraphiQL Playground
Access the interactive GraphQL playground at `GET /graphql` to:
- Test queries and mutations
- Explore the schema
- View documentation
- Debug requests

### Sample Queries
The playground includes pre-loaded sample queries for common operations.

## Migration from REST API

The GraphQL API provides the same functionality as the REST API but with more flexibility:

- **Single endpoint**: All operations through `/graphql`
- **Flexible queries**: Request only the data you need
- **Strong typing**: Schema validation and auto-completion
- **Real-time**: Support for subscriptions (future enhancement)

## Support

For issues or questions regarding the GraphQL API:
1. Check the GraphiQL playground for schema exploration
2. Review the error messages for debugging
3. Ensure proper authentication and permissions
4. Use the batch endpoint for complex operations
