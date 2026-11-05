package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"casbin-authz-platform/policy"
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type EnforceRequest struct {
	Sub string `json:"sub" binding:"required"`
	Dom string `json:"dom" binding:"required"`
	Obj string `json:"obj" binding:"required"`
	Act string `json:"act" binding:"required"`
}

type PolicyRequest struct {
	Sub string `json:"sub" binding:"required"`
	Dom string `json:"dom" binding:"required"`
	Obj string `json:"obj" binding:"required"`
	Act string `json:"act" binding:"required"`
}

func setupRouter(pm *policy.PolicyManager) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "casbin-authz-platform",
		})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/enforce", func(c *gin.Context) {
			var req EnforceRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			allowed, err := pm.Enforce(req.Sub, req.Dom, req.Obj, req.Act)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			reason := ""
			if allowed {
				reason = fmt.Sprintf("Matched policy: %s, %s, %s, %s", req.Sub, req.Dom, req.Obj, req.Act)
			} else {
				reason = "Access denied: no matching policy or role"
			}

			c.JSON(http.StatusOK, gin.H{
				"allowed": allowed,
				"reason":  reason,
			})
		})

		api.POST("/policies", func(c *gin.Context) {
			var req PolicyRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			added, err := pm.AddNamedPolicy("p", req.Sub, req.Dom, req.Obj, req.Act)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": added,
				"message": "Policy registered successfully",
			})
		})
	}

	return r
}

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	modelPath := os.Getenv("CASBIN_MODEL_PATH")
	if modelPath == "" {
		modelPath = "config/model_rbac_tenants.conf"
	}

	e, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		log.Fatalf("Failed to create Casbin enforcer: %v", err)
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	watcher, err := policy.NewRedisWatcher(redisURL, "casbin_policy_update")
	if err != nil {
		log.Printf("Watcher init error: %v", err)
	}

	pm := policy.NewPolicyManager(e, watcher)
	router := setupRouter(pm)

	log.Printf("Starting Casbin Authorization Platform on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
