package handlers

import (
	"net/http"
	"strconv"

	"messenger/internal/middleware"
	"messenger/internal/models"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService *services.ChatService
}

func NewChatHandler(s *services.ChatService) *ChatHandler {
	return &ChatHandler{chatService: s}
}

// GET /api/v1/chats
func (h *ChatHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	chats, err := h.chatService.ListChats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if chats == nil {
		chats = []models.Chat{}
	}
	c.JSON(http.StatusOK, chats)
}

// POST /api/v1/chats
func (h *ChatHandler) Create(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	var req models.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var chat *models.Chat
	var err error

	if req.Type == "direct" && len(req.MemberIDs) == 1 {
		chat, err = h.chatService.GetOrCreateDirect(c.Request.Context(), userID, req.MemberIDs[0])
	} else {
		chat, err = h.chatService.CreateGroup(c.Request.Context(), userID, &req)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, chat)
}

// GET /api/v1/chats/:id/messages
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	chatID := c.Param("id")
	beforeID := c.Query("before_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	msgs, err := h.chatService.GetMessages(c.Request.Context(), chatID, userID, beforeID, limit)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if msgs == nil {
		msgs = []models.Message{}
	}
	c.JSON(http.StatusOK, msgs)
}

// POST /api/v1/chats/:id/read
func (h *ChatHandler) MarkRead(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	chatID := c.Param("id")
	h.chatService.MarkRead(c.Request.Context(), chatID, userID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DELETE /api/v1/chats/:id
func (h *ChatHandler) Delete(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	chatID := c.Param("id")
	
	err := h.chatService.DeleteChat(c.Request.Context(), chatID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
