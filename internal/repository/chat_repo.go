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
	rows.Close() // Explicitly close rows to free connection for next queries

	for i := range chats {
		chat := &chats[i]

		// 1. Fetch Members and IdentityKeys
		mRows, err := r.db.Query(ctx,
			`SELECT u.id, u.username, u.avatar_url, 
			        (SELECT identity_key FROM public_keys WHERE user_id = u.id ORDER BY uploaded_at DESC LIMIT 1) as identity_key
			 FROM chat_members cm
			 JOIN users u ON cm.user_id = u.id
			 WHERE cm.chat_id = $1`, chat.ID)
		if err == nil {
			var members []models.User
			for mRows.Next() {
				var u models.User
				var idKey *string
				if err := mRows.Scan(&u.ID, &u.Username, &u.AvatarURL, &idKey); err == nil {
					u.IdentityKey = idKey
					members = append(members, u)
				}
			}
			mRows.Close()
			chat.Members = members
		}

		// 2. Fetch Last Message
		var lastMsg models.Message
		err = r.db.QueryRow(ctx,
			`SELECT id, chat_id, sender_id, ciphertext, iv, message_type, status, created_at 
			 FROM messages 
			 WHERE chat_id = $1 
			 ORDER BY created_at DESC LIMIT 1`, chat.ID).
			Scan(&lastMsg.ID, &lastMsg.ChatID, &lastMsg.SenderID, &lastMsg.Ciphertext, &lastMsg.IV, &lastMsg.MessageType, &lastMsg.Status, &lastMsg.CreatedAt)
		if err == nil {
			chat.LastMessage = &lastMsg
		}
		
		// 3. Fetch Unread Count
		var unread int
		err = r.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM messages 
			 WHERE chat_id = $1 AND sender_id != $2 AND status != 'read'`, chat.ID, userID).Scan(&unread)
		if err == nil {
			chat.UnreadCount = unread
		}
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

func (r *ChatRepo) RemoveMember(ctx context.Context, chatID, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM chat_members WHERE chat_id=$1 AND user_id=$2`, chatID, userID)
	return err
}

func (r *ChatRepo) GetMutualContactIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT user_id 
		FROM chat_members 
		WHERE chat_id IN (
			SELECT chat_id FROM chat_members WHERE user_id = $1
		) AND user_id != $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
