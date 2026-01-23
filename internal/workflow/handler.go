package workflow

import (
	"time"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/project"
	"github.com/Kyz7/cms/internal/response"
	"github.com/gofiber/fiber/v2"
)

type ChangeStatusRequest struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

type AddCommentRequest struct {
	Comment   string `json:"comment"`
	IsPrivate bool   `json:"is_private"`
}

type AssignEntryRequest struct {
	AssignedTo     uint       `json:"assigned_to"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	AutoTransition bool       `json:"auto_transition_to_draft"`
}

func ChangeStatusHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Status == "" {
		return response.ValidationError(c, map[string]string{
			"status": "status is required",
		})
	}

	entry, err := ChangeWorkflowStatus(uint(entryID), userID, body.Status, body.Comment)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Success(c, entry, "Status changed successfully")
}

func RequestReviewHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		Comment string `json:"comment"`
	}
	c.BodyParser(&body)

	entry, err := RequestReview(uint(entryID), userID, body.Comment)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Success(c, entry, "Entry sent for review")
}

func ApproveEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		Comment string `json:"comment"`
	}
	c.BodyParser(&body)

	entry, err := ApproveEntry(uint(entryID), userID, body.Comment)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Success(c, entry, "Entry approved successfully")
}

func RejectEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		Comment string `json:"comment"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Comment == "" {
		return response.ValidationError(c, map[string]string{
			"comment": "comment is required when rejecting",
		})
	}

	entry, err := RejectEntry(uint(entryID), userID, body.Comment)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Success(c, entry, "Entry rejected")
}

func PublishEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		Comment string `json:"comment"`
	}
	c.BodyParser(&body)

	entry, err := PublishEntry(uint(entryID), userID, body.Comment)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Success(c, entry, "Entry published successfully")
}

func GetHistoryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	history, err := GetWorkflowHistory(uint(entryID))
	if err != nil {
		return response.InternalError(c, "Failed to fetch workflow history")
	}

	return response.Success(c, history, "Workflow history retrieved successfully")
}

func AddCommentHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		Comment   string `json:"comment"`
		IsPrivate bool   `json:"is_private"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.Comment == "" {
		return response.ValidationError(c, map[string]string{
			"comment": "comment is required",
		})
	}

	comment, err := AddWorkflowComment(uint(entryID), userID, body.Comment, body.IsPrivate)
	if err != nil {
		return response.InternalError(c, "Failed to add comment")
	}

	return response.Created(c, comment, "Comment added successfully")
}

func GetCommentsHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	includePrivate := c.Query("include_private") == "true"

	comments, err := GetWorkflowComments(uint(entryID), includePrivate)
	if err != nil {
		return response.InternalError(c, "Failed to fetch comments")
	}

	return response.Success(c, comments, "Comments retrieved successfully")
}

func AssignEntryHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		AssignedTo     uint       `json:"assigned_to"`
		DueDate        *time.Time `json:"due_date,omitempty"`
		AutoTransition bool       `json:"auto_transition_to_draft"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	if body.AssignedTo == 0 {
		return response.ValidationError(c, map[string]string{
			"assigned_to": "assigned_to is required",
		})
	}

	assignment, err := AssignEntry(uint(entryID), body.AssignedTo, userID, body.DueDate, body.AutoTransition)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Created(c, assignment, "Entry assigned successfully")
}

func GetMyAssignmentsHandler(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	status := c.Query("status")

	assignments, err := GetMyAssignments(userID, status)
	if err != nil {
		return response.InternalError(c, "Failed to fetch assignments")
	}

	return response.Success(c, assignments, "Assignments retrieved successfully")
}

func CompleteAssignmentHandler(c *fiber.Ctx) error {
	assignmentID, err := c.ParamsInt("assignment_id")
	if err != nil {
		return response.BadRequest(c, "Invalid assignment ID", nil)
	}

	userID := c.Locals("user_id").(uint)

	var body struct {
		RequestReview bool `json:"request_review"`
	}
	c.BodyParser(&body)

	if err := CompleteAssignment(uint(assignmentID), userID, body.RequestReview); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Success(c, nil, "Assignment completed successfully")
}

func GetEntriesByStatusHandler(c *fiber.Ctx) error {
	contentTypeID, err := c.ParamsInt("content_type_id")
	if err != nil {
		return response.BadRequest(c, "Invalid content type ID", nil)
	}

	status := c.Query("status")

	userID := c.Locals("user_id").(uint)

	var ct models.ContentType
	if err := database.DB.First(&ct, contentTypeID).Error; err != nil {
		return response.NotFound(c, "Content type")
	}

	qProj := c.QueryInt("project_id")
	var pID *uint
	if ct.ProjectID != nil && *ct.ProjectID > 0 {
		t := uint(*ct.ProjectID)
		pID = &t
	} else if qProj > 0 {
		t := uint(qProj)
		pID = &t
	}

	if pID != nil {
		if _, err := project.GetProjectMember(*pID, userID); err != nil {
			return response.Forbidden(c, "You are not a member of this project")
		}
	}

	entries, err := GetEntriesByStatus(uint(contentTypeID), status, pID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch entries")
	}

	return response.Success(c, entries, "Entries retrieved successfully")
}

func GetWorkflowStatsHandler(c *fiber.Ctx) error {
	contentTypeID, err := c.ParamsInt("content_type_id")
	if err != nil {
		return response.BadRequest(c, "Invalid content type ID", nil)
	}

	stats, err := GetWorkflowStatistics(uint(contentTypeID))
	if err != nil {
		return response.InternalError(c, "Failed to fetch statistics")
	}

	return response.Success(c, stats, "Workflow statistics retrieved successfully")
}

func GetActiveAssignmentHandler(c *fiber.Ctx) error {
	entryID, err := c.ParamsInt("entry_id")
	if err != nil {
		return response.BadRequest(c, "Invalid entry ID", nil)
	}

	assignment, err := GetActiveAssignment(uint(entryID))
	if err != nil {
		if err.Error() == "no active assignment found" {
			return response.Success(c, nil, "No active assignment")
		}
		return response.InternalError(c, "Failed to fetch assignment")
	}

	return response.Success(c, assignment, "Assignment retrieved successfully")
}

func GetAssigneesHandler(c *fiber.Ctx) error {
	// Optional project_id to align with permission context (already handled by middleware)
	var pID *uint
	if pid := c.QueryInt("project_id"); pid > 0 {
		pp := uint(pid)
		pID = &pp
	}

	assignees, err := GetAssigneeUsers(pID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch assignees")
	}

	return response.Success(c, assignees, "Assignees retrieved successfully")
}
