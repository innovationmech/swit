package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouteManager_SetRealtimePusher_NoDuplicateRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a new router and route manager
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	// Setup routes first
	err := rm.SetupRoutes()
	assert.NoError(t, err)

	// Create mock API and pusher with correct constructors
	coordinator := new(MockSagaCoordinator)
	collector, _ := NewSagaMetricsCollector(nil)
	metricsAPI := NewMetricsAPI(collector, coordinator)
	realtimePusher := NewRealtimePusher(collector, nil)

	// Set metrics API - this should register the base metrics routes
	rm.SetMetricsAPI(metricsAPI)

	// Now set realtime pusher - this should NOT cause panic or duplicate routes
	rm.SetRealtimePusher(realtimePusher)

	// Verify that all routes are properly registered by making test requests
	// Test basic metrics endpoint
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/metrics", nil)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Test realtime metrics endpoint
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/metrics/realtime", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Test SSE stream endpoint
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/api/metrics/stream", nil)
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	// Clean up
	realtimePusher.Stop()
}

func TestRouteManager_SetRealtimePusher_BeforeSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a new router and route manager
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	// Create mock API and pusher with correct constructors
	coordinator := new(MockSagaCoordinator)
	collector, _ := NewSagaMetricsCollector(nil)
	metricsAPI := NewMetricsAPI(collector, coordinator)
	realtimePusher := NewRealtimePusher(collector, nil)

	// Set APIs before SetupRoutes
	rm.SetMetricsAPI(metricsAPI)
	rm.SetRealtimePusher(realtimePusher)

	// Setup routes - should include SSE route
	err := rm.SetupRoutes()
	assert.NoError(t, err)

	// Verify that all routes are properly registered
	// Test SSE stream endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/metrics/stream", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Clean up
	realtimePusher.Stop()
}
