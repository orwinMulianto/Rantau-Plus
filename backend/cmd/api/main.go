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
	"rantauplus/backend/internal/integration/authmail"
	"rantauplus/backend/internal/modules/assistant"
	"rantauplus/backend/internal/modules/auth"
	"rantauplus/backend/internal/modules/finance"
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
	var authService *auth.Service
	var setupService *finance.SetupService
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		db, err := postgres.Open(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			return errors.New("PostgreSQL gagal dihubungi; periksa DATABASE_URL dan status server")
		}
		pool, err := db.DB()
		if err != nil {
			return err
		}
		defer pool.Close()
		checkDatabase = func(ctx context.Context) error {
			return database.CheckReady(ctx, db)
		}
		if cfg.Auth != nil {
			setupService = finance.NewSetupService(db)
			sender, err := authmail.NewSender(cfg.Auth.SMTPAddr, cfg.Auth.SMTPFrom, cfg.Auth.SMTPUsername,
				cfg.Auth.SMTPPassword, cfg.Auth.SMTPPlaintext)
			if err != nil {
				return err
			}
			authService, err = auth.NewService(auth.NewRepository(db), cfg.Auth.EmailTokenKey, cfg.Auth.PasswordBlocklist)
			if err != nil {
				return err
			}
			workerContext, stopWorker := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				authService.RunEmailDelivery(workerContext, sender, auth.EmailLinks{Verify: cfg.Auth.VerifyURL, Reset: cfg.Auth.ResetURL})
			}()
			defer func() {
				stopWorker()
				<-done
			}()
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
	routes := httpserver.New(cfg.DemoEnabled, assistantService, checkDatabase, authService, setupService)
	if cfg.Auth != nil {
		if err := routes.SetTrustedProxies(cfg.Auth.TrustedProxies); err != nil {
			return errors.New("AUTH_TRUSTED_PROXIES harus berisi IP atau CIDR yang valid")
		}
	}
	server := &http.Server{
		Addr: cfg.Address, Handler: routes,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: time.Minute,
	}
	log.Printf("Rantau+ API mendengarkan alamat %s (demo=%t)", cfg.Address, cfg.DemoEnabled)
	return server.ListenAndServe()
}
