package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	userID := "test-user-1"
	client := &Client{
		UserID: userID,
		Send:   make(chan []byte, 10),
	}

	t.Run("Register Client", func(t *testing.T) {
		hub.register <- client
		// wait a bit for processing
		time.Sleep(50 * time.Millisecond)

		assert.True(t, hub.IsOnline(userID))
	})

	t.Run("SendToUsers", func(t *testing.T) {
		payload := []byte("test-payload")
		hub.SendToUsers([]string{userID}, payload)
		
		select {
		case msg := <-client.Send:
			assert.Equal(t, payload, msg)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for broadcast")
		}
	})

	t.Run("Unregister Client", func(t *testing.T) {
		hub.unregister <- client
		time.Sleep(50 * time.Millisecond)

		assert.False(t, hub.IsOnline(userID))

		// Send channel should be closed
		_, ok := <-client.Send
		assert.False(t, ok)
	})
}
