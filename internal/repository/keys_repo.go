package repository

import (
	"context"
	"encoding/json"
	"messenger/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type KeysRepo struct{ db *pgxpool.Pool }

func NewKeysRepo(db *pgxpool.Pool) *KeysRepo { return &KeysRepo{db: db} }

func (r *KeysRepo) Upsert(ctx context.Context, deviceID, userID string, req *models.UploadKeysRequest) error {
	keysJSON, _ := json.Marshal(req.OneTimeKeys)
	_, err := r.db.Exec(ctx,
		`INSERT INTO public_keys (device_id, user_id, identity_key, signed_prekey, prekey_sig, one_time_keys)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (device_id) DO UPDATE
		   SET user_id = EXCLUDED.user_id,
		       identity_key = EXCLUDED.identity_key,
		       signed_prekey = EXCLUDED.signed_prekey,
		       prekey_sig = EXCLUDED.prekey_sig,
		       one_time_keys = EXCLUDED.one_time_keys,
		       uploaded_at = NOW()`,
		deviceID, userID, req.IdentityKey, req.SignedPrekey, req.PrekeySign, keysJSON,
	)
	return err
}

func (r *KeysRepo) GetByUserID(ctx context.Context, userID string) (*models.PublicKey, error) {
	var pk models.PublicKey
	var keysJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, device_id, user_id, identity_key, signed_prekey, prekey_sig, one_time_keys, uploaded_at
		 FROM public_keys WHERE user_id = $1
		 ORDER BY uploaded_at DESC LIMIT 1`,
		userID,
	).Scan(&pk.ID, &pk.DeviceID, &pk.UserID, &pk.IdentityKey, &pk.SignedPrekey, &pk.PrekeySign, &keysJSON, &pk.UploadedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(keysJSON, &pk.OneTimeKeys)
	return &pk, nil
}
