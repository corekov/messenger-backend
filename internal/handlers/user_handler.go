package handlers

import (
	"net/http"

	"messenger/internal/middleware"
	"messenger/internal/models"
	"messenger/internal/repository"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo *repository.UserRepo
	keysRepo *repository.KeysRepo
}

func NewUserHandler(ur *repository.UserRepo, kr *repository.KeysRepo) *UserHandler {
	return &UserHandler{userRepo: ur, keysRepo: kr}
}

// GET /api/v1/users/search?q=username
func (h *UserHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if len(q) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query too short"})
		return
	}
	users, err := h.userRepo.Search(c.Request.Context(), q, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

// GET /api/v1/users/:id/keys
func (h *UserHandler) GetPublicKeys(c *gin.Context) {
	userID := c.Param("id")
	pk, err := h.keysRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "keys not found"})
		return
	}
	c.JSON(http.StatusOK, pk)
}

// POST /api/v1/users/keys
func (h *UserHandler) UploadKeys(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	deviceID := c.GetString(middleware.DeviceIDKey)

	var req models.UploadKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.keysRepo.Upsert(c.Request.Context(), deviceID, userID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
