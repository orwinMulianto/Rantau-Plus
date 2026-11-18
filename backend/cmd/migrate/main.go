package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"rantauplus/backend/database"
	"rantauplus/backend/database/postgres"
)

func main() {
	adoptInitial := flag.Bool("adopt-initial", false, "catat skema 000001 yang sebelumnya dipasang manual")
	flag.Parse()
	if err := run(*adoptInitial); err != nil {
		log.Fatal(err)
	}
}

func run(adoptInitial bool) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL harus diisi untuk menjalankan migrasi")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return errors.New("PostgreSQL gagal dihubungi. Periksa DATABASE_URL dan status server")
	}
	defer pool.Close()

	applied, err := database.Apply(ctx, pool, adoptInitial)
	if err != nil {
		return err
	}
	for _, version := range applied {
		log.Printf("Migrasi diterapkan: %s", version)
	}
	if len(applied) == 0 {
		log.Print("Database sudah menggunakan skema terbaru.")
	}
	return nil
}
