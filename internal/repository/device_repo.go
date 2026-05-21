package repository

import (
	"context"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceRepo struct{ db *pgxpool.Pool }

func NewDeviceRepo(db *pgxpool.Pool) *DeviceRepo { return &DeviceRepo{db: db} }

func (r *DeviceRepo) Upsert(ctx context.Context, userID, name, fp, platform string) (*models.Device, error) {
	var d models.Device
	err := r.db.QueryRow(ctx,
		`INSERT INTO devices (user_id, device_name, device_fp, platform)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (device_fp) DO UPDATE
		   SET last_active = NOW(), user_id = EXCLUDED.user_id
		 RETURNING id, user_id, device_name, device_fp, push_token, platform, created_at, last_active`,
		userID, name, fp, platform,
	).Scan(&d.ID, &d.UserID, &d.DeviceName, &d.DeviceFP, &d.PushToken, &d.Platform, &d.CreatedAt, &d.LastActive)
	return &d, err
}

func (r *DeviceRepo) FindByFP(ctx context.Context, fp string) (*models.Device, error) {
	var d models.Device
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, device_name, device_fp, push_token, platform, created_at, last_active
		 FROM devices WHERE device_fp = $1`,
		fp,
	).Scan(&d.ID, &d.UserID, &d.DeviceName, &d.DeviceFP, &d.PushToken, &d.Platform, &d.CreatedAt, &d.LastActive)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeviceRepo) UpdatePushToken(ctx context.Context, deviceID, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE devices SET push_token = $1 WHERE id = $2`, token, deviceID)
	return err
}
