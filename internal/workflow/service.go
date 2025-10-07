package workflow

import (
	"fmt"
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
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

	if !isValidTransition(entry.Status, targetStatus, user.Role.Name) {
		return nil, fmt.Errorf("invalid status transition from %s to %s for role %s",
			entry.Status, targetStatus, user.Role.Name)
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

func isValidTransition(fromStatus, toStatus models.WorkflowStatus, userRole string) bool {
	transitions := map[models.WorkflowStatus]map[models.WorkflowStatus][]string{
		models.StatusDraft: {
			models.StatusInReview: {"editor", "admin"},
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

func AssignEntry(entryID, assignedTo, assignedBy uint, dueDate *time.Time) (*models.WorkflowAssignment, error) {
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

func CompleteAssignment(assignmentID uint) error {
	return database.DB.Model(&models.WorkflowAssignment{}).
		Where("id = ?", assignmentID).
		Update("status", "completed").Error
}

func GetEntriesByStatus(contentTypeID uint, status string) ([]models.ContentEntry, error) {
	var entries []models.ContentEntry
	query := database.DB.Where("content_type_id = ?", contentTypeID)

	if status != "" {
		query = query.Where("status = ?", status)
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
