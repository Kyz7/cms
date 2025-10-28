package graphql

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// GraphQLRequest represents the structure of a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// GraphQLResponse represents the structure of a GraphQL response
type GraphQLResponse struct {
	Data       interface{}                `json:"data,omitempty"`
	Errors     []gqlerrors.FormattedError `json:"errors,omitempty"`
	Extensions map[string]interface{}     `json:"extensions,omitempty"`
}

// GraphQLHandler creates a Fiber handler for GraphQL requests
func (r *Resolvers) GraphQLHandler() fiber.Handler {
	schema, err := r.CreateSchema()
	if err != nil {
		panic(fmt.Sprintf("failed to create GraphQL schema: %v", err))
	}

	return func(c *fiber.Ctx) error {
		// Set CORS headers
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if c.Method() == "OPTIONS" {
			return c.SendStatus(200)
		}

		// Only allow POST requests
		if c.Method() != "POST" {
			return c.Status(405).JSON(fiber.Map{
				"error": "Method not allowed. Use POST for GraphQL requests.",
			})
		}

		// Parse request body
		var req GraphQLRequest
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid JSON in request body",
			})
		}

		// Validate query
		if req.Query == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Query is required",
			})
		}

		// Set up GraphQL execution parameters
		params := graphql.Params{
			Schema:         schema,
			RequestString:  req.Query,
			VariableValues: req.Variables,
			OperationName:  req.OperationName,
			Context:        c.Context(),
		}

		// Add user context if available
		if userID := r.getUserIDFromHeader(c); userID != 0 {
			// For now, we'll skip context handling
			// params.Context = contextWithUserID(params.Context, userID)
		}

		// Execute GraphQL query
		result := graphql.Do(params)

		// Create response
		response := GraphQLResponse{
			Data: result.Data,
		}

		// Add errors if any
		if len(result.Errors) > 0 {
			response.Errors = result.Errors
		}

		// Set content type
		c.Set("Content-Type", "application/json")

		// Return response
		if len(result.Errors) > 0 {
			return c.Status(200).JSON(response)
		}

		return c.Status(200).JSON(response)
	}
}

// GraphiQLHandler creates a handler for GraphiQL interface
func (r *Resolvers) GraphiQLHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>CMS GraphQL Playground</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
    <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root">
        <style>
            body {
                background-color: rgb(23, 42, 58);
                font-family: Open Sans, sans-serif;
                height: 90vh;
            }
            #root {
                height: 100%;
                width: 100%;
                display: flex;
                align-items: center;
                justify-content: center;
            }
            .loading {
                color: white;
                font-size: 32px;
                font-weight: 200;
                letter-spacing: 1px;
                text-anchor: middle;
            }
            .loading:after {
                content: 'GraphQL Playground';
                position: absolute;
                left: 50%;
                top: 50%;
                transform: translate(-50%, -60%);
            }
        </style>
        <div class="loading">Loading...</div>
    </div>
    <script>
        window.addEventListener('load', function (event) {
            const root = document.getElementById('root');
            root.classList.remove('loading');
            root.innerHTML = '';
            GraphQLPlayground.init(root, {
                endpoint: window.location.origin + '/graphql',
                settings: {
                    'request.credentials': 'include',
                },
                tabs: [
                    {
                        endpoint: window.location.origin + '/graphql',
                        query: ` + "`" + `
# Welcome to CMS GraphQL API
# Try these sample queries:

# Get all users
query GetUsers {
  users {
    id
    name
    email
    role {
      name
    }
  }
}

# Get all content types
query GetContentTypes {
  contentTypes {
    id
    name
    slug
    fields {
      name
      type
      required
    }
  }
}

# Get content entries
query GetContentEntries($contentTypeId: Int!) {
  contentEntries(contentTypeId: $contentTypeId) {
    id
    data
    status
    creator {
      name
    }
  }
}

# Login
mutation Login($email: String!, $password: String!) {
  login(email: $email, password: $password) {
    token
    user {
      id
      name
      email
    }
  }
}
` + "`" + `,
                        variables: {
                            contentTypeId: 1
                        }
                    }
                ]
            })
        });
    </script>
</body>
</html>`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	}
}

// BatchGraphQLHandler creates a handler for batch GraphQL requests
func (r *Resolvers) BatchGraphQLHandler() fiber.Handler {
	schema, err := r.CreateSchema()
	if err != nil {
		panic(fmt.Sprintf("failed to create GraphQL schema: %v", err))
	}

	return func(c *fiber.Ctx) error {
		// Set CORS headers
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if c.Method() == "OPTIONS" {
			return c.SendStatus(200)
		}

		// Only allow POST requests
		if c.Method() != "POST" {
			return c.Status(405).JSON(fiber.Map{
				"error": "Method not allowed. Use POST for GraphQL requests.",
			})
		}

		// Parse request body
		var requests []GraphQLRequest
		if err := json.Unmarshal(c.Body(), &requests); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid JSON in request body",
			})
		}

		// Validate requests
		if len(requests) == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "At least one request is required",
			})
		}

		// Limit batch size
		if len(requests) > 10 {
			return c.Status(400).JSON(fiber.Map{
				"error": "Too many requests. Maximum 10 requests per batch.",
			})
		}

		// Process each request
		var responses []GraphQLResponse
		for _, req := range requests {
			// Validate query
			if req.Query == "" {
				responses = append(responses, GraphQLResponse{
					Errors: []gqlerrors.FormattedError{{
						Message: "Query is required",
					}},
				})
				continue
			}

			// Set up GraphQL execution parameters
			params := graphql.Params{
				Schema:         schema,
				RequestString:  req.Query,
				VariableValues: req.Variables,
				OperationName:  req.OperationName,
				Context:        c.Context(),
			}

			// Add user context if available
			if userID := r.getUserIDFromHeader(c); userID != 0 {
				// For now, we'll skip context handling
				// params.Context = contextWithUserID(params.Context, userID)
			}

			// Execute GraphQL query
			result := graphql.Do(params)

			// Create response
			response := GraphQLResponse{
				Data: result.Data,
			}

			// Add errors if any
			if len(result.Errors) > 0 {
				response.Errors = result.Errors
			}

			responses = append(responses, response)
		}

		// Set content type
		c.Set("Content-Type", "application/json")

		// Return responses
		return c.Status(200).JSON(responses)
	}
}

// getUserIDFromHeader extracts user ID from Authorization header
func (r *Resolvers) getUserIDFromHeader(c *fiber.Ctx) uint {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0
	}

	// Extract token from "Bearer <token>" format
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		// Here you would validate the JWT token and extract user ID
		// For now, return 0 (no user)
		_ = token
	}

	return 0
}

// contextWithUserID adds user ID to context
func contextWithUserID(ctx interface{}, userID uint) interface{} {
	// This would be implemented based on your context handling
	// For now, just return the original context
	return ctx
}

// Utility function to read request body
func readRequestBody(c *fiber.Ctx) ([]byte, error) {
	body := c.Body()
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	return body, nil
}

// Utility function to parse GraphQL request from body
func parseGraphQLRequest(body []byte) (*GraphQLRequest, error) {
	var req GraphQLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL request: %v", err)
	}
	return &req, nil
}
