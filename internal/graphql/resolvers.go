package graphql

import (
	"github.com/Kyz7/cms/internal/middleware"
	"github.com/graphql-go/graphql"
	"gorm.io/gorm"
)

type Resolvers struct {
	DB *gorm.DB
}

func NewResolvers(db *gorm.DB) *Resolvers {
	return &Resolvers{
		DB: db,
	}
}

func (r *Resolvers) CreateSchema() (graphql.Schema, error) {
	// Ensure Project types are initialized before creating schema
	if ProjectType == nil || ProjectMemberType == nil {
		initProjectTypes()
	}

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
			"projects": &graphql.Field{
				Type:    graphql.NewList(getProjectType()),
				Resolve: r.ProjectsResolver,
			},
			"project": &graphql.Field{
				Type: getProjectType(),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.ProjectResolver,
			},
			"projectMembers": &graphql.Field{
				Type: graphql.NewList(getProjectMemberType()),
				Args: graphql.FieldConfigArgument{
					"projectId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: r.ProjectMembersResolver,
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
			"contentRelations": &graphql.Field{
				Type: graphql.NewList(ContentRelationType),
				Args: graphql.FieldConfigArgument{
					"fromContentId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: r.ContentRelationsResolver,
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
			"mediaFolders": &graphql.Field{
				Type:    graphql.NewList(MediaFolderType),
				Resolve: r.MediaFoldersResolver,
			},
			"mediaStats": &graphql.Field{
				Type:    MediaStatsType,
				Resolve: r.MediaStatsResolver,
			},
			"searchEntries": &graphql.Field{
				Type: SearchResultType,
				Args: graphql.FieldConfigArgument{
					"query":          &graphql.ArgumentConfig{Type: graphql.String},
					"contentTypeIds": &graphql.ArgumentConfig{Type: graphql.NewList(graphql.Int)},
					"fields":         &graphql.ArgumentConfig{Type: graphql.NewList(graphql.String)},
					"status":         &graphql.ArgumentConfig{Type: graphql.String},
					"createdBy":      &graphql.ArgumentConfig{Type: graphql.Int},
					"tags":           &graphql.ArgumentConfig{Type: graphql.NewList(graphql.String)},
					"fromDate":       &graphql.ArgumentConfig{Type: graphql.String},
					"toDate":         &graphql.ArgumentConfig{Type: graphql.String},
					"page":           &graphql.ArgumentConfig{Type: graphql.Int},
					"limit":          &graphql.ArgumentConfig{Type: graphql.Int},
					"sortBy":         &graphql.ArgumentConfig{Type: graphql.String},
					"orderBy":        &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.SearchEntriesResolver,
			},
			"advancedSearch": &graphql.Field{
				Type: SearchResultType,
				Args: graphql.FieldConfigArgument{
					"query":          &graphql.ArgumentConfig{Type: graphql.String},
					"contentTypeIds": &graphql.ArgumentConfig{Type: graphql.NewList(graphql.Int)},
					"fields":         &graphql.ArgumentConfig{Type: graphql.NewList(graphql.String)},
					"status":         &graphql.ArgumentConfig{Type: graphql.String},
					"createdBy":      &graphql.ArgumentConfig{Type: graphql.Int},
					"tags":           &graphql.ArgumentConfig{Type: graphql.NewList(graphql.String)},
					"fromDate":       &graphql.ArgumentConfig{Type: graphql.String},
					"toDate":         &graphql.ArgumentConfig{Type: graphql.String},
					"page":           &graphql.ArgumentConfig{Type: graphql.Int},
					"limit":          &graphql.ArgumentConfig{Type: graphql.Int},
					"sortBy":         &graphql.ArgumentConfig{Type: graphql.String},
					"orderBy":        &graphql.ArgumentConfig{Type: graphql.String},
					"filters":        &graphql.ArgumentConfig{Type: JSONType},
				},
				Resolve: r.AdvancedSearchResolver,
			},
			// Workflow Queries
			"workflowHistory": &graphql.Field{
				Type: graphql.NewList(WorkflowHistoryType),
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: r.WorkflowHistoryResolver,
			},
			"workflowComments": &graphql.Field{
				Type: graphql.NewList(WorkflowCommentType),
				Args: graphql.FieldConfigArgument{
					"entryId":        &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"includePrivate": &graphql.ArgumentConfig{Type: graphql.Boolean},
				},
				Resolve: r.WorkflowCommentsResolver,
			},
			"workflowAssignments": &graphql.Field{
				Type: graphql.NewList(WorkflowAssignmentType),
				Args: graphql.FieldConfigArgument{
					"status": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.WorkflowAssignmentsResolver,
			},
			"workflowStats": &graphql.Field{
				Type: WorkflowStatsType,
				Args: graphql.FieldConfigArgument{
					"contentTypeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: r.WorkflowStatsResolver,
			},
			// SEO preview
			"seoPreview": &graphql.Field{
				Type: JSONType,
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: r.SEOPreviewResolver,
			},
			// Search facets & autocomplete
			"searchFacets": &graphql.Field{
				Type: JSONType,
				Args: graphql.FieldConfigArgument{
					"query":          &graphql.ArgumentConfig{Type: graphql.String},
					"contentTypeIds": &graphql.ArgumentConfig{Type: graphql.NewList(graphql.Int)},
				},
				Resolve: r.SearchFacetsResolver,
			},
			"autocomplete": &graphql.Field{
				Type: graphql.NewList(graphql.String),
				Args: graphql.FieldConfigArgument{
					"field":         &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"prefix":        &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"contentTypeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"limit":         &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: r.AutocompleteResolver,
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
			"duplicateRole": &graphql.Field{
				Type: RoleType,
				Args: graphql.FieldConfigArgument{
					"id":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"name": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.DuplicateRoleResolver,
			},
			"assignRoleToUser": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"userId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"roleId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: r.AssignRoleToUserResolver,
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
					"projectId": &graphql.ArgumentConfig{
						Type: graphql.Int,
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
			"translateContentEntry": &graphql.Field{
				Type: ContentEntryType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"targetLang": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"sourceLang": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"fields": &graphql.ArgumentConfig{
						Type: graphql.NewList(graphql.String),
					},
				},
				Resolve: r.TranslateContentEntryResolver,
			},
			"createContentRelation": &graphql.Field{
				Type: ContentRelationType,
				Args: graphql.FieldConfigArgument{
					"fromContentId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"toContentId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"relationType":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: r.CreateContentRelationResolver,
			},
			"deleteContentRelation": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: r.DeleteContentRelationResolver,
			},
			"createMediaFolder": &graphql.Field{
				Type: MediaFolderType,
				Args: graphql.FieldConfigArgument{
					"name":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"parentId": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: r.CreateMediaFolderResolver,
			},
			// Content Fields
			"addContentField": &graphql.Field{
				Type: ContentFieldType,
				Args: graphql.FieldConfigArgument{
					"contentTypeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"name":          &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"type":          &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"required":      &graphql.ArgumentConfig{Type: graphql.Boolean},
					"isSeo":         &graphql.ArgumentConfig{Type: graphql.Boolean},
					"unique":        &graphql.ArgumentConfig{Type: graphql.Boolean},
					"maxLength":     &graphql.ArgumentConfig{Type: graphql.Int},
					"minLength":     &graphql.ArgumentConfig{Type: graphql.Int},
					"pattern":       &graphql.ArgumentConfig{Type: graphql.String},
					"minValue":      &graphql.ArgumentConfig{Type: graphql.Float},
					"maxValue":      &graphql.ArgumentConfig{Type: graphql.Float},
					"defaultValue":  &graphql.ArgumentConfig{Type: graphql.String},
					"placeholder":   &graphql.ArgumentConfig{Type: graphql.String},
					"helpText":      &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.AddContentFieldResolver,
			},
			"updateContentField": &graphql.Field{
				Type: ContentFieldType,
				Args: graphql.FieldConfigArgument{
					"id":           &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"name":         &graphql.ArgumentConfig{Type: graphql.String},
					"type":         &graphql.ArgumentConfig{Type: graphql.String},
					"required":     &graphql.ArgumentConfig{Type: graphql.Boolean},
					"isSeo":        &graphql.ArgumentConfig{Type: graphql.Boolean},
					"unique":       &graphql.ArgumentConfig{Type: graphql.Boolean},
					"maxLength":    &graphql.ArgumentConfig{Type: graphql.Int},
					"minLength":    &graphql.ArgumentConfig{Type: graphql.Int},
					"pattern":      &graphql.ArgumentConfig{Type: graphql.String},
					"minValue":     &graphql.ArgumentConfig{Type: graphql.Float},
					"maxValue":     &graphql.ArgumentConfig{Type: graphql.Float},
					"defaultValue": &graphql.ArgumentConfig{Type: graphql.String},
					"placeholder":  &graphql.ArgumentConfig{Type: graphql.String},
					"helpText":     &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.UpdateContentFieldResolver,
			},
			"deleteContentField": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: r.DeleteContentFieldResolver,
			},
			// Workflow Mutations
			"changeContentStatus": &graphql.Field{
				Type: WorkflowHistoryType,
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"status":  &graphql.ArgumentConfig{Type: graphql.NewNonNull(WorkflowStatusEnum)},
					"comment": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.ChangeContentStatusResolver,
			},
			"requestReview": &graphql.Field{
				Type: WorkflowHistoryType,
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"comment": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.RequestReviewResolver,
			},
			"approveEntry": &graphql.Field{
				Type: WorkflowHistoryType,
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"comment": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.ApproveEntryResolver,
			},
			"rejectEntry": &graphql.Field{
				Type: WorkflowHistoryType,
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"comment": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.RejectEntryResolver,
			},
			"publishEntry": &graphql.Field{
				Type: WorkflowHistoryType,
				Args: graphql.FieldConfigArgument{
					"entryId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"comment": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: r.PublishEntryResolver,
			},
			"addWorkflowComment": &graphql.Field{
				Type: WorkflowCommentType,
				Args: graphql.FieldConfigArgument{
					"entryId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"comment":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"isPrivate": &graphql.ArgumentConfig{Type: graphql.Boolean},
				},
				Resolve: r.AddWorkflowCommentResolver,
			},
			"assignEntry": &graphql.Field{
				Type: WorkflowAssignmentType,
				Args: graphql.FieldConfigArgument{
					"entryId":    &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"assignedTo": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
					"dueDate":    &graphql.ArgumentConfig{Type: TimeType},
				},
				Resolve: r.AssignEntryResolver,
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})

	return schema, err
}

func (r *Resolvers) getUserIDFromContext(p graphql.ResolveParams) (uint, error) {
	if p.Context == nil {
		return 0, nil
	}
	if v := p.Context.Value(contextKeyUserID); v != nil {
		switch id := v.(type) {
		case uint:
			return id, nil
		case int:
			if id < 0 {
				return 0, nil
			}
			return uint(id), nil
		}
	}
	return 0, nil
}

func (r *Resolvers) checkPermission(userID uint, module, action string) bool {
	if userID == 0 {
		return false
	}
	return middleware.HasPermission(userID, module, action)
}
