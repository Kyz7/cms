package auth

import (
	"fmt"

	"github.com/Kyz7/cms/internal/database"
	"github.com/Kyz7/cms/internal/models"
	"github.com/Kyz7/cms/internal/utils"
)

func RegisterUser(name, email, password string) (*models.User, error) {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := models.User{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		Provider: "local",
	}

	if err := database.DB.Create(&u).Error; err != nil {
		return nil, err
	}

	return &u, nil
}

func LoginUser(email, password string) (string, string, error) {
	var user models.User
	if err := database.DB.Preload("Role").Where("email = ?", email).First(&user).Error; err != nil {
		return "", "", err
	}

	if !utils.CheckPasswordHash(password, user.Password) {
		return "", "", fmt.Errorf("invalid credentials")
	}

	accessToken, err := utils.GenerateJWT(user.ID, user.Role.Name)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
