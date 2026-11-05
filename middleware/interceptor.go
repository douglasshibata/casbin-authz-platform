package middleware

import (
	"log"
	"net/http"

	"casbin-authz-platform/policy"
	"github.com/gin-gonic/gin"
)

// SubjectExtractor defines a strategy to extract sub, dom, obj, act from Gin context.
type SubjectExtractor func(c *gin.Context) (sub string, dom string, obj string, act string)

// DefaultExtractor extracts sub, dom, obj, act from headers and request URL.
func DefaultExtractor(c *gin.Context) (string, string, string, string) {
	sub := c.GetHeader("X-User")
	if sub == "" {
		sub = c.GetString("user")
	}
	if sub == "" {
		sub = "anonymous"
	}

	dom := c.GetHeader("X-Tenant")
	if dom == "" {
		dom = c.Param("tenant")
	}
	if dom == "" {
		dom = "default"
	}

	obj := c.Request.URL.Path
	act := c.Request.Method

	return sub, dom, obj, act
}

// CasbinInterceptor creates a Gin middleware for request-level authorization.
func CasbinInterceptor(pm *policy.PolicyManager, extractor SubjectExtractor) gin.HandlerFunc {
	if extractor == nil {
		extractor = DefaultExtractor
	}

	return func(c *gin.Context) {
		sub, dom, obj, act := extractor(c)

		allowed, err := pm.Enforce(sub, dom, obj, act)
		if err != nil {
			log.Printf("[Interceptor] Enforce evaluation error: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   "authorization_eval_error",
				"message": err.Error(),
			})
			return
		}

		if !allowed {
			log.Printf("[Interceptor] DENY: sub=%s dom=%s obj=%s act=%s", sub, dom, obj, act)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"allowed": false,
				"reason":  "Access denied by authorization policy",
			})
			return
		}

		c.Next()
	}
}
