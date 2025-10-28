package graphql

import (
	"github.com/graphql-go/graphql"
	"gorm.io/gorm"
)

type Resolvers struct {
	DB *gorm.DB
}

// NewResolvers creates a new resolvers instance
func NewResolvers(db *gorm.DB) *Resolvers {
	return &Resolvers{
		DB: db,
	}
}

// CreateSchema creates the complete GraphQL schema
func (r *Resolvers) CreateSchema() (graphql.Schema, error) {
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type:    UserType,
				Resolve: r.MeResolver,
			},
			"users": &graphql.Field{
				Type: graphql.NewList(UserType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: r.UsersResolver,
			},
			"user": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.UserResolver,
			},
			"roles": &graphql.Field{
				Type:    graphql.NewList(RoleType),
				Resolve: r.RolesResolver,
			},
			"role": &graphql.Field{
				Type: RoleType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.RoleResolver,
			},
			"contentTypes": &graphql.Field{
				Type:    graphql.NewList(ContentTypeType),
				Resolve: r.ContentTypesResolver,
			},
			"contentType": &graphql.Field{
				Type: ContentTypeType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.ContentTypeResolver,
			},
			"contentEntries": &graphql.Field{
				Type: graphql.NewList(ContentEntryType),
				Args: graphql.FieldConfigArgument{
					"contentTypeId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: r.ContentEntriesResolver,
			},
			"contentEntry": &graphql.Field{
				Type: ContentEntryType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.ContentEntryResolver,
			},
			"mediaFiles": &graphql.Field{
				Type: graphql.NewList(MediaFileType),
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"folder": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: r.MediaFilesResolver,
			},
			"mediaFile": &graphql.Field{
				Type: MediaFileType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.MediaFileResolver,
			},
		},
	})

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"login": &graphql.Field{
				Type: AuthPayloadType,
				Args: graphql.FieldConfigArgument{
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: r.LoginResolver,
			},
			"register": &graphql.Field{
				Type: AuthPayloadType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: r.RegisterResolver,
			},
			"createUser": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"roleId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: r.CreateUserResolver,
			},
			"updateUser": &graphql.Field{
				Type: UserType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"name": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"email": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"roleId": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"status": &graphql.ArgumentConfig{
						Type: UserStatusEnum,
					},
				},
				Resolve: r.UpdateUserResolver,
			},
			"deleteUser": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.DeleteUserResolver,
			},
			"createRole": &graphql.Field{
				Type: RoleType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"description": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: r.CreateRoleResolver,
			},
			"updateRole": &graphql.Field{
				Type: RoleType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"name": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"description": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: r.UpdateRoleResolver,
			},
			"deleteRole": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.DeleteRoleResolver,
			},
			"createContentType": &graphql.Field{
				Type: ContentTypeType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"slug": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"enableSeo": &graphql.ArgumentConfig{
						Type: graphql.Boolean,
					},
				},
				Resolve: r.CreateContentTypeResolver,
			},
			"updateContentType": &graphql.Field{
				Type: ContentTypeType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"name": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"slug": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"enableSeo": &graphql.ArgumentConfig{
						Type: graphql.Boolean,
					},
				},
				Resolve: r.UpdateContentTypeResolver,
			},
			"deleteContentType": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.DeleteContentTypeResolver,
			},
			"createContentEntry": &graphql.Field{
				Type: ContentEntryType,
				Args: graphql.FieldConfigArgument{
					"contentTypeId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"data": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(JSONType),
					},
				},
				Resolve: r.CreateContentEntryResolver,
			},
			"updateContentEntry": &graphql.Field{
				Type: ContentEntryType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"data": &graphql.ArgumentConfig{
						Type: JSONType,
					},
					"status": &graphql.ArgumentConfig{
						Type: WorkflowStatusEnum,
					},
				},
				Resolve: r.UpdateContentEntryResolver,
			},
			"deleteContentEntry": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.DeleteContentEntryResolver,
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})

	return schema, err
}

// Helper function to get user ID from context
func (r *Resolvers) getUserIDFromContext(p graphql.ResolveParams) (uint, error) {
	// This should be implemented based on your JWT middleware
	// For now, return 0 (no user)
	return 0, nil
}

// Helper function to check permissions
func (r *Resolvers) checkPermission(userID uint, module, action string) bool {
	// This should be implemented based on your permission system
	// For now, return true (allow all)
	return true
}
