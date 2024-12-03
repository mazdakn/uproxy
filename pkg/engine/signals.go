package engine

import (
	"context"
	"os/signal"
	"syscall"
)

func setupSignals() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
