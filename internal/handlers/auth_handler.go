package handlers

import (
	"errors"
	"net/http"

	"messenger/internal/middleware"
	"messenger/internal/models"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
	fileService *services.FileService
}

func NewAuthHandler(authService *services.AuthService, fileService *services.FileService) *AuthHandler {
	return &AuthHandler{authService: authService, fileService: fileService}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.authService.Register(c.Request.Context(), &req, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.authService.Login(c.Request.Context(), &req, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.authService.Logout(c.Request.Context(), req.RefreshToken)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	user, err := h.authService.GetMe(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// PUT /api/v1/auth/me/bio
func (h *AuthHandler) UpdateBio(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	var req struct {
		Bio string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.authService.UpdateBio(c.Request.Context(), userID, req.Bio); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update bio"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /api/v1/auth/me/avatar
func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	if fileHeader.Size > services.MaxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file size exceeds limit"})
		return
	}

	fileContent, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer fileContent.Close()

	fileName := fileHeader.Filename
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Avatars don't need encrypted keys since they are public. Pass empty string.
	file, err := h.fileService.UploadFile(c.Request.Context(), userID, fileContent, fileName, mimeType, fileHeader.Size, "")
	if err != nil {
		if errors.Is(err, services.ErrFileTooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload avatar"})
		}
		return
	}

	avatarURL := file.PresignedURL
	if err := h.authService.UpdateAvatarURL(c.Request.Context(), userID, avatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update avatar in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"avatar_url": avatarURL,
	})
}
