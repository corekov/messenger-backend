package main

import (
	"log"

	"messenger/internal/config"
	"messenger/internal/database"
	"messenger/internal/handlers"
	"messenger/internal/middleware"
	"messenger/internal/repository"
	"messenger/internal/services"
	ws "messenger/internal/websocket"
	jwtpkg "messenger/pkg/jwt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db := database.NewPostgres(cfg.DbDSN)
	rdb := database.NewRedis(cfg.RedisAddr, cfg.RedisPassword)
	_ = rdb

	jwtMgr := jwtpkg.NewManager(
		cfg.JWTAccessSecret,
		cfg.JWTRefreshSecret,
		cfg.JWTAccessTTL,
		cfg.JWTRefreshTTL,
	)

	userRepo    := repository.NewUserRepo(db)
	deviceRepo  := repository.NewDeviceRepo(db)
	sessionRepo := repository.NewSessionRepo(db)
	chatRepo    := repository.NewChatRepo(db)
	messageRepo := repository.NewMessageRepo(db)
	keysRepo    := repository.NewKeysRepo(db)
	fileRepo    := repository.NewFileRepo(db)

	authService    := services.NewAuthService(userRepo, deviceRepo, sessionRepo, jwtMgr, cfg.JWTRefreshTTL)
	chatService    := services.NewChatService(chatRepo, messageRepo)
	messageService := services.NewMessageService(messageRepo, chatRepo)
	fileService    := services.NewFileService(fileRepo, "./uploads")

	hub := ws.NewHub()
	go hub.Run()

	authHandler := handlers.NewAuthHandler(authService, fileService)
	chatHandler := handlers.NewChatHandler(chatService)
	userHandler := handlers.NewUserHandler(userRepo, keysRepo)
	fileHandler := handlers.NewFileHandler(fileService)
	wsHandler   := ws.NewWSHandler(hub, messageService, chatRepo, userRepo, jwtMgr)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api/v1")

	// Публичные маршруты
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login",    authHandler.Login)
		auth.POST("/refresh",  authHandler.Refresh)
		auth.POST("/logout",   authHandler.Logout)
	}

	// WebSocket — авторизация внутри хендлера через query token
	v1.GET("/ws", wsHandler.Handle)

	// Защищённые маршруты
	protected := v1.Group("")
	protected.Use(middleware.Auth(jwtMgr))
	{
		protected.GET("/auth/me", authHandler.Me)
		protected.PUT("/auth/me/bio", authHandler.UpdateBio)
		protected.POST("/auth/me/avatar", authHandler.UploadAvatar)

		protected.GET("/chats",              chatHandler.List)
		protected.POST("/chats",             chatHandler.Create)
		protected.GET("/chats/:id/messages", chatHandler.GetMessages)
		protected.POST("/chats/:id/read",    chatHandler.MarkRead)
		protected.DELETE("/chats/:id",       chatHandler.Delete)

		protected.GET("/users/search",  userHandler.Search)
		protected.GET("/users/:id/keys", userHandler.GetPublicKeys)
		protected.POST("/users/keys",    userHandler.UploadKeys)

		protected.POST("/files/upload",      fileHandler.UploadFile)
		protected.GET("/files/:id/download", fileHandler.DownloadFile)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "1.0.0"})
	})

	log.Printf("🚀 Messenger API started on :%s (env=%s)", cfg.AppPort, cfg.Env)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
