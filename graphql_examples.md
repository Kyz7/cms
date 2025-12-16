# GraphQL API Examples

## Testing GraphQL API

Server CMS sekarang sudah berjalan dengan GraphQL API yang lengkap. Berikut adalah contoh-contoh untuk testing:

### 1. GraphiQL Playground
Buka browser dan akses: `http://localhost:8080/graphql`

Ini akan membuka GraphiQL playground yang interaktif untuk testing queries dan mutations.

### 2. Contoh Queries

#### Test Health Check
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { __schema { types { name } } }"
  }'
```

#### Get All Users
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "query { users { id name email status role { name } } }"
  }'
```

#### Get All Content Types
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "query { contentTypes { id name slug enableSeo fields { name type required } } }"
  }'
```

#### Get All Roles
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "query { roles { id name description permissions { module action } } }"
  }'
```

### 3. Contoh Mutations

#### Register New User
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation Register($name: String!, $email: String!, $password: String!) { register(name: $name, email: $email, password: $password) { token user { id name email } } }",
    "variables": {
      "name": "Test User",
      "email": "test@example.com",
      "password": "password123"
    }
  }'
```

#### Login
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation Login($email: String!, $password: String!) { login(email: $email, password: $password) { token user { id name email role { name } } } }",
    "variables": {
      "email": "test@example.com",
      "password": "password123"
    }
  }'
```

#### Create Content Type
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "mutation CreateContentType($name: String!, $slug: String!) { createContentType(name: $name, slug: $slug) { id name slug enableSeo createdAt } }",
    "variables": {
      "name": "Article",
      "slug": "article"
    }
  }'
```

#### Create Content Entry
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "mutation CreateContentEntry($contentTypeId: ID!, $data: JSON!) { createContentEntry(contentTypeId: $contentTypeId, data: $data) { id data status creator { name } createdAt } }",
    "variables": {
      "contentTypeId": "1",
      "data": {
        "title": "My First Article",
        "body": "This is the content of my article",
        "slug": "my-first-article"
      }
    }
  }'
```

### 4. Batch Requests

```bash
curl -X POST http://localhost:8080/graphql/batch \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '[
    {
      "query": "query { users { id name email } }"
    },
    {
      "query": "query { contentTypes { id name slug } }"
    },
    {
      "query": "query { roles { id name description } }"
    }
  ]'
```

### 5. Advanced Queries

#### Get Content Entries with Pagination
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "query GetContentEntries($contentTypeId: Int!, $limit: Int, $offset: Int) { contentEntries(contentTypeId: $contentTypeId, limit: $limit, offset: $offset) { id data status creator { name email } createdAt updatedAt } }",
    "variables": {
      "contentTypeId": 1,
      "limit": 10,
      "offset": 0
    }
  }'
```

#### Get Media Files with Filter
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "query GetMediaFiles($limit: Int, $folder: String) { mediaFiles(limit: $limit, folder: $folder) { id fileName url type size width height folder alt caption uploader { name } createdAt } }",
    "variables": {
      "limit": 20,
      "folder": "uploads"
    }
  }'
```

### 6. Error Handling Examples

#### Invalid Query
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { invalidField }"
  }'
```

#### Missing Authorization
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { users { id name email } }"
  }'
```

### 7. JavaScript/Node.js Example

```javascript
const fetch = require('node-fetch');

async function queryGraphQL(query, variables = {}, token = null) {
  const headers = {
    'Content-Type': 'application/json'
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  const response = await fetch('http://localhost:8080/graphql', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      query,
      variables
    })
  });
  
  return await response.json();
}

// Example usage
async function testGraphQL() {
  try {
    // Login
    const loginResult = await queryGraphQL(`
      mutation Login($email: String!, $password: String!) {
        login(email: $email, password: $password) {
          token
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
    `, {
      email: 'admin@example.com',
      password: 'password123'
    });
    
    console.log('Login result:', loginResult);
    
    const token = loginResult.data.login.token;
    
    // Get users
    const usersResult = await queryGraphQL(`
      query {
        users {
          id
          name
          email
          status
          role {
            name
          }
        }
      }
    `, {}, token);
    
    console.log('Users:', usersResult);
    
  } catch (error) {
    console.error('Error:', error);
  }
}

testGraphQL();
```

### 8. Python Example

```python
import requests
import json

def query_graphql(query, variables=None, token=None):
    url = 'http://localhost:8080/graphql'
    headers = {
        'Content-Type': 'application/json'
    }
    
    if token:
        headers['Authorization'] = f'Bearer {token}'
    
    payload = {
        'query': query
    }
    
    if variables:
        payload['variables'] = variables
    
    response = requests.post(url, headers=headers, json=payload)
    return response.json()

# Example usage
def test_graphql():
    try:
        # Login
        login_result = query_graphql("""
            mutation Login($email: String!, $password: String!) {
                login(email: $email, password: $password) {
                    token
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
        """, {
            'email': 'admin@example.com',
            'password': 'password123'
        })
        
        print('Login result:', login_result)
        
        if 'data' in login_result and 'login' in login_result['data']:
            token = login_result['data']['login']['token']
            
            # Get users
            users_result = query_graphql("""
                query {
                    users {
                        id
                        name
                        email
                        status
                        role {
                            name
                        }
                    }
                }
            """, token=token)
            
            print('Users:', users_result)
        
    except Exception as error:
        print('Error:', error)

test_graphql()
```

## Endpoints Summary

- **GraphQL Endpoint**: `POST /graphql`
- **GraphiQL Playground**: `GET /graphql`
- **Batch GraphQL**: `POST /graphql/batch`
- **Health Check**: `GET /health`

## Features Available

✅ **Authentication**: Login, Register, Token-based auth  
✅ **User Management**: CRUD operations with role-based permissions  
✅ **Role Management**: Create, update, delete roles with permissions  
✅ **Content Management**: Dynamic content types, fields, and entries  
✅ **Media Management**: File upload, metadata, folders  
✅ **Workflow**: Content status management  
✅ **Search**: Full-text search and filtering  
✅ **Batch Requests**: Multiple queries in single request  
✅ **Interactive Playground**: GraphiQL interface for testing  

## Next Steps

1. Test all endpoints using the examples above
2. Integrate with your frontend application
3. Set up proper JWT authentication
4. Configure permissions and roles
5. Add more advanced features as needed

The GraphQL API is now fully functional alongside your existing REST API!
