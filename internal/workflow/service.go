package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/project"
	"gorm.io/gorm"
)

func ChangeWorkflowStatus(entryID, userID uint, toStatus string, comment string) (*models.ContentEntry, error) {
	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return nil, fmt.Errorf("entry not found")
	}

	var user models.User
	if err := database.DB.Preload("Role").First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	targetStatus := models.WorkflowStatus(toStatus)

	// Determine user role: project role if entry belongs to project, otherwise global role
	userRole := user.Role.Name
	if entry.ProjectID != nil {
		// Entry belongs to a project, check project membership
		projectRole, err := project.GetUserProjectRole(*entry.ProjectID, userID)
		if err != nil {
			return nil, fmt.Errorf("you are not a member of this project")
		}
		// Map project role to workflow role
		userRole = mapProjectRoleToWorkflowRole(projectRole)
	}

	if !isValidTransition(entry.Status, targetStatus, userRole) {
		return nil, fmt.Errorf("invalid status transition from %s to %s for role %s",
			entry.Status, targetStatus, userRole)
	}

	if entry.Status == models.StatusInReview && targetStatus == models.StatusRejected {
		if userRole == "editor" {
			if strings.TrimSpace(comment) == "" {
				return nil, fmt.Errorf("comment is required when rejecting from In Review for role editor")
			}
		}
	}

	fromStatus := entry.Status
	entry.Status = targetStatus

	if targetStatus == models.StatusPublished {
		now := time.Now()
		entry.PublishedAt = &now
	}

	if err := database.DB.Save(&entry).Error; err != nil {
		return nil, err
	}

	history := models.WorkflowHistory{
		EntryID:    entryID,
		FromStatus: fromStatus,
		ToStatus:   targetStatus,
		ChangedBy:  userID,
		Comment:    comment,
	}
	if err := database.DB.Create(&history).Error; err != nil {
		return nil, err
	}

	return &entry, nil
}

// mapProjectRoleToWorkflowRole maps project roles to workflow roles
// owner/admin -> admin, editor -> editor, viewer -> viewer
func mapProjectRoleToWorkflowRole(projectRole string) string {
	switch projectRole {
	case models.ProjectRoleOwner, models.ProjectRoleAdmin:
		return "admin"
	case models.ProjectRoleEditor:
		return "editor"
	case models.ProjectRoleViewer:
		return "viewer"
	default:
		return "viewer"
	}
}

func isValidTransition(fromStatus, toStatus models.WorkflowStatus, userRole string) bool {
	transitions := map[models.WorkflowStatus]map[models.WorkflowStatus][]string{
		models.StatusDraft: {
			models.StatusInReview: {"content_writer", "editor", "admin"},
		},
		models.StatusInReview: {
			models.StatusReadyForApproval: {"editor", "admin"},
			models.StatusRejected:         {"editor", "admin"},
			models.StatusDraft:            {"editor", "admin"},
		},
		models.StatusReadyForApproval: {
			models.StatusApproved: {"manager", "admin"},
			models.StatusRejected: {"manager", "admin"},
		},
		models.StatusApproved: {
			models.StatusPublished: {"manager", "admin"},
		},
		models.StatusRejected: {
			models.StatusDraft: {"editor", "admin"},
		},
	}

	allowedRoles, exists := transitions[fromStatus][toStatus]
	if !exists {
		return false
	}

	for _, role := range allowedRoles {
		if role == userRole {
			return true
		}
	}
	return false
}

func GetWorkflowHistory(entryID uint) ([]models.WorkflowHistory, error) {
	var history []models.WorkflowHistory
	err := database.DB.
		Where("entry_id = ?", entryID).
		Preload("User").
		Order("created_at DESC").
		Find(&history).Error

	return history, err
}

func AddWorkflowComment(entryID, userID uint, comment string, isPrivate bool) (*models.WorkflowComment, error) {
	wfComment := models.WorkflowComment{
		EntryID:   entryID,
		UserID:    userID,
		Comment:   comment,
		IsPrivate: isPrivate,
	}

	if err := database.DB.Create(&wfComment).Error; err != nil {
		return nil, err
	}

	database.DB.Preload("User").First(&wfComment, wfComment.ID)
	return &wfComment, nil
}

