package handlers

import (
	"net/http"
	"time"

	"chat-be/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB        *gorm.DB
	JWTSecret string
}

type registerReq struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid body", "error": err.Error()})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	u := models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	if err := h.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed create user", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    u.ID,
		"name":  u.Name,
		"email": u.Email,
	})
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid body", "error": err.Error()})
		return
	}

	var u models.User
	if err := h.DB.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "wrong email/password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "wrong email/password"})
		return
	}

	claims := jwt.MapClaims{
		"user_id": u.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(h.JWTSecret))

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenStr,
		"user": gin.H{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
		},
	})
}
