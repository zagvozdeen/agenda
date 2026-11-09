package logger

import (
	"fmt"
	"log/slog"
	"os"

	"gitverse.ru/zagvozdeen/agenda/internal/config"
)

func New(cfg *config.Config) (*slog.Logger, func(), error) {
	if cfg.App.IsProduction {
		file, err := os.OpenFile("agenda.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		return slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})), func() {
			if err := file.Close(); err != nil {
				slog.Error("Failed to close log file", slog.Any("error", err))
			}
		}, nil
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})), func() {}, nil
}
