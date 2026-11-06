package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Spruhantom/tomos-chat-server/internal/api"
)

func main() {
	databaseURL := os.Getenv("TOMCHAT_DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("TOMCHAT_DATABASE_URL must be set")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil { log.Fatalf("configure database: %v", err) }
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil { log.Fatalf("connect to PostgreSQL: %v", err) }

	registrationEnabled := os.Getenv("TOMCHAT_REGISTRATION_ENABLED") == "true"
	handler := api.New(pool, registrationEnabled)
	if err := handler.Migrate(); err != nil { log.Fatalf("database migrations: %v", err) }

	addr := os.Getenv("TOMCHAT_LISTEN_ADDR")
	if addr == "" { addr = "127.0.0.1:8080" }

	server := &http.Server{
		Addr: addr,
		Handler: handler.Handler(),
		ReadHeaderTimeout: 5*time.Second,
		IdleTimeout: 60*time.Second,
	}
	log.Printf("TomChat server listening on %s (registration enabled: %t)", addr, registrationEnabled)
	log.Fatal(server.ListenAndServe())
}
