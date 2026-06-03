package services

import (
	"context"
	"fmt"
	"errors"
	"log"
	"messenger/internal/models"
	"messenger/internal/repository"
	jwtpkg "messenger/pkg/jwt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo    *repository.UserRepo
	deviceRepo  *repository.DeviceRepo
	sessionRepo *repository.SessionRepo
	jwtMgr      *jwtpkg.Manager
	refreshTTL  int64
}

func NewAuthService(
	userRepo *repository.UserRepo,
	deviceRepo *repository.DeviceRepo,
	sessionRepo *repository.SessionRepo,
	jwtMgr *jwtpkg.Manager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		deviceRepo:  deviceRepo,
		sessionRepo: sessionRepo,
		jwtMgr:      jwtMgr,
		refreshTTL:  int64(refreshTTL.Seconds()),
	}
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest, ip string) (*models.AuthResponse, error) {
	existing, _ := s.userRepo.FindByUsername(ctx, req.Username)
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Create(ctx, req.Username, string(hash))
	if err != nil {
		return nil, err
	}

	device, err := s.deviceRepo.Upsert(ctx, user.ID, req.DeviceName, req.DeviceFP, req.Platform)
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, device.ID, ip)
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest, ip string) (*models.AuthResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	device, err := s.deviceRepo.FindByFP(ctx, req.DeviceFP)
	if err != nil {
		// Device not found, register it automatically for this user
		device, err = s.deviceRepo.Upsert(ctx, user.ID, "Mobile Device", req.DeviceFP, "android")
		if err != nil {
			return nil, fmt.Errorf("failed to register device on login: %w", err)
		}
	} else if device.UserID != user.ID {
		// This device fingerprint belongs to another user
		device, err = s.deviceRepo.Upsert(ctx, user.ID, "Mobile Device", req.DeviceFP, "android")
		if err != nil {
			return nil, fmt.Errorf("failed to re-register device on login: %w", err)
		}
	}

	return s.issueTokens(ctx, user, device.ID, ip)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	claims, err := s.jwtMgr.ParseRefresh(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	session, err := s.sessionRepo.FindByToken(ctx, refreshToken)
	if err != nil {
		fmt.Printf("sessionRepo.FindByToken error: %v\n", err) // keeping this to replace it correctly
	}
	log.Printf("DEBUG REFRESH: err=%v, session=%+v, claims.UserID=%v", err, session, claims.UserID)
	if err != nil || session.UserID != claims.UserID {
		return nil, errors.New("session not found or expired")
	}

	s.sessionRepo.Delete(ctx, refreshToken)

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, claims.DeviceID, "")
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.sessionRepo.Delete(ctx, refreshToken)
}

func (s *AuthService) GetMe(ctx context.Context, userID string) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *AuthService) UpdateBio(ctx context.Context, userID, bio string) error {
	return s.userRepo.UpdateBio(ctx, userID, bio)
}

func (s *AuthService) UpdateAvatarURL(ctx context.Context, userID, avatarURL string) error {
	return s.userRepo.UpdateAvatarURL(ctx, userID, avatarURL)
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User, deviceID, ip string) (*models.AuthResponse, error) {
    access, err := s.jwtMgr.GenerateAccess(user.ID, deviceID)
    if err != nil {
        return nil, err
    }
    refresh, err := s.jwtMgr.GenerateRefresh(user.ID, deviceID)
    if err != nil {
        return nil, err
    }

    if err := s.sessionRepo.Create(ctx, user.ID, deviceID, refresh, ip, s.refreshTTL); err != nil {
        return nil, fmt.Errorf("failed to save session: %w", err) // ← было тихое падение
    }

    return &models.AuthResponse{
        AccessToken:  access,
        RefreshToken: refresh,
        User:         *user,
    }, nil
}
