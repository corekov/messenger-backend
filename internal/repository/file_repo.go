package repository

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FileRepo struct{ db *pgxpool.Pool }

func NewFileRepo(db *pgxpool.Pool) *FileRepo {
	return &FileRepo{db: db}
}

func (r *FileRepo) Create(ctx context.Context, file *models.File) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO files (uploader_id, storage_key, file_name, mime_type, size_bytes, encrypted_key)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		file.UploaderID, file.StorageKey, file.FileName, file.MimeType, file.SizeBytes, file.EncryptedKey,
	).Scan(&file.ID, &file.CreatedAt)
	return err
}

func (r *FileRepo) GetByID(ctx context.Context, id string) (*models.File, error) {
	var f models.File
	err := r.db.QueryRow(ctx,
		`SELECT id, uploader_id, storage_key, file_name, mime_type, size_bytes, encrypted_key, created_at
		 FROM files WHERE id = $1`, id,
	).Scan(&f.ID, &f.UploaderID, &f.StorageKey, &f.FileName, &f.MimeType, &f.SizeBytes, &f.EncryptedKey, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
