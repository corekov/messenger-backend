package repository

import (
    "context"
    "fmt"
    "messenger/internal/models"

    "github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct{ db *pgxpool.Pool }

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(ctx context.Context, userID, deviceID, refreshToken, ip string, ttlSeconds int64) error {
    query := fmt.Sprintf(
        `INSERT INTO sessions (user_id, device_id, refresh_token, expires_at, ip_address)
         VALUES ($1, $2, $3, NOW() + interval '%d seconds', $4)`,
        ttlSeconds,
    )
    // inet не принимает "" — передаём nil если ip пустой
    var ipArg interface{}
    if ip != "" {
        ipArg = ip
    }
    _, err := r.db.Exec(ctx, query, userID, deviceID, refreshToken, ipArg)
    return err
}

func (r *SessionRepo) FindByToken(ctx context.Context, token string) (*models.Session, error) {
    var s models.Session
    err := r.db.QueryRow(ctx,
        `SELECT id, user_id, device_id, refresh_token, expires_at, ip_address::text, created_at
         FROM sessions WHERE refresh_token = $1 AND expires_at > NOW()`,
        token,
    ).Scan(&s.ID, &s.UserID, &s.DeviceID, &s.RefreshToken, &s.ExpiresAt, &s.IPAddress, &s.CreatedAt)
    if err != nil {
        return nil, err
    }
    return &s, nil
}

func (r *SessionRepo) Delete(ctx context.Context, token string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE refresh_token = $1`, token)
    return err
}

func (r *SessionRepo) DeleteAllForUser(ctx context.Context, userID string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
    return err
}