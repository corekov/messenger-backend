package repository

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepo struct{ db *pgxpool.Pool }

func NewChatRepo(db *pgxpool.Pool) *ChatRepo { return &ChatRepo{db: db} }

func (r *ChatRepo) Create(ctx context.Context, chatType, createdBy string, name *string, memberIDs []string) (*models.Chat, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var chat models.Chat
	err = tx.QueryRow(ctx,
		`INSERT INTO chats (type, name, created_by) VALUES ($1, $2, $3)
		 RETURNING id, type, name, created_by, created_at`,
		chatType, name, createdBy,
	).Scan(&chat.ID, &chat.Type, &chat.Name, &chat.CreatedBy, &chat.CreatedAt)
	if err != nil {
		return nil, err
	}

	// добавляем создателя
	allMembers := append([]string{createdBy}, memberIDs...)
	for _, uid := range allMembers {
		role := "member"
		if uid == createdBy {
			role = "admin"
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO chat_members (chat_id, user_id, role) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`, chat.ID, uid, role)
		if err != nil {
			return nil, err
		}
	}

	return &chat, tx.Commit(ctx)
}

func (r *ChatRepo) FindDirectChat(ctx context.Context, userA, userB string) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.QueryRow(ctx,
		`SELECT c.id, c.type, c.name, c.created_by, c.created_at
		 FROM chats c
		 JOIN chat_members cm1 ON c.id = cm1.chat_id AND cm1.user_id = $1
		 JOIN chat_members cm2 ON c.id = cm2.chat_id AND cm2.user_id = $2
		 WHERE c.type = 'direct'
		 LIMIT 1`,
		userA, userB,
	).Scan(&chat.ID, &chat.Type, &chat.Name, &chat.CreatedBy, &chat.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *ChatRepo) ListByUser(ctx context.Context, userID string) ([]models.Chat, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT c.id, c.type, c.name, c.created_by, c.created_at
		 FROM chats c
		 JOIN chat_members cm ON c.id = cm.chat_id
		 WHERE cm.user_id = $1
		 ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chats []models.Chat
	for rows.Next() {
		var c models.Chat
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
}

func (r *ChatRepo) GetMembers(ctx context.Context, chatID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id FROM chat_members WHERE chat_id = $1`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *ChatRepo) IsMember(ctx context.Context, chatID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM chat_members WHERE chat_id=$1 AND user_id=$2)`,
		chatID, userID,
	).Scan(&exists)
	return exists, err
}
