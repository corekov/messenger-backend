package handlers

import (
	"errors"
	"net/http"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	fileService *services.FileService
}

func NewFileHandler(fileService *services.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

func (h *FileHandler) UploadFile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Read encrypted_key
	encryptedKey := c.PostForm("encrypted_key")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	if fileHeader.Size > services.MaxFileSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file size exceeds the 50MB limit"})
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

	file, err := h.fileService.UploadFile(c.Request.Context(), userID.(string), fileContent, fileName, mimeType, fileHeader.Size, encryptedKey)
	if err != nil {
		if errors.Is(err, services.ErrFileTooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		}
		return
	}

	c.JSON(http.StatusOK, file)
}

func (h *FileHandler) DownloadFile(c *gin.Context) {
	fileID := c.Param("id")

	fileRecord, err := h.fileService.GetFileRecord(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	filePath := h.fileService.GetFilePath(fileRecord.StorageKey)
	
	// We could also set headers like Content-Disposition or Content-Type
	if fileRecord.MimeType != "" {
		c.Header("Content-Type", fileRecord.MimeType)
	} else {
		c.Header("Content-Type", "application/octet-stream")
	}

	c.File(filePath)
}
