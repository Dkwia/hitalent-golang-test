package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"organization-api/internal/config"
	"organization-api/internal/db"
	"organization-api/internal/handler"
	"organization-api/internal/repository"
	"organization-api/internal/service"

	"github.com/pressly/goose/v3"
)

func main() {
	cfg := config.Load()
	gdb, err := db.Open(cfg.DSN())
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		log.Fatalf("db handle: %v", err)
	}
	defer sqlDB.Close()

	goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		log.Fatalf("goose up: %v", err)
	}

	repo := repository.New(gdb)
	svc := service.New(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/departments/", h)
	mux.Handle("/departments", h)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: mux}

	go func() {
		log.Printf("server listening on %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
