package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"messenger/internal/models"
	"messenger/internal/repository"
	"messenger/internal/services"
	jwtpkg "messenger/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub        *Hub
	msgService *services.MessageService
	chatRepo   *repository.ChatRepo
	userRepo   *repository.UserRepo
	jwtMgr     *jwtpkg.Manager
}

func NewWSHandler(
	hub *Hub,
	ms *services.MessageService,
	cr *repository.ChatRepo,
	ur *repository.UserRepo,
	jwtMgr *jwtpkg.Manager,
) *WSHandler {
	return &WSHandler{hub: hub, msgService: ms, chatRepo: cr, userRepo: ur, jwtMgr: jwtMgr}
}

func (h *WSHandler) Handle(c *gin.Context) {
	// Токен через query param (Flutter WS не поддерживает кастомные заголовки)
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if token == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtMgr.ParseAccess(token)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := NewClient(claims.UserID, claims.DeviceID, h.hub, conn)
	h.hub.register <- client

	go h.broadcastPresence(claims.UserID, true)
	go client.WritePump()

	client.ReadPump(h.handleIncoming)

	go h.broadcastPresence(claims.UserID, false)
	h.userRepo.UpdateLastSeen(context.Background(), claims.UserID)
}

func (h *WSHandler) handleIncoming(userID string, raw []byte) {
	var event models.WSEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return
	}
	ctx := context.Background()

	switch event.Type {
	case models.WSTypeMessage:
		var req models.SendMessageRequest
		if err := mapPayload(event.Payload, &req); err != nil {
			log.Printf("invalid payload for message: %v", err)
			return
		}
		msg, err := h.msgService.Send(ctx, userID, &req)
		if err != nil {
			log.Printf("send message error: %v", err)
			return
		}
		memberIDs, err := h.chatRepo.GetMembers(ctx, req.ChatID)
		if err != nil {
			log.Printf("get members error: %v", err)
			return
		}
		payloadBytes, _ := json.Marshal(msg)
		eventBytes, _ := json.Marshal(models.WSEvent{Type: models.WSTypeMessage, Payload: payloadBytes})
		h.hub.SendToUsers(memberIDs, eventBytes)

	case models.WSTypeTyping:
		var p struct {
			ChatID string `json:"chat_id"`
		}
		if err := mapPayload(event.Payload, &p); err != nil {
			return
		}
		memberIDs, err := h.chatRepo.GetMembers(ctx, p.ChatID)
		if err != nil {
			return
		}
		payloadBytes, _ := json.Marshal(map[string]string{"chat_id": p.ChatID, "user_id": userID})
		eventBytes, _ := json.Marshal(models.WSEvent{Type: models.WSTypeTyping, Payload: payloadBytes})
		
		filtered := make([]string, 0, len(memberIDs))
		for _, id := range memberIDs {
			if id != userID {
				filtered = append(filtered, id)
			}
		}
		h.hub.SendToUsers(filtered, eventBytes)

	case models.WSTypeMessageRead:
		var p struct {
			ChatID string `json:"chat_id"`
		}
		if err := mapPayload(event.Payload, &p); err != nil {
			return
		}
		if err := h.msgService.MarkRead(ctx, p.ChatID, userID); err != nil {
			log.Printf("mark read error: %v", err)
			return
		}
		memberIDs, err := h.chatRepo.GetMembers(ctx, p.ChatID)
		if err != nil {
			return
		}
		payloadBytes, _ := json.Marshal(map[string]string{"chat_id": p.ChatID, "reader_id": userID})
		eventBytes, _ := json.Marshal(models.WSEvent{Type: models.WSTypeMessageRead, Payload: payloadBytes})
		h.hub.SendToUsers(memberIDs, eventBytes)

	// WebRTC signaling
	case models.WSTypeCallOffer, models.WSTypeCallAnswer,
		models.WSTypeCallICE, models.WSTypeCallEnd:
		var p struct {
			TargetUserID string      `json:"target_user_id"`
			Data         interface{} `json:"data"`
		}
		if err := mapPayload(event.Payload, &p); err != nil {
			return
		}
		payloadBytes, _ := json.Marshal(map[string]interface{}{"from": userID, "data": p.Data})
		eventBytes, _ := json.Marshal(models.WSEvent{Type: event.Type, Payload: payloadBytes})
		h.hub.SendToUsers([]string{p.TargetUserID}, eventBytes)
	}
}

func (h *WSHandler) broadcastPresence(userID string, online bool) {
	eventType := models.WSTypeOffline
	if online {
		eventType = models.WSTypeOnline
	}
	
	ctx := context.Background()
	mutualIDs, err := h.chatRepo.GetMutualContactIDs(ctx, userID)
	if err != nil {
		log.Printf("[presence] error fetching mutual contacts for user=%s: %v", userID, err)
		return
	}
	
	if len(mutualIDs) > 0 {
		payload := map[string]interface{}{
			"user_id": userID,
			"status":  eventType,
		}
		
		// If offline, we could theoretically send last_seen here, 
		// but since we update it in the database right after, the client can fetch it or just use "offline".
		
		payloadBytes, _ := json.Marshal(payload)
		eventBytes, _ := json.Marshal(models.WSEvent{
			Type:    eventType,
			Payload: payloadBytes,
		})
		
		h.hub.SendToUsers(mutualIDs, eventBytes)
	}

	log.Printf("[presence] user=%s online=%v type=%s (broadcasted to %d contacts)", userID, online, eventType, len(mutualIDs))
}

func mapPayload(payload json.RawMessage, dst interface{}) error {
	return json.Unmarshal(payload, dst)
}
