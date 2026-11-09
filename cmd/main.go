package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gitverse.ru/zagvozdeen/agenda/internal/api"
	"gitverse.ru/zagvozdeen/agenda/internal/cli"
	"gitverse.ru/zagvozdeen/agenda/internal/config"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Failed to run application", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	return cli.Run(ctx, func(ctx context.Context, cfg *config.Config, log *slog.Logger, storage store.Storage) error {
		return api.NewApplication(cfg, log, storage).Run(ctx)
	})
}
