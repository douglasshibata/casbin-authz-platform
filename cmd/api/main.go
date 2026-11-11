package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"casbin-authz-platform/config"
	"casbin-authz-platform/internal/engine"
	handler "casbin-authz-platform/internal/handler/http"
	"casbin-authz-platform/internal/middleware"
	"casbin-authz-platform/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func setupRouter(ee *engine.EnforcerEngine, log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.TraceMiddleware())

	authzHandler := handler.NewAuthzHandler(ee.SyncedEnforcer)
	policyHandler := handler.NewPolicyHandler(ee.SyncedEnforcer, log)
	authzMw := middleware.NewAuthorizer(ee.SyncedEnforcer, log)

	r.GET("/health", func(c *gin.Context) {
		traceID := c.GetString(middleware.ContextTraceID)
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "casbin-authz-platform",
			"trace_id": traceID,
		})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/enforce", authzHandler.Enforce)
		api.POST("/batch-enforce", authzHandler.BatchEnforce)
		api.POST("/policies", policyHandler.AddPolicy)
		api.DELETE("/policies", policyHandler.RemovePolicy)

		// Protected resource example utilizing Casbin interceptor middleware
		api.GET("/protected/resource", authzMw.RequirePermission(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message":  "Acesso permitido a recurso protegido",
				"user":     c.GetString("user_id"),
				"tenant":   c.GetString("tenant_id"),
				"trace_id": c.GetString(middleware.ContextTraceID),
			})
		})
	}

	return r
}

func main() {
	cfg := config.LoadConfig()

	log, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		panic("falha ao inicializar logger: " + err.Error())
	}
	defer func() {
		_ = log.Sync()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engineCfg := engine.Config{
		DriverName:     cfg.DriverName,
		DataSourceName: cfg.DataSourceName,
		ModelPath:      cfg.ModelPath,
		RedisAddr:      cfg.RedisAddr,
		RedisPassword:  cfg.RedisPassword,
		RedisDB:        cfg.RedisDB,
	}

	ee, err := engine.NewEnforcerEngine(ctx, engineCfg, log)
	if err != nil {
		log.Fatal("falha ao inicializar motor de autorização", zap.Error(err))
	}
	defer func() {
		_ = ee.Close()
	}()

	router := setupRouter(ee, log)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Info("Iniciando Casbin Authorization Platform",
			zap.String("port", cfg.ServerPort),
			zap.String("driver", cfg.DriverName),
			zap.String("model", cfg.ModelPath),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Erro fatal no servidor HTTP", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Recebido sinal de encerramento. Iniciando graceful shutdown...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Falha durante o encerramento do servidor HTTP", zap.Error(err))
	}

	log.Info("Servidor encerrado com sucesso.")
}
