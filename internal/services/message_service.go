package services

import (
	"context"
	"errors"
	"time"

	"messenger/internal/models"
	"messenger/internal/repository"
)

type MessageService struct {
	messageRepo *repository.MessageRepo
	chatRepo    *repository.ChatRepo
}

func NewMessageService(mr *repository.MessageRepo, cr *repository.ChatRepo) *MessageService {
	return &MessageService{messageRepo: mr, chatRepo: cr}
}

func (s *MessageService) Send(ctx context.Context, senderID string, req *models.SendMessageRequest) (*models.Message, error) {
	ok, err := s.chatRepo.IsMember(ctx, req.ChatID, senderID)
	if err != nil || !ok {
		return nil, errors.New("not a member of this chat")
	}

	chat, err := s.chatRepo.FindByID(ctx, req.ChatID)
	if err != nil {
		return nil, errors.New("chat not found")
	}

	msgType := req.MessageType
	if msgType == "" {
		msgType = "text"
	}

	msg := &models.Message{
		ChatID:      req.ChatID,
		SenderID:    senderID,
		Ciphertext:  req.Ciphertext,
		IV:          req.IV,
		MessageType: msgType,
		FileID:      req.FileID,
		ReplyTo:     req.ReplyTo,
		Status:      "sent",
	}

	if chat.IsSecret && chat.MessageTTL != nil {
		importTime := time.Now().Add(time.Duration(*chat.MessageTTL) * time.Second)
		msg.ExpiresAt = &importTime
	}

	return s.messageRepo.Create(ctx, msg)
}

func (s *MessageService) Delete(ctx context.Context, messageID, userID string) error {
	return s.messageRepo.SoftDelete(ctx, messageID, userID)
}

func (s *MessageService) MarkRead(ctx context.Context, chatID, userID string) error {
	return s.messageRepo.UpdateStatusByChatAndUser(ctx, chatID, userID, "read")
}
