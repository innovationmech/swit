// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package monitoring

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRouteManager_SetRealtimePusher_NoPanic verifies that SetRealtimePusher
// can be called after SetMetricsAPI without causing panic due to duplicate route registration.
// This is a regression test for the issue where calling SetRealtimePusher after SetMetricsAPI
// would cause Gin to panic when trying to re-register the same routes.
func TestRouteManager_SetRealtimePusher_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a new router and route manager
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	// Setup routes first
	err := rm.SetupRoutes()
	assert.NoError(t, err)

	// Create mock API
	coordinator := new(MockSagaCoordinator)
	collector, _ := NewSagaMetricsCollector(nil)
	metricsAPI := NewMetricsAPI(collector, coordinator)

	// Set metrics API - this should register the base metrics routes
	rm.SetMetricsAPI(metricsAPI)

	// Now set a nil realtime pusher - this should NOT cause panic or duplicate routes
	// The main purpose is to verify that setupMetricsRoutes() is not called again
	assert.NotPanics(t, func() {
		rm.SetRealtimePusher(nil)
	}, "Setting realtime pusher should not cause panic due to duplicate route registration")

	// Verify the fix by checking that we can set the pusher multiple times without panic
	assert.NotPanics(t, func() {
		rm.SetRealtimePusher(nil)
	}, "Setting realtime pusher multiple times should not cause panic")
}

// TestRouteManager_GetAPIGroup tests GetAPIGroup method.
func TestRouteManager_GetAPIGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	group := rm.GetAPIGroup()
	assert.NotNil(t, group, "GetAPIGroup should return non-nil group")
}

// TestRouteManager_SetQueryAPI tests SetQueryAPI method.
func TestRouteManager_SetQueryAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	coordinator := new(MockSagaCoordinator)
	queryAPI := NewSagaQueryAPI(coordinator)

	rm.SetQueryAPI(queryAPI)
	// Should not panic or error
}

// TestRouteManager_SetControlAPI tests SetControlAPI method.
func TestRouteManager_SetControlAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	coordinator := new(MockSagaCoordinator)
	controlAPI := NewSagaControlAPI(coordinator, nil)

	rm.SetControlAPI(controlAPI)
	// Should not panic or error
}

// TestRouteManager_SetVisualizationAPI tests SetVisualizationAPI method.
func TestRouteManager_SetVisualizationAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	coordinator := new(MockSagaCoordinator)
	visualizationAPI := NewSagaVisualizationAPI(coordinator)

	rm.SetVisualizationAPI(visualizationAPI)
	// Should not panic or error
}

// TestRouteManager_SetAlertsAPI tests SetAlertsAPI method.
func TestRouteManager_SetAlertsAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	// AlertIntegration requires a RecoveryAlertingManager, so we skip actual creation
	// and just test that SetAlertsAPI doesn't panic with nil
	rm.SetAlertsAPI(nil)
	// Should not panic or error
}

// TestRouteManager_SetupSSERoute tests setupSSERoute method.
func TestRouteManager_SetupSSERoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	config := DefaultServerConfig()
	rm := NewRouteManager(router, config)

	collector, _ := NewSagaMetricsCollector(nil)
	pusherConfig := DefaultRealtimePusherConfig()
	pusher := NewRealtimePusher(collector, pusherConfig)

	rm.SetRealtimePusher(pusher)
	// Should not panic or error
}
