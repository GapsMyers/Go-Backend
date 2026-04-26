package main

import (
	"backend/auth"
	"backend/config"
	"backend/database"
	"backend/handlers"
	"backend/middleware"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if strings.EqualFold(cfg.AppEnv, "production") {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpireMinutes)
	authHandler := handlers.NewAuthHandler(db, jwtService)
	matkulHandler := handlers.NewMatkulHandler(db)
	deadlineHandler := handlers.NewDeadlineHandler(db)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.AllowedOrigins))

	router.GET("/health", handlers.Health)

	api := router.Group("/api")
	{
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)

		protected := api.Group("")
		protected.Use(middleware.JWTAuth(jwtService))
		{
			protected.POST("/matkul", matkulHandler.Create)
			protected.GET("/matkul", matkulHandler.List)
			protected.PATCH("/matkul/:id", matkulHandler.Update)
			protected.GET("/me", authHandler.Me)
			protected.POST("/deadlines", deadlineHandler.Create)
			protected.GET("/deadlines", deadlineHandler.List)
			protected.PATCH("/deadlines/:id", deadlineHandler.Update)
			protected.PATCH("/deadlines/:id/toggle", deadlineHandler.ToggleStatus)
			protected.DELETE("/deadlines/:id", deadlineHandler.Delete)
		}
	}

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Printf("server running on :%s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("start server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	log.Println("server shutdown complete")
}
