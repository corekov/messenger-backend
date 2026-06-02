package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"messenger/internal/models"

	"github.com/google/uuid"
)

const MaxFileSize = 50 * 1024 * 1024 // 50MB

var ErrFileTooLarge = errors.New("file size exceeds the 50MB limit")

type FileRepository interface {
	Create(ctx context.Context, file *models.File) error
	GetByID(ctx context.Context, id string) (*models.File, error)
}

type FileService struct {
	fileRepo FileRepository
	basePath string
}

func NewFileService(fileRepo FileRepository, basePath string) *FileService {
	// Ensure the base path exists
	os.MkdirAll(basePath, os.ModePerm)
	return &FileService{
		fileRepo: fileRepo,
		basePath: basePath,
	}
}

// UploadFile reads the file content, saves it to disk, and stores metadata in the DB.
func (s *FileService) UploadFile(ctx context.Context, uploaderID string, fileReader io.Reader, fileName, mimeType string, sizeBytes int64, encryptedKey string) (*models.File, error) {
	if sizeBytes > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	// Generate a storage key to prevent too many files in one directory
	// Format: YYYY/MM/DD/uuid
	now := time.Now()
	datePath := filepath.Join(fmt.Sprintf("%04d", now.Year()), fmt.Sprintf("%02d", now.Month()), fmt.Sprintf("%02d", now.Day()))
	fileUUID := uuid.New().String()
	storageKey := filepath.Join(datePath, fileUUID)

	fullPath := filepath.Join(s.basePath, storageKey)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	out, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file on disk: %w", err)
	}
	defer out.Close()

	// Use LimitReader to enforce size strictly during copy
	limitReader := io.LimitReader(fileReader, MaxFileSize+1)
	written, err := io.Copy(out, limitReader)
	if err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	if written > MaxFileSize {
		os.Remove(fullPath)
		return nil, ErrFileTooLarge
	}

	file := &models.File{
		UploaderID:   uploaderID,
		StorageKey:   storageKey,
		FileName:     fileName,
		MimeType:     mimeType,
		SizeBytes:    written,
		EncryptedKey: encryptedKey,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	// Build a dummy PresignedURL for the response
	file.PresignedURL = fmt.Sprintf("/api/v1/files/%s/download", file.ID)

	return file, nil
}

func (s *FileService) GetFileRecord(ctx context.Context, fileID string) (*models.File, error) {
	return s.fileRepo.GetByID(ctx, fileID)
}

func (s *FileService) GetFilePath(storageKey string) string {
	return filepath.Join(s.basePath, storageKey)
}
