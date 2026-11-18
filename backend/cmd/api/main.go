package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/database"
	"rantauplus/backend/database/postgres"
	"rantauplus/backend/internal/config"
	"rantauplus/backend/internal/httpserver"
	assistantclient "rantauplus/backend/internal/integration/assistant"
	"rantauplus/backend/internal/modules/assistant"
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
	var checkDatabase func(context.Context) error
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := postgres.Open(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			return errors.New("PostgreSQL gagal dihubungi; periksa DATABASE_URL dan status server")
		}
		defer pool.Close()
		checkDatabase = func(ctx context.Context) error {
			return database.CheckReady(ctx, pool)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	var assistantService *assistant.Service
	if cfg.DemoEnabled && cfg.AIServiceURL != "" {
		client, err := assistantclient.NewClient(cfg.AIServiceURL)
		if err != nil {
			return err
		}
		assistantService = assistant.NewService(client)
	}
	server := &http.Server{
		Addr: cfg.Address, Handler: httpserver.New(cfg.DemoEnabled, assistantService, checkDatabase),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: time.Minute,
	}
	log.Printf("Rantau+ API mendengarkan alamat %s (demo=%t)", cfg.Address, cfg.DemoEnabled)
	return server.ListenAndServe()
}
