//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseURL = "http://localhost:8080/api/v1"

type RegisterRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name"`
	DeviceFP   string `json:"device_fp"`
	Platform   string `json:"platform"`
}

type AuthResponse struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DeviceFP string `json:"device_fp"`
}

type CreateChatRequest struct {
	Type      string   `json:"type"`
	MemberIDs []string `json:"member_ids"`
}

func doRequest(t *testing.T, method, endpoint string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp, respBody
}

func TestE2EFlow(t *testing.T) {
	t.Parallel()

	// 1. Register User A
	userA := fmt.Sprintf("alice_%d", time.Now().UnixNano())
	fpA := "fp_alice_123"

	reqA := RegisterRequest{
		Username:   userA,
		Password:   "securepass123",
		DeviceName: "AlicePhone",
		DeviceFP:   fpA,
		Platform:   "ios",
	}

	resp, respBody := doRequest(t, http.MethodPost, "/auth/register", reqA, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode, string(respBody))

	var authA AuthResponse
	err := json.Unmarshal(respBody, &authA)
	require.NoError(t, err)
	require.NotEmpty(t, authA.AccessToken)
	require.NotEmpty(t, authA.RefreshToken)
	require.NotNil(t, authA.User)
	userID_A := authA.User["id"].(string)
	require.NotEmpty(t, userID_A)

	t.Run("GetMe", func(t *testing.T) {
		resp, respBody := doRequest(t, http.MethodGet, "/auth/me", nil, authA.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))

		var me map[string]interface{}
		err := json.Unmarshal(respBody, &me)
		require.NoError(t, err)
		assert.Equal(t, userID_A, me["id"])
		assert.Equal(t, userA, me["username"])
	})

	t.Run("DuplicateRegistration", func(t *testing.T) {
		resp, respBody := doRequest(t, http.MethodPost, "/auth/register", reqA, "")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, string(respBody), "username already taken")
	})

	// 2. Register User B
	userB := fmt.Sprintf("bob_%d", time.Now().UnixNano())
	fpB := "fp_bob_123"

	reqB := RegisterRequest{
		Username:   userB,
		Password:   "bobpass123",
		DeviceName: "BobPhone",
		DeviceFP:   fpB,
		Platform:   "android",
	}

	resp, respBody = doRequest(t, http.MethodPost, "/auth/register", reqB, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode, string(respBody))

	var authB AuthResponse
	err = json.Unmarshal(respBody, &authB)
	require.NoError(t, err)
	userID_B := authB.User["id"].(string)
	require.NotEmpty(t, userID_B)

	// 3. Login User A
	t.Run("Login", func(t *testing.T) {
		loginReq := LoginRequest{
			Username: userA,
			Password: "securepass123",
			DeviceFP: fpA,
		}
		resp, respBody := doRequest(t, http.MethodPost, "/auth/login", loginReq, "")
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
		var authLogin AuthResponse
		err := json.Unmarshal(respBody, &authLogin)
		require.NoError(t, err)
		assert.NotEmpty(t, authLogin.AccessToken)
	})

	// 4. Refresh Token
	t.Run("Refresh", func(t *testing.T) {
		refreshReq := map[string]string{"refresh_token": authA.RefreshToken}
		resp, respBody := doRequest(t, http.MethodPost, "/auth/refresh", refreshReq, "")
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
		var authRefresh AuthResponse
		err := json.Unmarshal(respBody, &authRefresh)
		require.NoError(t, err)
		assert.NotEmpty(t, authRefresh.AccessToken)
		authA.AccessToken = authRefresh.AccessToken // Use new access token
		authA.RefreshToken = authRefresh.RefreshToken
	})

	// 5. User Search
	t.Run("SearchUsers", func(t *testing.T) {
		resp, respBody := doRequest(t, http.MethodGet, "/users/search?q="+userB, nil, authA.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
		
		var users []map[string]interface{}
		err := json.Unmarshal(respBody, &users)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 1)
		
		found := false
		for _, u := range users {
			if u["id"] == userID_B {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find User B in search results")
	})

	// 6. Upload Keys and Get Keys
	t.Run("E2EEKeys", func(t *testing.T) {
		keysReq := map[string]interface{}{
			"identity_key":  "id_key_123",
			"signed_prekey": "sig_prekey_123",
			"prekey_sig":    "prekey_sig_123",
			"one_time_keys": []string{"otk_1", "otk_2"},
		}
		resp, respBody := doRequest(t, http.MethodPost, "/users/keys", keysReq, authA.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))

		resp, respBody = doRequest(t, http.MethodGet, "/users/"+userID_A+"/keys", nil, authB.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
		
		var keysResp map[string]interface{}
		err := json.Unmarshal(respBody, &keysResp)
		require.NoError(t, err)
		assert.Equal(t, "id_key_123", keysResp["identity_key"])
	})

	// 7. Chats and Messages
	var chatID string
	t.Run("CreateChat", func(t *testing.T) {
		chatReq := CreateChatRequest{
			Type:      "direct",
			MemberIDs: []string{userID_B},
		}
		resp, respBody := doRequest(t, http.MethodPost, "/chats", chatReq, authA.AccessToken)
		assert.Equal(t, http.StatusCreated, resp.StatusCode, string(respBody))

		var chatResp map[string]interface{}
		err := json.Unmarshal(respBody, &chatResp)
		require.NoError(t, err)
		chatID = chatResp["id"].(string)
		assert.NotEmpty(t, chatID)
	})

	t.Run("ListChats", func(t *testing.T) {
		resp, respBody := doRequest(t, http.MethodGet, "/chats", nil, authA.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))

		var chats []map[string]interface{}
		err := json.Unmarshal(respBody, &chats)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(chats), 1)
	})

	t.Run("GetMessages", func(t *testing.T) {
		resp, respBody := doRequest(t, http.MethodGet, "/chats/"+chatID+"/messages", nil, authA.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))

		var msgs []map[string]interface{}
		err := json.Unmarshal(respBody, &msgs)
		require.NoError(t, err)
		// Should be empty initially
		assert.Len(t, msgs, 0)
	})

	t.Run("MarkRead", func(t *testing.T) {
		resp, respBody := doRequest(t, http.MethodPost, "/chats/"+chatID+"/read", nil, authA.AccessToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
		var res map[string]interface{}
		err := json.Unmarshal(respBody, &res)
		require.NoError(t, err)
		assert.Equal(t, "ok", res["status"])
	})

	// 8. Logout
	t.Run("Logout", func(t *testing.T) {
		logoutReq := map[string]string{"refresh_token": authA.RefreshToken}
		resp, respBody := doRequest(t, http.MethodPost, "/auth/logout", logoutReq, "")
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(respBody))
		
		// Attempting to refresh should now fail
		resp, _ = doRequest(t, http.MethodPost, "/auth/refresh", logoutReq, "")
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})
}
