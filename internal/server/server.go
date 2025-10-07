package server

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func New(db *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024,
	})

	app.Static("/uploads", "./uploads", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		MaxAge:    3600,
	})

	// 	app.Use(csrf.New(csrf.Config{
	//     KeyLookup:      "header:X-CSRF-Token",
	//     CookieName:     "csrf_",
	//     CookieSameSite: "Strict",
	//     Expiration:     1 * time.Hour,
	// }))

	SetupRoutes(app)

	return app
}
