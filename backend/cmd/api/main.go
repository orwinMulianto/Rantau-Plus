package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/config"
	"rantauplus/backend/internal/integration/assistant"
	"rantauplus/backend/internal/repository/postgres"
	"rantauplus/backend/internal/transport/http/router"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := postgres.Open(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			return errors.New("PostgreSQL gagal dihubungi; periksa DATABASE_URL dan status server")
		}
		defer pool.Close()
	}

	gin.SetMode(gin.ReleaseMode)
	var aiClient *assistant.Client
	if cfg.DemoEnabled && cfg.AIServiceURL != "" {
		client, err := assistant.NewClient(cfg.AIServiceURL)
		if err != nil {
			return err
		}
		aiClient = client
	}
	server := &http.Server{
		Addr: cfg.Address, Handler: router.New(cfg.DemoEnabled, aiClient),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: time.Minute,
	}
	log.Printf("Rantau+ API mendengarkan alamat %s (demo=%t)", cfg.Address, cfg.DemoEnabled)
	return server.ListenAndServe()
}
