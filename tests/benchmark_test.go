package tests

import (
	"path/filepath"
	"testing"

	"github.com/casbin/casbin/v2"
)

func BenchmarkEnforceParallel(b *testing.B) {
	modelPath := filepath.Join("..", "config", "model_rbac_tenants.conf")
	e, err := casbin.NewSyncedEnforcer(modelPath)
	if err != nil {
		b.Fatalf("Falha ao inicializar enforcer: %v", err)
	}

	// Carrega políticas estáticas para carga:
	// p, sub, dom, obj, act
	// g, user, role, dom
	// g2, res, res_group, dom
	_, _ = e.AddPolicy("admin", "tenant-1", "/api/v1/resource", "write")
	_, _ = e.AddGroupingPolicy("alice", "admin", "tenant-1")
	_, _ = e.AddNamedGroupingPolicy("g2", "/api/v1/resource", "/api/v1/resource", "tenant-1")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			allowed, err := e.Enforce("alice", "tenant-1", "/api/v1/resource", "write")
			if err != nil || !allowed {
				b.Fatalf("Erro de enforcement durante benchmark")
			}
		}
	})
}
