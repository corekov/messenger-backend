package sweeper_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"messenger/internal/models"
	"messenger/internal/repository"
	"messenger/internal/sweeper"
	"messenger/internal/websocket"
)

// TestSweeper_Integration tests the background job for deleting expired messages.
// It requires a test database connection to run.
func TestSweeper_Integration(t *testing.T) {
	// Note: This test is designed to run with a test database.
	// In a real environment, you would use a connection pool (pgxpool) connected to a test DB.
	t.Skip("Skipping integration test in unit test suite. Requires active PostgreSQL connection.")

	// 1. Setup mock/test database connection here...
	// In a real environment, you would use a connection pool (pgxpool) connected to a test DB.

	// 2. Initialize repositories
	msgRepo := repository.NewMessageRepo(nil) // Requires a valid pgxpool.Pool in real scenario
	chatRepo := repository.NewChatRepo(nil)   // Requires a valid pgxpool.Pool in real scenario
	hub := ws.NewHub()

	// 3. Initialize Sweeper
	sw := sweeper.NewSweeper(msgRepo, chatRepo, hub)

	// 4. Create a test chat with message_ttl = 1 (1 second)
	ctx := context.Background()
	chatTTL := 1
	chat, err := chatRepo.Create(ctx, "direct", "user_1", nil, []string{"user_2"}, true, &chatTTL)
	require.NoError(t, err)

	// 5. Insert a message into the chat
	msgReq := &models.Message{
		ChatID:      chat.ID,
		SenderID:    "user_1",
		Ciphertext:  "ciphertext",
		IV:          "iv",
		MessageType: "text",
	}
	msg, err := msgRepo.Create(ctx, msgReq)
	require.NoError(t, err)
	_ = msg // Suppress unused warning

	// Verify message exists
	msgs, err := msgRepo.ListByChatPaginated(ctx, chat.ID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)

	// 6. Start the sweeper with a short interval
	sw.Start(500 * time.Millisecond)
	defer sw.Stop()

	// Wait for TTL to expire and sweeper to run
	time.Sleep(2 * time.Second)

	// 7. Verify message is deleted
	msgsAfter, err := msgRepo.ListByChatPaginated(ctx, chat.ID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgsAfter, 0, "Message should be deleted by sweeper")
}
