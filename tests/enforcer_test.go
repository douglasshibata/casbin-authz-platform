package tests

import (
	"path/filepath"
	"testing"

	"casbin-authz-platform/policy"
	"github.com/casbin/casbin/v2"
)

// TestTenantRBAC validates multi-tenant RBAC with domain inheritance.
func TestTenantRBAC(t *testing.T) {
	modelPath := filepath.Join("..", "config", "model_rbac_tenants.conf")
	e, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		t.Fatalf("Failed to initialize enforcer: %v", err)
	}

	pm := policy.NewPolicyManager(e, nil)

	// Setup Tenant Policies:
	// p, admin, tenant-1, /api/v1/*, (GET|POST|DELETE)
	// p, reader, tenant-1, /api/v1/reports, GET
	// g, alice, admin, tenant-1
	// g, bob, reader, tenant-1
	// g, charlie, admin, tenant-2

	_, _ = pm.AddNamedPolicy("p", "admin", "tenant-1", "/api/v1/*", "(GET|POST|DELETE)")
	_, _ = pm.AddNamedPolicy("p", "reader", "tenant-1", "/api/v1/reports", "GET")
	_, _ = pm.AddNamedPolicy("g", "alice", "admin", "tenant-1")
	_, _ = pm.AddNamedPolicy("g", "bob", "reader", "tenant-1")
	_, _ = pm.AddNamedPolicy("g", "charlie", "admin", "tenant-2")
	_, _ = pm.AddNamedPolicy("p", "admin", "tenant-2", "/api/v1/finance", "GET")

	tests := []struct {
		name     string
		sub      string
		dom      string
		obj      string
		act      string
		expected bool
	}{
		{"Alice (admin tenant-1) accesses /api/v1/reports with GET", "alice", "tenant-1", "/api/v1/reports", "GET", true},
		{"Alice (admin tenant-1) deletes /api/v1/users/42", "alice", "tenant-1", "/api/v1/users/42", "DELETE", true},
		{"Bob (reader tenant-1) reads reports", "bob", "tenant-1", "/api/v1/reports", "GET", true},
		{"Bob (reader tenant-1) denied delete", "bob", "tenant-1", "/api/v1/reports", "DELETE", false},
		{"Cross-tenant isolation: Charlie denied tenant-1", "charlie", "tenant-1", "/api/v1/reports", "GET", false},
		{"Cross-tenant isolation: Alice denied tenant-2", "alice", "tenant-2", "/api/v1/finance", "GET", false},
		{"Charlie allowed on tenant-2 finance", "charlie", "tenant-2", "/api/v1/finance", "GET", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := pm.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("Enforce error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("Expected %v but got %v for (sub=%s, dom=%s, obj=%s, act=%s)",
					tt.expected, allowed, tt.sub, tt.dom, tt.obj, tt.act)
			}
		})
	}
}

type UserSubject struct {
	Name string
	Age  int
	Role string
}

// TestDynamicABAC validates attribute-based access control evaluation.
func TestDynamicABAC(t *testing.T) {
	modelPath := filepath.Join("..", "config", "model_abac.conf")
	e, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		t.Fatalf("Failed to initialize ABAC enforcer: %v", err)
	}

	// Policy: subject must be at least 18 years old to access adult content
	_, _ = e.AddPolicy("r.sub.Age >= 18", "/adult/content", "read")
	// Policy: subject must have Role == 'manager' to approve budget
	_, _ = e.AddPolicy("r.sub.Role == 'manager'", "/finance/budget", "approve")

	u1 := UserSubject{Name: "YoungBob", Age: 16, Role: "intern"}
	u2 := UserSubject{Name: "AdultAlice", Age: 25, Role: "developer"}
	u3 := UserSubject{Name: "ManagerEve", Age: 35, Role: "manager"}

	tests := []struct {
		name     string
		sub      UserSubject
		obj      string
		act      string
		expected bool
	}{
		{"Minor denied adult content", u1, "/adult/content", "read", false},
		{"Adult allowed adult content", u2, "/adult/content", "read", true},
		{"Developer denied budget approval", u2, "/finance/budget", "approve", false},
		{"Manager allowed budget approval", u3, "/finance/budget", "approve", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := e.Enforce(tt.sub, tt.obj, tt.act)
			if err != nil {
				t.Fatalf("Enforce error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("Expected %v but got %v for %s", tt.expected, allowed, tt.name)
			}
		})
	}
}

// BenchmarkEnforcer benchmarks decision latency ensuring sub-millisecond evaluation.
func BenchmarkEnforcer(b *testing.B) {
	modelPath := filepath.Join("..", "config", "model_rbac_tenants.conf")
	e, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		b.Fatalf("Failed to initialize enforcer: %v", err)
	}

	_, _ = e.AddNamedPolicy("p", "admin", "tenant-1", "/api/v1/*", "(GET|POST)")
	_, _ = e.AddNamedPolicy("g", "alice", "admin", "tenant-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Enforce("alice", "tenant-1", "/api/v1/data", "GET")
	}
}
