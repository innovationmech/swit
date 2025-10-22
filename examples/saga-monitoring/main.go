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

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/innovationmech/swit/pkg/saga/monitoring"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create metrics collector with default configuration
	config := monitoring.DefaultConfig()
	config.EnableLabeledMetrics = true

	collector, err := monitoring.NewSagaMetricsCollector(config)
	if err != nil {
		log.Fatalf("Failed to create metrics collector: %v", err)
	}

	// Start HTTP server for metrics
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", indexHandler(collector))

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting Saga monitoring example on port %s", port)
		log.Printf("Metrics available at http://localhost:%s/metrics", port)
		log.Printf("Dashboard at http://localhost:%s/", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start saga simulation in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go simulateSagaWorkload(ctx, collector)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// healthHandler returns a simple health check response
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// indexHandler returns a simple dashboard page
func indexHandler(collector monitoring.MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := collector.GetMetrics()

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Saga Monitoring Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 1200px;
            margin: 50px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 10px;
        }
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }
        .metric-card {
            background-color: #f9f9f9;
            padding: 20px;
            border-radius: 6px;
            border-left: 4px solid #4CAF50;
        }
        .metric-label {
            color: #666;
            font-size: 14px;
            margin-bottom: 5px;
        }
        .metric-value {
            color: #333;
            font-size: 32px;
            font-weight: bold;
        }
        .metric-unit {
            color: #999;
            font-size: 14px;
        }
        .links {
            margin-top: 30px;
            padding: 20px;
            background-color: #e8f5e9;
            border-radius: 6px;
        }
        .links a {
            display: inline-block;
            margin: 5px 10px;
            padding: 10px 20px;
            background-color: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 4px;
        }
        .links a:hover {
            background-color: #45a049;
        }
        .failure-reasons {
            margin-top: 20px;
        }
        .failure-item {
            padding: 10px;
            margin: 5px 0;
            background-color: #ffebee;
            border-left: 4px solid #f44336;
            border-radius: 4px;
        }
        .success-rate {
            color: %s;
        }
    </style>
    <meta http-equiv="refresh" content="5">
</head>
<body>
    <div class="container">
        <h1>🎯 Saga Monitoring Example</h1>
        
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Sagas Started</div>
                <div class="metric-value">%d</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Sagas Completed</div>
                <div class="metric-value success-rate">%d</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Sagas Failed</div>
                <div class="metric-value" style="color: #f44336;">%d</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Active Sagas</div>
                <div class="metric-value" style="color: #2196F3;">%d</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Success Rate</div>
                <div class="metric-value success-rate">%.2f%%</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Avg Duration</div>
                <div class="metric-value">%.2f<span class="metric-unit">s</span></div>
            </div>
        </div>
        
        <div class="failure-reasons">
            <h2>Failure Reasons</h2>
            %s
        </div>
        
        <div class="links">
            <h2>📊 Monitoring Links</h2>
            <a href="/metrics" target="_blank">Prometheus Metrics</a>
            <a href="http://localhost:9090" target="_blank">Prometheus UI</a>
            <a href="http://localhost:3000" target="_blank">Grafana Dashboard</a>
            <a href="http://localhost:16686" target="_blank">Jaeger UI</a>
            <a href="http://localhost:9093" target="_blank">Alertmanager</a>
        </div>
        
        <p style="margin-top: 20px; color: #666;">
            <small>Last updated: %s | Auto-refresh every 5 seconds</small>
        </p>
    </div>
</body>
</html>
`, getSuccessRateColor(metrics),
			metrics.SagasStarted,
			metrics.SagasCompleted,
			metrics.SagasFailed,
			metrics.ActiveSagas,
			getSuccessRate(metrics),
			metrics.AvgDuration,
			formatFailureReasons(metrics.FailureReasons),
			time.Now().Format(time.RFC3339))
	}
}

func getSuccessRate(metrics *monitoring.Metrics) float64 {
	if metrics.SagasStarted == 0 {
		return 0
	}
	return float64(metrics.SagasCompleted) / float64(metrics.SagasStarted) * 100
}

func getSuccessRateColor(metrics *monitoring.Metrics) string {
	rate := getSuccessRate(metrics)
	if rate >= 99 {
		return "#4CAF50" // Green
	} else if rate >= 95 {
		return "#FF9800" // Orange
	}
	return "#f44336" // Red
}

func formatFailureReasons(reasons map[string]int64) string {
	if len(reasons) == 0 {
		return `<div class="failure-item">No failures recorded</div>`
	}

	html := ""
	for reason, count := range reasons {
		html += fmt.Sprintf(`<div class="failure-item"><strong>%s:</strong> %d occurrences</div>`, reason, count)
	}
	return html
}

// simulateSagaWorkload simulates saga execution for demonstration purposes
func simulateSagaWorkload(ctx context.Context, collector monitoring.MetricsCollector) {
	log.Println("Starting saga workload simulation...")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	sagaID := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping saga workload simulation")
			return
		case <-ticker.C:
			// Simulate multiple saga executions
			numSagas := rand.Intn(3) + 1
			for i := 0; i < numSagas; i++ {
				sagaID++
				go executeSaga(fmt.Sprintf("saga-%d", sagaID), collector)
			}
		}
	}
}

// executeSaga simulates a single saga execution
func executeSaga(sagaID string, collector monitoring.MetricsCollector) {
	// Record saga start
	collector.RecordSagaStarted(sagaID)

	// Simulate saga execution time (1-10 seconds)
	executionTime := time.Duration(rand.Intn(9000)+1000) * time.Millisecond
	time.Sleep(executionTime)

	// Randomly decide success or failure (90% success rate)
	if rand.Float64() < 0.90 {
		// Success
		collector.RecordSagaCompleted(sagaID, executionTime)
	} else {
		// Failure - randomly choose a failure reason
		failureReasons := []string{
			"timeout",
			"network_error",
			"database_error",
			"validation_error",
			"downstream_service_unavailable",
			"insufficient_resources",
		}
		reason := failureReasons[rand.Intn(len(failureReasons))]
		collector.RecordSagaFailed(sagaID, reason)
	}
}
