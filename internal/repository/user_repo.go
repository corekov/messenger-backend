package repository

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct{ db *pgxpool.Pool }

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(ctx context.Context, username, passwordHash string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, username, password_hash, avatar_url, bio, last_seen, is_active, created_at`,
		username, passwordHash,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Bio, &u.LastSeen, &u.IsActive, &u.CreatedAt)
	return &u, err
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`SELECT id, username, password_hash, avatar_url, bio, last_seen, is_active, created_at
		 FROM users WHERE username = $1 AND is_active = true`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Bio, &u.LastSeen, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`SELECT id, username, password_hash, avatar_url, bio, last_seen, is_active, created_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.AvatarURL, &u.Bio, &u.LastSeen, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) UpdateLastSeen(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET last_seen = NOW() WHERE id = $1`, userID)
	return err
}

func (r *UserRepo) Search(ctx context.Context, query string, limit int) ([]models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, username, avatar_url, bio, last_seen, is_active, created_at
		 FROM users
		 WHERE username ILIKE $1 AND is_active = true
		 LIMIT $2`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.AvatarURL, &u.Bio, &u.LastSeen, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
