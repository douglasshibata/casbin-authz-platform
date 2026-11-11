package engine

import (
	"context"
	"time"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
)

// WatcherReconnector coordinates watcher resilience and health checks.
type WatcherReconnector struct {
	enforcer *casbin.SyncedEnforcer
	logger   *zap.Logger
}

// NewWatcherReconnector creates a new resilience manager for Casbin watchers.
func NewWatcherReconnector(e *casbin.SyncedEnforcer, logger *zap.Logger) *WatcherReconnector {
	return &WatcherReconnector{
		enforcer: e,
		logger:   logger,
	}
}

// StartAutoSync starts a periodic background policy reload to prevent split-brain.
func (wr *WatcherReconnector) StartAutoSync(ctx context.Context, interval time.Duration) {
	go startPeriodicSync(ctx, wr.enforcer, interval, wr.logger)
}
