package repository

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepo struct{ db *pgxpool.Pool }

func NewMessageRepo(db *pgxpool.Pool) *MessageRepo { return &MessageRepo{db: db} }

func (r *MessageRepo) Create(ctx context.Context, m *models.Message) (*models.Message, error) {
	err := r.db.QueryRow(ctx,
		`INSERT INTO messages (chat_id, sender_id, ciphertext, iv, message_type, file_id, reply_to, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, chat_id, sender_id, ciphertext, iv, message_type, file_id, reply_to, status, created_at, expires_at, is_deleted`,
		m.ChatID, m.SenderID, m.Ciphertext, m.IV, m.MessageType, m.FileID, m.ReplyTo, m.ExpiresAt,
	).Scan(
		&m.ID, &m.ChatID, &m.SenderID, &m.Ciphertext, &m.IV, &m.MessageType,
		&m.FileID, &m.ReplyTo, &m.Status, &m.CreatedAt, &m.ExpiresAt, &m.IsDeleted,
	)
	return m, err
}

func (r *MessageRepo) ListByChatPaginated(ctx context.Context, chatID, beforeID string, limit int) ([]models.Message, error) {
	var rows pgx.Rows
	var err error

	if beforeID == "" {
		rows, err = r.db.Query(ctx,
			`SELECT id, chat_id, sender_id, ciphertext, iv, message_type, file_id, reply_to, status, created_at, expires_at, is_deleted
			 FROM messages
			 WHERE chat_id = $1 AND is_deleted = false
			   AND (expires_at IS NULL OR expires_at > NOW())
			 ORDER BY created_at DESC LIMIT $2`,
			chatID, limit,
		)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, chat_id, sender_id, ciphertext, iv, message_type, file_id, reply_to, status, created_at, expires_at, is_deleted
			 FROM messages
			 WHERE chat_id = $1 AND is_deleted = false
			   AND (expires_at IS NULL OR expires_at > NOW())
			   AND created_at < (SELECT created_at FROM messages WHERE id = $3)
			 ORDER BY created_at DESC LIMIT $2`,
			chatID, limit, beforeID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.Ciphertext, &m.IV,
			&m.MessageType, &m.FileID, &m.ReplyTo, &m.Status,
			&m.CreatedAt, &m.ExpiresAt, &m.IsDeleted); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *MessageRepo) UpdateStatus(ctx context.Context, messageID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE messages SET status=$1 WHERE id=$2`, status, messageID)
	return err
}

func (r *MessageRepo) UpdateStatusByChatAndUser(ctx context.Context, chatID, readerID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE messages SET status=$1 WHERE chat_id=$2 AND sender_id!=$3 AND status!='read'`,
		status, chatID, readerID)
	return err
}

func (r *MessageRepo) SoftDelete(ctx context.Context, messageID, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE messages SET is_deleted=true WHERE id=$1 AND sender_id=$2`,
		messageID, userID)
	return err
}

func (r *MessageRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx,
		`UPDATE messages SET is_deleted=true WHERE expires_at IS NOT NULL AND expires_at<=NOW()`)
	return err
}
