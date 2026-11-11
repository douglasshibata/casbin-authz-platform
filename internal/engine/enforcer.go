package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	DriverName     string
	DataSourceName string
	ModelPath      string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
}

type EnforcerEngine struct {
	*casbin.SyncedEnforcer
	watcher persist.Watcher
	logger  *zap.Logger
}

func NewEnforcerEngine(ctx context.Context, cfg Config, logger *zap.Logger) (*EnforcerEngine, error) {
	var dialector gorm.Dialector
	if cfg.DriverName == "postgres" {
		dialector = postgres.Open(cfg.DataSourceName)
	} else {
		dialector = sqlite.Open(cfg.DataSourceName)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no banco de políticas: %w", err)
	}

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar o adapter GORM do Casbin: %w", err)
	}

	syncedEnforcer, err := casbin.NewSyncedEnforcer(cfg.ModelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar o SyncedEnforcer: %w", err)
	}

	// Carrega todas as regras persistidas na inicialização
	if err := syncedEnforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("falha ao carregar políticas do storage: %w", err)
	}

	var watcher persist.Watcher
	if cfg.RedisAddr != "" {
		w, err := rediswatcher.NewWatcher(cfg.RedisAddr, rediswatcher.WatcherOptions{
			Options: redis.Options{
				Password: cfg.RedisPassword,
				DB:       cfg.RedisDB,
			},
			Channel: "/casbin/policy/sync",
		})
		if err != nil {
			logger.Warn("Redis Watcher inativo. Operando em modo de réplica isolada", zap.Error(err))
		} else {
			watcher = w
			if err := syncedEnforcer.SetWatcher(watcher); err != nil {
				return nil, fmt.Errorf("falha ao registrar o watcher no enforcer: %w", err)
			}

			// Hook acionado toda vez que outra réplica publica um evento de alteração
			_ = watcher.SetUpdateCallback(func(msg string) {
				logger.Info("Atualização de política recebida via Redis Pub/Sub. Recarregando memória...", zap.String("source", msg))
				if err := syncedEnforcer.LoadPolicy(); err != nil {
					logger.Error("Erro crítico ao recarregar políticas via watcher", zap.Error(err))
				}
			})
		}
	}

	// Ativa cache interno de decisões para ganho de performance (opcional, configurável)
	syncedEnforcer.EnableAutoSave(true)

	// Sincronização periódica de salvaguarda contra perdas de pacotes no Pub/Sub (Heartbeat)
	go startPeriodicSync(ctx, syncedEnforcer, 60*time.Second, logger)

	return &EnforcerEngine{
		SyncedEnforcer: syncedEnforcer,
		watcher:        watcher,
		logger:         logger,
	}, nil
}

func startPeriodicSync(ctx context.Context, e *casbin.SyncedEnforcer, interval time.Duration, logger *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.LoadPolicy(); err != nil {
				logger.Warn("Falha durante recarga periódica preventiva de políticas", zap.Error(err))
			}
		}
	}
}

func (ee *EnforcerEngine) Close() error {
	if ee.watcher != nil {
		ee.watcher.Close()
	}
	return nil
}
