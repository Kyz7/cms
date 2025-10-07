package auth

import (
	"strings"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/response"
	"github.com/Kyz7/cms/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func JWTProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Missing authorization token",
				},
			})
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "INVALID_TOKEN_FORMAT",
					"message": "Invalid token format",
				},
			})
		}

		userID, err := utils.ParseJWT(tokenParts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired token",
				},
			})
		}

		c.Locals("user_id", userID)
		return c.Next()
	}
}

func RoleProtected(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)

		var u models.User
		if err := database.DB.Preload("Role").First(&u, userID).Error; err != nil {
			return response.Unauthorized(c, "User not found")
		}

		for _, role := range allowedRoles {
			if u.Role.Name == role {
				return c.Next()
			}
		}

		return response.Forbidden(c, "You don't have permission to access this resource")
	}
}
