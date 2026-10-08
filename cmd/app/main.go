package main

import (
	"log/slog"
	"os"

	"booking/config"
	"booking/internal/app"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config.Load", "err", err)
		os.Exit(1)
	}

	if err := app.Run(cfg); err != nil {
		slog.Error("app.Run", "err", err)
		os.Exit(1)
	}
}
