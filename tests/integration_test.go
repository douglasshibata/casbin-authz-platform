package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"casbin-authz-platform/internal/engine"
	handler "casbin-authz-platform/internal/handler/http"
	"casbin-authz-platform/internal/middleware"
	"casbin-authz-platform/pkg/logger"

	"github.com/gin-gonic/gin"
)

func TestIntegrationEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, _ := logger.NewLogger("debug")
	modelPath := filepath.Join("..", "config", "model_rbac_tenants.conf")

	cfg := engine.Config{
		DriverName:     "sqlite",
		DataSourceName: "file::memory:?cache=shared",
		ModelPath:      modelPath,
	}

	ctx := context.Background()
	ee, err := engine.NewEnforcerEngine(ctx, cfg, log)
	if err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}
	defer ee.Close()

	// Setup basic test policies:
	// p, role, dom, obj, act
	_, _ = ee.AddNamedPolicy("p", "admin", "tenant-1", "/api/v1/resource", "GET")
	_, _ = ee.AddNamedGroupingPolicy("g", "alice", "admin", "tenant-1")
	_, _ = ee.AddNamedGroupingPolicy("g2", "/api/v1/resource", "/api/v1/resource", "tenant-1")

	r := gin.New()
	r.Use(middleware.TraceMiddleware())

	authzHandler := handler.NewAuthzHandler(ee.SyncedEnforcer)
	policyHandler := handler.NewPolicyHandler(ee.SyncedEnforcer, log)
	authzMw := middleware.NewAuthorizer(ee.SyncedEnforcer, log)

	r.POST("/api/v1/enforce", authzHandler.Enforce)
	r.POST("/api/v1/batch-enforce", authzHandler.BatchEnforce)
	r.POST("/api/v1/policies", policyHandler.AddPolicy)
	r.GET("/api/v1/resource", authzMw.RequirePermission(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("Single Enforce Allowed", func(t *testing.T) {
		body := strings.NewReader(`{"sub":"alice","dom":"tenant-1","obj":"/api/v1/resource","act":"GET"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/enforce", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"allowed":true`) {
			t.Errorf("Expected allowed true in response: %s", w.Body.String())
		}
	})

	t.Run("Batch Enforce", func(t *testing.T) {
		body := strings.NewReader(`{
			"checks": [
				{"sub":"alice","dom":"tenant-1","obj":"/api/v1/resource","act":"GET"},
				{"sub":"bob","dom":"tenant-1","obj":"/api/v1/resource","act":"GET"}
			]
		}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/batch-enforce", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"allowed":true`) || !strings.Contains(w.Body.String(), `"allowed":false`) {
			t.Errorf("Unexpected batch response: %s", w.Body.String())
		}
	})

	t.Run("Middleware RequirePermission Authorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/resource", nil)
		req.Header.Set("X-User-ID", "alice")
		req.Header.Set("X-Tenant-ID", "tenant-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Middleware RequirePermission Forbidden", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/resource", nil)
		req.Header.Set("X-User-ID", "bob")
		req.Header.Set("X-Tenant-ID", "tenant-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
