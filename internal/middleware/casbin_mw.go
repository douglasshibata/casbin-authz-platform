package middleware

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Authorizer struct {
	enforcer *casbin.SyncedEnforcer
	logger   *zap.Logger
}

func NewAuthorizer(e *casbin.SyncedEnforcer, l *zap.Logger) *Authorizer {
	return &Authorizer{enforcer: e, logger: l}
}

// RequirePermission intercepta e valida r = (sub, dom, obj, act)
func (a *Authorizer) RequirePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Context values injetados pelo middleware de Autenticação/JWT ou headers
		user := c.GetString("user_id")
		if user == "" {
			user = c.GetHeader("X-User-ID")
		}
		if user == "" {
			user = c.GetHeader("X-User")
		}

		tenant := c.GetString("tenant_id")
		if tenant == "" {
			tenant = c.GetHeader("X-Tenant-ID")
		}
		if tenant == "" {
			tenant = c.GetHeader("X-Tenant")
		}

		traceID := c.GetString(ContextTraceID)

		if user == "" || tenant == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":    "Identidade do usuário ou tenant ausente no contexto",
				"trace_id": traceID,
			})
			return
		}

		obj := c.Request.URL.Path
		act := c.Request.Method

		allowed, err := a.enforcer.Enforce(user, tenant, obj, act)
		if err != nil {
			a.logger.Error("Erro interno durante avaliação de política",
				zap.String("trace_id", traceID),
				zap.String("sub", user),
				zap.String("dom", tenant),
				zap.String("obj", obj),
				zap.String("act", act),
				zap.Error(err),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":    "Erro durante validação de autorização",
				"trace_id": traceID,
			})
			return
		}

		if !allowed {
			a.logger.Warn("Acesso Negado (403 Forbidden)",
				zap.String("trace_id", traceID),
				zap.String("sub", user),
				zap.String("dom", tenant),
				zap.String("obj", obj),
				zap.String("act", act),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":    "Permissão insuficiente",
				"detail":   "Nenhuma política concedeu acesso a esta ação no tenant informado",
				"trace_id": traceID,
			})
			return
		}

		c.Next()
	}
}
