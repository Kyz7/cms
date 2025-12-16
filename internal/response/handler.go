package response

import (
	"github.com/gofiber/fiber/v2"
)

type StandardResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Data    interface{}  `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    *Meta        `json:"meta,omitempty"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int64 `json:"total_pages,omitempty"`
}

func Success(c *fiber.Ctx, data interface{}, message string) error {
	return c.JSON(StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(c *fiber.Ctx, data interface{}, meta *Meta, message string) error {
	return c.JSON(StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Created(c *fiber.Ctx, data interface{}, message string) error {
	return c.Status(fiber.StatusCreated).JSON(StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func Error(c *fiber.Ctx, statusCode int, errorCode string, message string, details interface{}) error {
	return c.Status(statusCode).JSON(StandardResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	})
}

func BadRequest(c *fiber.Ctx, message string, details interface{}) error {
	return Error(c, fiber.StatusBadRequest, "BAD_REQUEST", message, details)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

func Forbidden(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusForbidden, "FORBIDDEN", message, nil)
}

func NotFound(c *fiber.Ctx, resource string) error {
	return Error(c, fiber.StatusNotFound, "NOT_FOUND", resource+" not found", nil)
}

func Conflict(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusConflict, "CONFLICT", message, nil)
}

func ValidationError(c *fiber.Ctx, errors interface{}) error {
	return Error(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", errors)
}

func InternalError(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

func CalculateMeta(page, limit int, total int64) *Meta {
	totalPages := total / int64(limit)
	if total%int64(limit) > 0 {
		totalPages++
	}

	return &Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
