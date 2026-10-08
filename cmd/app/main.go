package main

import (
	"context"
	"log/slog"
	"os"

	"booking/config"
	"booking/internal/app"
)

func main() {
	c, err := config.Load()
	if err != nil {
		slog.Error("config.Load", "err", err)
		os.Exit(1)
	}

	err = app.Run(context.Background(), c)
	if err != nil {
		slog.Error("app.Run", "err", err)
		os.Exit(1)
	}
}