func GetWorkflowComments(entryID uint, includePrivate bool) ([]models.WorkflowComment, error) {
	var comments []models.WorkflowComment
	query := database.DB.Where("entry_id = ?", entryID)

	if !includePrivate {
		query = query.Where("is_private = ?", false)
	}

	err := query.
		Preload("User").
		Order("created_at DESC").
		Find(&comments).Error

	return comments, err
}

func AssignEntry(entryID, assignedTo, assignedBy uint, dueDate *time.Time, autoTransitionToDraft bool) (*models.WorkflowAssignment, error) {
	// 1. Check entry status
	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return nil, fmt.Errorf("entry not found")
	}

	if entry.Status != models.StatusDraft && entry.Status != models.StatusRejected {
		return nil, fmt.Errorf("assignment can only be created when entry is in Draft or Rejected status (current: %s)", entry.Status)
	}

	// 1b. Prevent duplicate pending assignment for the same entry
	{
		var existing models.WorkflowAssignment
		if err := database.DB.Where("entry_id = ? AND status = ?", entryID, "pending").First(&existing).Error; err == nil {
			return nil, fmt.Errorf("an active assignment already exists for this entry")
		}
	}

	// 2. Check assignee role
	var assignee models.User
	if err := database.DB.Preload("Role").First(&assignee, assignedTo).Error; err != nil {
		return nil, fmt.Errorf("assignee user not found")
	}

	if assignee.Role == nil || assignee.Role.Name != "content_writer" {
		return nil, fmt.Errorf("assignment can only be directed to users with 'content_writer' role")
	}

	assignment := models.WorkflowAssignment{
		EntryID:    entryID,
		AssignedTo: assignedTo,
		AssignedBy: assignedBy,
		Status:     "pending",
		DueDate:    dueDate,
	}

	if err := database.DB.Create(&assignment).Error; err != nil {
		return nil, err
	}

	// Auto-transition Rejected -> Draft upon assignment
	if autoTransitionToDraft && entry.Status == models.StatusRejected {
		_, err := ChangeWorkflowStatus(entryID, assignedBy, string(models.StatusDraft), "Auto-transition to Draft upon assignment")
		if err != nil {
			// Fallback: update entry status directly if ChangeWorkflowStatus fails
			entry.Status = models.StatusDraft
			if err := database.DB.Save(&entry).Error; err != nil {
				return nil, fmt.Errorf("failed to update entry status: %v", err)
			}

			// Create workflow history entry
			history := models.WorkflowHistory{
				EntryID:    entryID,
				FromStatus: models.StatusRejected,
				ToStatus:   models.StatusDraft,
				ChangedBy:  assignedBy,
				Comment:    "Auto-transition to Draft upon assignment",
			}
			database.DB.Create(&history)
		}
	}

	database.DB.Preload("User").Preload("Assigner").First(&assignment, assignment.ID)
	return &assignment, nil
}

