package http

import (
	"fmt"
	"net/http"

	"casbin-authz-platform/internal/domain"
	"casbin-authz-platform/internal/middleware"
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type AuthzHandler struct {
	enforcer *casbin.SyncedEnforcer
}

func NewAuthzHandler(e *casbin.SyncedEnforcer) *AuthzHandler {
	return &AuthzHandler{enforcer: e}
}

// Enforce handles single authorization checks.
func (h *AuthzHandler) Enforce(c *gin.Context) {
	var req domain.CheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	allowed, err := h.enforcer.Enforce(req.Subject, req.Domain, req.Object, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro durante validação: " + err.Error()})
		return
	}

	reason := ""
	if allowed {
		reason = fmt.Sprintf("Matched policy: %s, %s, %s, %s", req.Subject, req.Domain, req.Object, req.Action)
	} else {
		reason = "Access denied: no matching policy or role"
	}

	traceID := c.GetString(middleware.ContextTraceID)

	c.JSON(http.StatusOK, gin.H{
		"allowed":  allowed,
		"reason":   reason,
		"trace_id": traceID,
	})
}

// BatchEnforce handles high-throughput authorization batch checks.
func (h *AuthzHandler) BatchEnforce(c *gin.Context) {
	var req domain.BatchCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	results := make([]domain.BatchCheckResult, len(req.Checks))
	for i, item := range req.Checks {
		allowed, _ := h.enforcer.Enforce(item.Subject, item.Domain, item.Object, item.Action)
		results[i] = domain.BatchCheckResult{
			Check:   item,
			Allowed: allowed,
		}
	}

	traceID := c.GetString(middleware.ContextTraceID)

	c.JSON(http.StatusOK, gin.H{
		"data":     results,
		"trace_id": traceID,
	})
}
