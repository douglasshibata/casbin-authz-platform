package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	HeaderTraceID   = "X-Trace-ID"
	HeaderRequestID = "X-Request-ID"
	ContextTraceID  = "trace_id"
)

// TraceMiddleware extracts or generates trace_id and request_id and sets it in context and response headers.
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(HeaderTraceID)
		if traceID == "" {
			traceID = c.GetHeader(HeaderRequestID)
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Set(ContextTraceID, traceID)
		c.Header(HeaderTraceID, traceID)
		c.Header(HeaderRequestID, traceID)

		c.Next()
	}
}