func GetMyAssignments(userID uint, status string) ([]models.WorkflowAssignment, error) {
	var assignments []models.WorkflowAssignment
	query := database.DB.Where("assigned_to = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.
		Preload("Entry").
		Preload("Assigner").
		Order("created_at DESC").
		Find(&assignments).Error

	return assignments, err
}

func GetEntriesByStatus(contentTypeID uint, status string, projectID *uint) ([]models.ContentEntry, error) {
	var entries []models.ContentEntry
	query := database.DB.Where("content_type_id = ?", contentTypeID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if projectID != nil && *projectID > 0 {
		query = query.Where("project_id = ?", *projectID)
	} else {
		query = query.Where("project_id IS NULL")
	}

	err := query.Order("created_at DESC").Find(&entries).Error
	return entries, err
}

func RequestReview(entryID, userID uint, comment string) (*models.ContentEntry, error) {
	var entry models.ContentEntry
	if err := database.DB.First(&entry, entryID).Error; err != nil {
		return nil, fmt.Errorf("entry not found")
	}

	if entry.Status != models.StatusDraft {
		return nil, fmt.Errorf("can only request review from draft status, current status: %s", entry.Status)
	}

	return ChangeWorkflowStatus(entryID, userID, string(models.StatusInReview), comment)
}

func ApproveEntry(entryID, userID uint, comment string) (*models.ContentEntry, error) {
	return ChangeWorkflowStatus(entryID, userID, string(models.StatusApproved), comment)
}

func RejectEntry(entryID, userID uint, comment string) (*models.ContentEntry, error) {
	return ChangeWorkflowStatus(entryID, userID, string(models.StatusRejected), comment)
}

func PublishEntry(entryID, userID uint, comment string) (*models.ContentEntry, error) {
	return ChangeWorkflowStatus(entryID, userID, string(models.StatusPublished), comment)
}

func GetWorkflowStatistics(contentTypeID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var total int64
	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ?", contentTypeID).
		Count(&total)

	var draft, inReview, readyForApproval, approved, published, rejected int64

	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND status = ?", contentTypeID, models.StatusDraft).
		Count(&draft)

	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND status = ?", contentTypeID, models.StatusInReview).
		Count(&inReview)

	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND status = ?", contentTypeID, models.StatusReadyForApproval).
		Count(&readyForApproval)

	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND status = ?", contentTypeID, models.StatusApproved).
		Count(&approved)

	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND status = ?", contentTypeID, models.StatusPublished).
		Count(&published)

	database.DB.Model(&models.ContentEntry{}).
		Where("content_type_id = ? AND status = ?", contentTypeID, models.StatusRejected).
		Count(&rejected)

	stats["total"] = total
	stats["draft"] = draft
	stats["in_review"] = inReview
	stats["ready_for_approval"] = readyForApproval
	stats["approved"] = approved
	stats["published"] = published
	stats["rejected"] = rejected

	return stats, nil
}

func GetActiveAssignment(entryID uint) (*models.WorkflowAssignment, error) {
	var assignment models.WorkflowAssignment
	err := database.DB.
		Where("entry_id = ? AND status = ?", entryID, "pending").
		Preload("User").
		Preload("Assigner").
		First(&assignment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no active assignment found")
		}
		return nil, err
	}

	return &assignment, nil
}

func GetContentWriterUsers() ([]models.User, error) {
	var users []models.User
	err := database.DB.
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "content_writer").
		Preload("Role").
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetAssigneeUsers returns users eligible to be assignees (content_writer).
// Matches old backend behavior: only content_writer role users are returned.
func GetAssigneeUsers(projectID *uint) ([]models.User, error) {
	var users []models.User
	query := database.DB.
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "content_writer").
		Preload("Role")

	// Future: if projectID != nil, we could filter users who are members of that project.
	// For now, we return global list of eligible roles.

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func CompleteAssignment(assignmentID, userID uint, requestReview bool) error {
	var assignment models.WorkflowAssignment
	if err := database.DB.Preload("Entry").First(&assignment, assignmentID).Error; err != nil {
		return fmt.Errorf("assignment not found")
	}

	if assignment.AssignedTo != userID {
		return fmt.Errorf("only the assigned user can complete this assignment")
	}

	if assignment.Status == "completed" {
		return fmt.Errorf("assignment is already completed")
	}

	assignment.Status = "completed"
	if err := database.DB.Save(&assignment).Error; err != nil {
		return fmt.Errorf("failed to complete assignment: %v", err)
	}

	if requestReview && assignment.Entry != nil && assignment.Entry.Status == models.StatusDraft {
		_, err := RequestReview(assignment.EntryID, userID, "Auto-requested review upon assignment completion")
		if err != nil {
			return fmt.Errorf("assignment completed but failed to request review: %v", err)
		}
	}

	return nil
}
