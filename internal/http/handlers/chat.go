package handlers

import (
	"net/http"
	"strconv"

	"chat-be/internal/models"
	"chat-be/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChatHandler struct {
	DB  *gorm.DB
	Hub *ws.Hub
}

type createDirectReq struct {
	OtherUserID uint `json:"other_user_id" binding:"required"`
}

func (h *ChatHandler) CreateDirectConversation(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req createDirectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid body", "error": err.Error()})
		return
	}

	// Simple: selalu buat conversation baru
	conv := models.Conversation{Type: "direct"}
	if err := h.DB.Create(&conv).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed create conversation", "error": err.Error()})
		return
	}

	parts := []models.ConversationParticipant{
		{ConversationID: conv.ID, UserID: userID},
		{ConversationID: conv.ID, UserID: req.OtherUserID},
	}
	if err := h.DB.Create(&parts).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed add participants", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"conversation_id": conv.ID})
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var parts []models.ConversationParticipant
	if err := h.DB.Where("user_id = ?", userID).Find(&parts).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed", "error": err.Error()})
		return
	}

	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		ids = append(ids, p.ConversationID)
	}

	var convs []models.Conversation
	if len(ids) > 0 {
		_ = h.DB.Where("id IN ?", ids).Order("updated_at desc").Find(&convs).Error
	}
	c.JSON(http.StatusOK, gin.H{"data": convs})
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	_ = userID // untuk produksi: validasi user adalah participant

	convID64, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	convID := uint(convID64)

	limit := 30
	if v := c.Query("limit"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 && x <= 200 {
			limit = x
		}
	}

	var msgs []models.Message
	q := h.DB.Where("conversation_id = ?", convID).Order("id desc").Limit(limit)

	if before := c.Query("before_id"); before != "" {
		if bid, err := strconv.ParseUint(before, 10, 64); err == nil {
			q = q.Where("id < ?", bid)
		}
	}

	if err := q.Find(&msgs).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed", "error": err.Error()})
		return
	}

	// Karena query desc, balikkan supaya tampil ascending di UI
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	c.JSON(http.StatusOK, gin.H{"data": msgs})
}

type sendMessageReq struct {
	Body string `json:"body" binding:"required"`
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	convID64, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	convID := uint(convID64)

	var req sendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid body", "error": err.Error()})
		return
	}

	msg := models.Message{
		ConversationID: convID,
		SenderID:       userID,
		Body:           req.Body,
	}

	if err := h.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed create message", "error": err.Error()})
		return
	}

	// cari participants untuk broadcast
	var parts []models.ConversationParticipant
	_ = h.DB.Where("conversation_id = ?", convID).Find(&parts).Error

	targetUserIDs := make([]uint, 0, len(parts))
	for _, p := range parts {
		targetUserIDs = append(targetUserIDs, p.UserID)
	}

	h.Hub.BroadcastToUsers(targetUserIDs, ws.Event{
		Type: "message:new",
		Data: msg,
	})

	c.JSON(http.StatusCreated, gin.H{"data": msg})
}
