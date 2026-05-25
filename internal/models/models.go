package models

import (
	"encoding/json"
	"time"
)

// ─── User ──────────────────────────────────────────────────────────────────────
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	Bio          *string    `json:"bio,omitempty"`
	LastSeen     *time.Time `json:"last_seen,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	IdentityKey  *string    `json:"identity_key,omitempty"` // populated via joins
}

// ─── Device ────────────────────────────────────────────────────────────────────
type Device struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	DeviceName string    `json:"device_name"`
	DeviceFP   string    `json:"device_fp"`
	PushToken  *string   `json:"push_token,omitempty"`
	Platform   string    `json:"platform"`
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
}

// ─── PublicKey ─────────────────────────────────────────────────────────────────
type PublicKey struct {
	ID           string    `json:"id"`
	DeviceID     string    `json:"device_id"`
	UserID       string    `json:"user_id"`
	IdentityKey  string    `json:"identity_key"`
	SignedPrekey string    `json:"signed_prekey"`
	PrekeySign   string    `json:"prekey_sig"`
	OneTimeKeys  []string  `json:"one_time_keys"`
	UploadedAt   time.Time `json:"uploaded_at"`
}

// ─── Session ───────────────────────────────────────────────────────────────────
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	IPAddress    *string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── Chat ─────────────────────────────────────────────────────────────────────
type Chat struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // direct | group
	Name      *string   `json:"name,omitempty"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	// virtual fields (populated via JOIN)
	LastMessage *Message  `json:"last_message,omitempty"`
	UnreadCount int       `json:"unread_count"`
	Members     []User    `json:"members,omitempty"`
}

// ─── Message ──────────────────────────────────────────────────────────────────
type Message struct {
	ID          string     `json:"id"`
	ChatID      string     `json:"chat_id"`
	SenderID    string     `json:"sender_id"`
	Ciphertext  string     `json:"ciphertext"`
	IV          string     `json:"iv"`
	MessageType string     `json:"message_type"` // text|file|image|voice
	FileID      *string    `json:"file_id,omitempty"`
	ReplyTo     *string    `json:"reply_to,omitempty"`
	Status      string     `json:"status"` // sent|delivered|read
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsDeleted   bool       `json:"is_deleted"`
}

// ─── File ─────────────────────────────────────────────────────────────────────
type File struct {
	ID           string    `json:"id"`
	UploaderID   string    `json:"uploader_id"`
	StorageKey   string    `json:"-"`
	FileName     string    `json:"file_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	EncryptedKey string    `json:"encrypted_key"`
	CreatedAt    time.Time `json:"created_at"`
	// used only for responses
	PresignedURL string `json:"url,omitempty"`
}

// ─── DTO ──────────────────────────────────────────────────────────────────────
type RegisterRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=32"`
	Password   string `json:"password" binding:"required,min=8"`
	DeviceName string `json:"device_name" binding:"required"`
	DeviceFP   string `json:"device_fp" binding:"required"`
	Platform   string `json:"platform" binding:"required,oneof=ios android"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	DeviceFP string `json:"device_fp" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type SendMessageRequest struct {
	ChatID      string  `json:"chat_id" binding:"required"`
	Ciphertext  string  `json:"ciphertext" binding:"required"`
	IV          string  `json:"iv" binding:"required"`
	MessageType string  `json:"message_type"`
	FileID      *string `json:"file_id,omitempty"`
	ReplyTo     *string `json:"reply_to,omitempty"`
}

type CreateChatRequest struct {
	Type      string   `json:"type" binding:"required,oneof=direct group"`
	Name      *string  `json:"name,omitempty"`
	MemberIDs []string `json:"member_ids" binding:"required,min=1"`
}

type UploadKeysRequest struct {
	IdentityKey  string   `json:"identity_key" binding:"required"`
	SignedPrekey string   `json:"signed_prekey" binding:"required"`
	PrekeySign   string   `json:"prekey_sig" binding:"required"`
	OneTimeKeys  []string `json:"one_time_keys"`
}

// ─── WebSocket Events ─────────────────────────────────────────────────────────
type WSEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	WSTypeMessage         = "message"
	WSTypeMessageAck      = "message_ack"
	WSTypeMessageRead     = "message_read"
	WSTypeTyping          = "typing"
	WSTypeOnline          = "online"
	WSTypeOffline         = "offline"
	WSTypeCallOffer       = "call_offer"
	WSTypeCallAnswer      = "call_answer"
	WSTypeCallICE         = "call_ice"
	WSTypeCallEnd         = "call_end"
)
