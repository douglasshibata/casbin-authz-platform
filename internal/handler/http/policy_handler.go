package http

import (
	"net/http"

	"casbin-authz-platform/internal/domain"
	"casbin-authz-platform/internal/middleware"
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PolicyHandler struct {
	enforcer *casbin.SyncedEnforcer
	logger   *zap.Logger
}

func NewPolicyHandler(e *casbin.SyncedEnforcer, l *zap.Logger) *PolicyHandler {
	return &PolicyHandler{
		enforcer: e,
		logger:   l,
	}
}

// AddPolicy registers a new policy rule or role assignment.
func (h *PolicyHandler) AddPolicy(c *gin.Context) {
	var req domain.PolicyRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	var added bool
	var err error

	ptype := req.PType
	if ptype == "" {
		ptype = "p"
	}

	if len(req.Params) > 0 {
		if ptype == "g" || ptype == "g2" {
			added, err = h.enforcer.AddNamedGroupingPolicy(ptype, req.Params)
		} else {
			added, err = h.enforcer.AddNamedPolicy(ptype, req.Params)
		}
	} else if req.Sub != "" && req.Dom != "" {
		if ptype == "g" {
			// g, user, role, domain
			added, err = h.enforcer.AddNamedGroupingPolicy("g", req.Sub, req.Obj, req.Dom)
		} else {
			// p, sub, dom, obj, act
			added, err = h.enforcer.AddNamedPolicy("p", req.Sub, req.Dom, req.Obj, req.Act)
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetros de política insuficientes"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	traceID := c.GetString(middleware.ContextTraceID)

	c.JSON(http.StatusOK, gin.H{
		"success":  added,
		"message":  "Policy registered successfully",
		"trace_id": traceID,
	})
}

// RemovePolicy removes a policy rule or role assignment.
func (h *PolicyHandler) RemovePolicy(c *gin.Context) {
	var req domain.PolicyRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	var removed bool
	var err error

	ptype := req.PType
	if ptype == "" {
		ptype = "p"
	}

	if len(req.Params) > 0 {
		if ptype == "g" || ptype == "g2" {
			removed, err = h.enforcer.RemoveNamedGroupingPolicy(ptype, req.Params)
		} else {
			removed, err = h.enforcer.RemoveNamedPolicy(ptype, req.Params)
		}
	} else if req.Sub != "" && req.Dom != "" {
		if ptype == "g" {
			removed, err = h.enforcer.RemoveNamedGroupingPolicy("g", req.Sub, req.Obj, req.Dom)
		} else {
			removed, err = h.enforcer.RemoveNamedPolicy("p", req.Sub, req.Dom, req.Obj, req.Act)
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetros de política insuficientes"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	traceID := c.GetString(middleware.ContextTraceID)

	c.JSON(http.StatusOK, gin.H{
		"success":  removed,
		"message":  "Policy removed successfully",
		"trace_id": traceID,
	})
}
