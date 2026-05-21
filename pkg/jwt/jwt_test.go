package jwtpkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTManager(t *testing.T) {
	manager := NewManager("access_secret", "refresh_secret", 15*time.Minute, 24*time.Hour)

	userID := "user-123"
	deviceID := "device-456"

	t.Run("Generate and Parse Access Token", func(t *testing.T) {
		token, err := manager.GenerateAccess(userID, deviceID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := manager.ParseAccess(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, deviceID, claims.DeviceID)
	})

	t.Run("Generate and Parse Refresh Token", func(t *testing.T) {
		token, err := manager.GenerateRefresh(userID, deviceID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := manager.ParseRefresh(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, deviceID, claims.DeviceID)
	})

	t.Run("Parse Access with Refresh Secret Should Fail", func(t *testing.T) {
		token, _ := manager.GenerateAccess(userID, deviceID)
		_, err := manager.ParseRefresh(token)
		assert.Error(t, err)
	})

	t.Run("Parse Expired Token", func(t *testing.T) {
		// Create a manager with 0 TTL
		expiredManager := NewManager("secret", "secret", -1*time.Minute, -1*time.Minute)
		token, _ := expiredManager.GenerateAccess(userID, deviceID)
		
		_, err := expiredManager.ParseAccess(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is expired")
	})
}
