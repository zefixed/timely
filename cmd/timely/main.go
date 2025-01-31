package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"
	"timely/internal/config"
	"timely/internal/lib/logger/sl"
	"timely/internal/storage/postgres"
)

const (
    envLocal    = "local"
    envDev      = "dev"
    envProd     = "prod"
)

func main(){
    cfg := config.MustLoad()

    log := setupLogger(cfg.Env)

    log.Info("starting pollsAPI", slog.String("env", cfg.Env))
    log.Debug("debug messages are enabled")
    
    storage, err := postgres.New(cfg.StoragePath)
    if err != nil {
        log.Error("failed to init storage", sl.Err(err))
        os.Exit(1)
    }
        
    _ = storage

    // TODO: init router: chi, "chi render"

    // TODO: run server
}

func setupLogger(env string) *slog.Logger {
    var log *slog.Logger

    switch env {
    case envLocal:
        log = slog.New(
            slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
        )
    case envDev:
        log = slog.New(
            slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
        )
    case envProd:
        log = slog.New(
            slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
        )
    }
    
    return log
}
