package middleware

import (
	"github.com/innovationmech/swit/pkg/messaging"
)

// Register built-in factories of this subpackage into the global registry.
func init() {
	if messaging.GlobalMiddlewareRegistry != nil {
		_ = messaging.GlobalMiddlewareRegistry.RegisterFactory("tracing", CreateTracingMiddleware)
	}
}


