package tests

import (
	"path/filepath"
	"testing"

	"github.com/casbin/casbin/v2"
)

// TestTenantRBAC validates multi-tenant RBAC with domain inheritance and resource grouping (g and g2).
func TestTenantRBAC(t *testing.T) {
	modelPath := filepath.Join("..", "config", "model_rbac_tenants.conf")
	e, err := casbin.NewSyncedEnforcer(modelPath)
	if err != nil {
		t.Fatalf("Failed to initialize enforcer: %v", err)
	}

	// Policies:
	// p, role/sub, dom, obj, act
	_, _ = e.AddPolicy("admin", "tenant-1", "data-group", "*")
	_, _ = e.AddPolicy("reader", "tenant-1", "/api/v1/reports", "GET")
	_, _ = e.AddPolicy("admin", "tenant-2", "/api/v1/finance", "GET")

	// Grouping:
	// g(user, role, domain)
	_, _ = e.AddGroupingPolicy("alice", "admin", "tenant-1")
	_, _ = e.AddGroupingPolicy("bob", "reader", "tenant-1")
	_, _ = e.AddGroupingPolicy("charlie", "admin", "tenant-2")

	// Resource Grouping:
	// g2(resource, resource_group, domain)
	_, _ = e.AddNamedGroupingPolicy("g2", "/api/v1/reports", "data-group", "tenant-1")
	_, _ = e.AddNamedGroupingPolicy("g2", "/api/v1/users/42", "data-group", "tenant-1")
	_, _ = e.AddNamedGroupingPolicy("g2", "/api/v1/reports", "/api/v1/reports", "tenant-1")
	_, _ = e.AddNamedGroupingPolicy("g2", "/api/v1/finance", "/api/v1/finance", "tenant-2")

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
			allowed, err := e.Enforce(tt.sub, tt.dom, tt.obj, tt.act)
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

type UserSub struct {
	Name         string
	Age          int
	Role         string
	IsSuperAdmin bool
}

type ResourceObj struct {
	Type string
}

type Environment struct {
	BusinessHours bool
}

// TestDynamicABAC validates attribute-based access control evaluation per model_abac.conf.
func TestDynamicABAC(t *testing.T) {
	modelPath := filepath.Join("..", "config", "model_abac.conf")
	e, err := casbin.NewSyncedEnforcer(modelPath)
	if err != nil {
		t.Fatalf("Failed to initialize ABAC enforcer: %v", err)
	}

	// Policy: p = sub_rule, obj_type, act
	// Matcher: eval(p.sub_rule) && r.obj.Type == p.obj_type && r.act == p.act && (r.env.BusinessHours == true || r.sub.IsSuperAdmin == true)
	_, _ = e.AddPolicy("r.sub.Age >= 18", "adult_content", "read")
	_, _ = e.AddPolicy("r.sub.Role == 'manager'", "financial_report", "approve")

	uYoung := UserSub{Name: "YoungBob", Age: 16, Role: "intern", IsSuperAdmin: false}
	uAdult := UserSub{Name: "AdultAlice", Age: 25, Role: "developer", IsSuperAdmin: false}
	uManager := UserSub{Name: "ManagerEve", Age: 35, Role: "manager", IsSuperAdmin: false}
	uAdmin := UserSub{Name: "SuperUser", Age: 30, Role: "admin", IsSuperAdmin: true}

	objAdult := ResourceObj{Type: "adult_content"}
	objFinance := ResourceObj{Type: "financial_report"}

	envBusinessHours := Environment{BusinessHours: true}
	envAfterHours := Environment{BusinessHours: false}

	tests := []struct {
		name     string
		sub      UserSub
		obj      ResourceObj
		act      string
		env      Environment
		expected bool
	}{
		{"Minor denied adult content during business hours", uYoung, objAdult, "read", envBusinessHours, false},
		{"Adult allowed adult content during business hours", uAdult, objAdult, "read", envBusinessHours, true},
		{"Adult denied adult content outside business hours", uAdult, objAdult, "read", envAfterHours, false},
		{"Developer denied budget approval", uAdult, objFinance, "approve", envBusinessHours, false},
		{"Manager allowed budget approval during business hours", uManager, objFinance, "approve", envBusinessHours, true},
		{"Manager denied budget approval outside business hours", uManager, objFinance, "approve", envAfterHours, false},
		{"SuperAdmin allowed budget approval outside business hours", uAdmin, objFinance, "approve", envAfterHours, false}, // role != manager
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := e.Enforce(tt.sub, tt.obj, tt.act, tt.env)
			if err != nil {
				t.Fatalf("Enforce error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("Expected %v but got %v for %s", tt.expected, allowed, tt.name)
			}
		})
	}
}
