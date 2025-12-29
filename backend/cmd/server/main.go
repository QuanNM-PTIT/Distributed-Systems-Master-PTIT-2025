package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"p2p-chat-app/backend/internal/config"
	"p2p-chat-app/backend/internal/db"
	"p2p-chat-app/backend/internal/handlers"
	"p2p-chat-app/backend/internal/middleware"
	"p2p-chat-app/backend/internal/ws"
)

func main() {
	cfg := config.Load()
	database, err := db.New(cfg.DBDSN)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}

	router := gin.Default()
	router.Use(middleware.CORS(cfg.AllowedOrigin))
	router.Use(middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst).Middleware())

	authHandler := &handlers.AuthHandler{DB: database, JWTSecret: cfg.JWTSecret}
	friendsHandler := &handlers.FriendsHandler{DB: database}
	groupsHandler := &handlers.GroupsHandler{DB: database}
	presenceHandler := &handlers.PresenceHandler{DB: database}
	usersHandler := &handlers.UsersHandler{DB: database}

	api := router.Group("/api/v1")
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)

	authed := api.Group("")
	authed.Use(middleware.JWTAuth(cfg.JWTSecret))
	authed.POST("/friends/request", friendsHandler.Request)
	authed.POST("/friends/accept", friendsHandler.Accept)
	authed.GET("/friends/requests", friendsHandler.Requests)
	authed.GET("/friends/list", friendsHandler.List)
	authed.POST("/groups", groupsHandler.Create)
	authed.POST("/groups/invite", groupsHandler.Invite)
	authed.POST("/groups/leave", groupsHandler.Leave)
	authed.GET("/groups/:id/members", groupsHandler.Members)
	authed.GET("/groups/list", groupsHandler.List)
	authed.GET("/presence", presenceHandler.List)
	authed.GET("/users/me", usersHandler.Me)
	authed.GET("/users/search", usersHandler.Search)

	hub := ws.NewHub()
	wsHandler := &ws.Handler{Hub: hub, DB: database, JWTSecret: cfg.JWTSecret}
	router.GET("/ws", wsHandler.ServeWS)

	log.Printf("server listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
