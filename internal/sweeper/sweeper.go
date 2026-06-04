package sweeper

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"messenger/internal/models"
	"messenger/internal/repository"
	"messenger/internal/websocket"
)

type Sweeper struct {
	messageRepo *repository.MessageRepo
	chatRepo    *repository.ChatRepo
	hub         *ws.Hub
	ticker      *time.Ticker
	quit        chan struct{}
}

func NewSweeper(mr *repository.MessageRepo, cr *repository.ChatRepo, hub *ws.Hub) *Sweeper {
	return &Sweeper{
		messageRepo: mr,
		chatRepo:    cr,
		hub:         hub,
		quit:        make(chan struct{}),
	}
}

func (s *Sweeper) Start(interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.sweep()
			case <-s.quit:
				s.ticker.Stop()
				return
			}
		}
	}()
}

func (s *Sweeper) Stop() {
	close(s.quit)
}

func (s *Sweeper) sweep() {
	ctx := context.Background()
	
	expiredMsgs, err := s.messageRepo.DeleteExpired(ctx)
	if err != nil {
		log.Printf("[Sweeper] Error deleting expired messages: %v", err)
		return
	}

	if len(expiredMsgs) == 0 {
		return
	}

	log.Printf("[Sweeper] Deleted %d expired messages", len(expiredMsgs))

	// For each expired message, broadcast deletion
	for _, msg := range expiredMsgs {
		memberIDs, err := s.chatRepo.GetMembers(ctx, msg.ChatID)
		if err != nil {
			log.Printf("[Sweeper] Error getting chat members for chat %s: %v", msg.ChatID, err)
			continue
		}

		payload := models.DeleteMessagePayload{
			ChatID:    msg.ChatID,
			MessageID: msg.ID,
		}
		
		payloadBytes, _ := json.Marshal(payload)
		eventBytes, _ := json.Marshal(models.WSEvent{
			Type:    models.WSTypeMessageDelete,
			Payload: payloadBytes,
		})

		s.hub.SendToUsers(memberIDs, eventBytes)
	}
}
