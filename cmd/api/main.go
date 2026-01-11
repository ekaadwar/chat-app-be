package main

import (
	"fmt"
	"log"

	"chat-be/internal/config"
	"chat-be/internal/database"
	"chat-be/internal/http/handlers"
	"chat-be/internal/http/middleware"
	"chat-be/internal/models"
	"chat-be/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	if cfg.DBDSN == "" || cfg.JWTSecret == "" {
		log.Fatal("DB_DSN dan JWT_SECRET wajib diisi di .env")
	}

	db, err := database.ConnectMySQL(cfg.DBDSN)
	if err != nil {
		log.Fatal("failed connect db:", err)
	}

	// Auto-migrate tables
	if err := db.AutoMigrate(
		&models.User{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.Message{},
	); err != nil {
		log.Fatal("failed migrate:", err)
	}

	hub := ws.NewHub()

	r := gin.Default()

	// Auth
	authH := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret}
	r.POST("/api/v1/auth/register", authH.Register)
	r.POST("/api/v1/auth/login", authH.Login)

	// WebSocket endpoint
	wsH := &handlers.WSHandler{
		Hub:                  hub,
		JWTSecret:            cfg.JWTSecret,
		WSInsecureSkipVerify: cfg.WSInsecureSkipVerify,
	}
	r.GET("/ws", wsH.Handle)

	// Protected routes
	authed := r.Group("/api/v1")
	authed.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	chatH := &handlers.ChatHandler{DB: db, Hub: hub}
	authed.POST("/conversations/direct", chatH.CreateDirectConversation)
	authed.GET("/conversations", chatH.ListConversations)
	authed.GET("/conversations/:id/messages", chatH.ListMessages)
	authed.POST("/conversations/:id/messages", chatH.SendMessage)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Println("listening on", addr)
	log.Fatal(r.Run(addr))
}
