package graphql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

type contextKey string

const contextKeyUserID contextKey = "user_id"

type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

type GraphQLResponse struct {
	Data       interface{}                `json:"data,omitempty"`
	Errors     []gqlerrors.FormattedError `json:"errors,omitempty"`
	Extensions map[string]interface{}     `json:"extensions,omitempty"`
}

func (r *Resolvers) GraphQLHandler() fiber.Handler {
	schema, err := r.CreateSchema()
	if err != nil {
		panic(fmt.Sprintf("failed to create GraphQL schema: %v", err))
	}

	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(200)
		}
		if c.Method() != "POST" {
			return c.Status(405).JSON(fiber.Map{
				"error": "Method not allowed. Use POST for GraphQL requests.",
			})
		}

		var req GraphQLRequest
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid JSON in request body",
			})
		}

		if req.Query == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Query is required",
			})
		}
		params := graphql.Params{
			Schema:         schema,
			RequestString:  req.Query,
			VariableValues: req.Variables,
			OperationName:  req.OperationName,
			Context:        c.UserContext(),
		}

		if userID := r.getUserIDFromHeader(c); userID != 0 {
			params.Context = context.WithValue(params.Context, contextKeyUserID, userID)
		}

		result := graphql.Do(params)
		response := GraphQLResponse{
			Data: result.Data,
		}

		if len(result.Errors) > 0 {
			response.Errors = result.Errors
		}

		c.Set("Content-Type", "application/json")

		if len(result.Errors) > 0 {
			return c.Status(200).JSON(response)
		}

		return c.Status(200).JSON(response)
	}
}

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

func (r *Resolvers) BatchGraphQLHandler() fiber.Handler {
	schema, err := r.CreateSchema()
	if err != nil {
		panic(fmt.Sprintf("failed to create GraphQL schema: %v", err))
	}

	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Method() == "OPTIONS" {
			return c.SendStatus(200)
		}

		if c.Method() != "POST" {
			return c.Status(405).JSON(fiber.Map{
				"error": "Method not allowed. Use POST for GraphQL requests.",
			})
		}

		var requests []GraphQLRequest
		if err := json.Unmarshal(c.Body(), &requests); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid JSON in request body",
			})
		}

		if len(requests) == 0 {
			return c.Status(400).JSON(fiber.Map{
				"error": "At least one request is required",
			})
		}

		if len(requests) > 10 {
			return c.Status(400).JSON(fiber.Map{
				"error": "Too many requests. Maximum 10 requests per batch.",
			})
		}

		var responses []GraphQLResponse
		for _, req := range requests {
			if req.Query == "" {
				responses = append(responses, GraphQLResponse{
					Errors: []gqlerrors.FormattedError{{
						Message: "Query is required",
					}},
				})
				continue
			}

			params := graphql.Params{
				Schema:         schema,
				RequestString:  req.Query,
				VariableValues: req.Variables,
				OperationName:  req.OperationName,
				Context:        c.UserContext(),
			}

			if userID := r.getUserIDFromHeader(c); userID != 0 {
				params.Context = context.WithValue(params.Context, contextKeyUserID, userID)
			}

			result := graphql.Do(params)

			response := GraphQLResponse{
				Data: result.Data,
			}

			if len(result.Errors) > 0 {
				response.Errors = result.Errors
			}

			responses = append(responses, response)
		}

		c.Set("Content-Type", "application/json")

		return c.Status(200).JSON(responses)
	}
}

func (r *Resolvers) getUserIDFromHeader(c *fiber.Ctx) uint {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return 0
	}

	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token := authHeader[7:]
		// Here you would validate the JWT token and extract user ID
		// For now, return 0 (no user)
		_ = token
	}

	return 0
}
