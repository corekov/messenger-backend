package services

import (
	"context"
	"errors"
	"messenger/internal/models"
	"messenger/internal/repository"
)

type ChatService struct {
	chatRepo    *repository.ChatRepo
	messageRepo *repository.MessageRepo
}

func NewChatService(chatRepo *repository.ChatRepo, messageRepo *repository.MessageRepo) *ChatService {
	return &ChatService{chatRepo: chatRepo, messageRepo: messageRepo}
}

func (s *ChatService) GetOrCreateDirect(ctx context.Context, userID, targetUserID string) (*models.Chat, error) {
	chat, err := s.chatRepo.FindDirectChat(ctx, userID, targetUserID)
	if err == nil {
		return chat, nil
	}
	return s.chatRepo.Create(ctx, "direct", userID, nil, []string{targetUserID})
}

func (s *ChatService) CreateGroup(ctx context.Context, userID string, req *models.CreateChatRequest) (*models.Chat, error) {
	if req.Type == "group" && (req.Name == nil || *req.Name == "") {
		return nil, errors.New("group name is required")
	}
	return s.chatRepo.Create(ctx, req.Type, userID, req.Name, req.MemberIDs)
}

func (s *ChatService) ListChats(ctx context.Context, userID string) ([]models.Chat, error) {
	return s.chatRepo.ListByUser(ctx, userID)
}

func (s *ChatService) GetMessages(ctx context.Context, chatID, userID, beforeID string, limit int) ([]models.Message, error) {
	ok, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil || !ok {
		return nil, errors.New("access denied")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.messageRepo.ListByChatPaginated(ctx, chatID, beforeID, limit)
}

func (s *ChatService) MarkRead(ctx context.Context, chatID, userID string) error {
	return s.messageRepo.UpdateStatusByChatAndUser(ctx, chatID, userID, "read")
}

func (s *ChatService) DeleteChat(ctx context.Context, chatID, userID string) error {
	// First check if the user is a member of the chat
	ok, err := s.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil || !ok {
		return errors.New("access denied")
	}
	return s.chatRepo.RemoveMember(ctx, chatID, userID)
}
