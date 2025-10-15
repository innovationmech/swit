// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QueryPlan represents a PostgreSQL query execution plan.
type QueryPlan struct {
	// Query is the SQL query that was analyzed
	Query string `json:"query"`

	// Plan is the raw EXPLAIN output
	Plan string `json:"plan,omitempty"`

	// PlanJSON is the JSON-formatted EXPLAIN output
	PlanJSON map[string]interface{} `json:"plan_json,omitempty"`

	// ExecutionTime is the actual execution time (only available with EXPLAIN ANALYZE)
	ExecutionTime float64 `json:"execution_time_ms,omitempty"`

	// PlanningTime is the time spent planning the query
	PlanningTime float64 `json:"planning_time_ms,omitempty"`

	// TotalCost is the estimated total cost of the query
	TotalCost float64 `json:"total_cost,omitempty"`

	// IndexesUsed lists the indexes used by the query
	IndexesUsed []string `json:"indexes_used,omitempty"`

	// HasSeqScan indicates if the query performs a sequential scan
	HasSeqScan bool `json:"has_seq_scan"`

	// Warnings contains any performance warnings
	Warnings []string `json:"warnings,omitempty"`
}

// QueryAnalyzer provides query plan analysis capabilities for PostgreSQL storage.
type QueryAnalyzer struct {
	storage *PostgresStateStorage
}

// NewQueryAnalyzer creates a new query analyzer for the PostgreSQL storage.
func NewQueryAnalyzer(storage *PostgresStateStorage) *QueryAnalyzer {
	return &QueryAnalyzer{
		storage: storage,
	}
}

// ExplainQuery analyzes a query and returns its execution plan.
// It uses EXPLAIN to get the query plan without actually executing the query.
func (qa *QueryAnalyzer) ExplainQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	return qa.explainQueryInternal(ctx, false, false, query, args...)
}

// ExplainQueryJSON analyzes a query and returns its execution plan in JSON format.
func (qa *QueryAnalyzer) ExplainQueryJSON(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	return qa.explainQueryInternal(ctx, false, true, query, args...)
}

// AnalyzeQuery analyzes a query by executing it and returning detailed performance metrics.
// Warning: This actually executes the query, so use with caution on production data.
func (qa *QueryAnalyzer) AnalyzeQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	return qa.explainQueryInternal(ctx, true, false, query, args...)
}

// AnalyzeQueryJSON analyzes a query and returns detailed performance metrics in JSON format.
// Warning: This actually executes the query.
func (qa *QueryAnalyzer) AnalyzeQueryJSON(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	return qa.explainQueryInternal(ctx, true, true, query, args...)
}

// explainQueryInternal performs the actual EXPLAIN or EXPLAIN ANALYZE operation.
func (qa *QueryAnalyzer) explainQueryInternal(ctx context.Context, analyze, jsonFormat bool, query string, args ...interface{}) (*QueryPlan, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	qa.storage.mu.RLock()
	defer qa.storage.mu.RUnlock()

	if err := qa.storage.checkClosed(); err != nil {
		return nil, err
	}

	// Build EXPLAIN query
	explainQuery := "EXPLAIN"
	if analyze {
		explainQuery += " (ANALYZE, BUFFERS, TIMING)"
	}
	if jsonFormat {
		explainQuery += " (FORMAT JSON)"
	}
	explainQuery += " " + query

	// Apply query timeout
	queryCtx, cancel := context.WithTimeout(ctx, qa.storage.config.QueryTimeout)
	defer cancel()

	// Execute EXPLAIN query
	rows, err := qa.storage.db.QueryContext(queryCtx, explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to explain query: %w", err)
	}
	defer rows.Close()

	plan := &QueryPlan{
		Query: query,
	}

	if jsonFormat {
		// Parse JSON format output
		if rows.Next() {
			var jsonStr string
			if err := rows.Scan(&jsonStr); err != nil {
				return nil, fmt.Errorf("failed to scan explain output: %w", err)
			}

			var jsonArray []map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &jsonArray); err != nil {
				return nil, fmt.Errorf("failed to parse explain json: %w", err)
			}

			if len(jsonArray) > 0 {
				plan.PlanJSON = jsonArray[0]
				qa.extractMetricsFromJSON(plan)
			}
		}
	} else {
		// Parse text format output
		var lines []string
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				return nil, fmt.Errorf("failed to scan explain output: %w", err)
			}
			lines = append(lines, line)
		}
		plan.Plan = strings.Join(lines, "\n")
		qa.extractMetricsFromText(plan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading explain output: %w", err)
	}

	// Add warnings based on analysis
	qa.addWarnings(plan)

	return plan, nil
}

// extractMetricsFromJSON extracts key metrics from JSON EXPLAIN output.
func (qa *QueryAnalyzer) extractMetricsFromJSON(plan *QueryPlan) {
	if plan.PlanJSON == nil {
		return
	}

	// Extract execution time
	if execTime, ok := plan.PlanJSON["Execution Time"].(float64); ok {
		plan.ExecutionTime = execTime
	}

	// Extract planning time
	if planTime, ok := plan.PlanJSON["Planning Time"].(float64); ok {
		plan.PlanningTime = planTime
	}

	// Extract plan details
	if planDetails, ok := plan.PlanJSON["Plan"].(map[string]interface{}); ok {
		if totalCost, ok := planDetails["Total Cost"].(float64); ok {
			plan.TotalCost = totalCost
		}

		// Recursively extract indexes used
		plan.IndexesUsed = qa.extractIndexesFromPlan(planDetails)

		// Check for sequential scans
		plan.HasSeqScan = qa.hasSeqScanInPlan(planDetails)
	}
}

// extractMetricsFromText extracts key metrics from text EXPLAIN output.
func (qa *QueryAnalyzer) extractMetricsFromText(plan *QueryPlan) {
	if plan.Plan == "" {
		return
	}

	lines := strings.Split(plan.Plan, "\n")
	for _, line := range lines {
		// Extract cost
		if strings.Contains(line, "cost=") {
			if cost := qa.extractCostFromLine(line); cost > 0 {
				plan.TotalCost = cost
			}
		}

		// Extract execution time
		if strings.Contains(line, "Execution Time:") {
			if execTime := qa.extractTimeFromLine(line); execTime > 0 {
				plan.ExecutionTime = execTime
			}
		}

		// Extract planning time
		if strings.Contains(line, "Planning Time:") {
			if planTime := qa.extractTimeFromLine(line); planTime > 0 {
				plan.PlanningTime = planTime
			}
		}

		// Check for index scan
		if strings.Contains(line, "Index Scan") || strings.Contains(line, "Index Only Scan") ||
			strings.Contains(line, "Bitmap Index Scan") {
			indexName := qa.extractIndexNameFromLine(line)
			if indexName != "" {
				plan.IndexesUsed = append(plan.IndexesUsed, indexName)
			}
		}

		// Check for sequential scan
		if strings.Contains(line, "Seq Scan") {
			plan.HasSeqScan = true
		}
	}
}

// extractIndexesFromPlan recursively extracts index names from the plan JSON.
func (qa *QueryAnalyzer) extractIndexesFromPlan(planNode map[string]interface{}) []string {
	var indexes []string

	// Check node type
	nodeType, _ := planNode["Node Type"].(string)
	if strings.Contains(nodeType, "Index") {
		if indexName, ok := planNode["Index Name"].(string); ok {
			indexes = append(indexes, indexName)
		}
	}

	// Check for plans array (nested nodes)
	if plans, ok := planNode["Plans"].([]interface{}); ok {
		for _, p := range plans {
			if childPlan, ok := p.(map[string]interface{}); ok {
				indexes = append(indexes, qa.extractIndexesFromPlan(childPlan)...)
			}
		}
	}

	return indexes
}

// hasSeqScanInPlan recursively checks if the plan contains a sequential scan.
func (qa *QueryAnalyzer) hasSeqScanInPlan(planNode map[string]interface{}) bool {
	// Check node type
	nodeType, _ := planNode["Node Type"].(string)
	if nodeType == "Seq Scan" {
		return true
	}

	// Check for plans array (nested nodes)
	if plans, ok := planNode["Plans"].([]interface{}); ok {
		for _, p := range plans {
			if childPlan, ok := p.(map[string]interface{}); ok {
				if qa.hasSeqScanInPlan(childPlan) {
					return true
				}
			}
		}
	}

	return false
}

// extractCostFromLine extracts the total cost from an EXPLAIN line.
func (qa *QueryAnalyzer) extractCostFromLine(line string) float64 {
	// Example: "cost=0.00..15.50"
	if idx := strings.Index(line, "cost="); idx >= 0 {
		costStr := line[idx+5:]
		if endIdx := strings.Index(costStr, " "); endIdx >= 0 {
			costStr = costStr[:endIdx]
		}
		if sepIdx := strings.Index(costStr, ".."); sepIdx >= 0 {
			costStr = costStr[sepIdx+2:]
		}
		var cost float64
		fmt.Sscanf(costStr, "%f", &cost)
		return cost
	}
	return 0
}

// extractTimeFromLine extracts execution/planning time from an EXPLAIN line.
func (qa *QueryAnalyzer) extractTimeFromLine(line string) float64 {
	// Example: "Execution Time: 0.123 ms"
	if idx := strings.Index(line, ": "); idx >= 0 {
		timeStr := line[idx+2:]
		timeStr = strings.TrimSpace(timeStr)
		timeStr = strings.TrimSuffix(timeStr, " ms")
		var execTime float64
		fmt.Sscanf(timeStr, "%f", &execTime)
		return execTime
	}
	return 0
}

// extractIndexNameFromLine extracts the index name from an EXPLAIN line.
func (qa *QueryAnalyzer) extractIndexNameFromLine(line string) string {
	// Example: "Index Scan using idx_saga_state on saga_instances"
	if idx := strings.Index(line, " using "); idx >= 0 {
		indexStr := line[idx+7:]
		parts := strings.Fields(indexStr)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// addWarnings adds performance warnings based on the query plan.
func (qa *QueryAnalyzer) addWarnings(plan *QueryPlan) {
	// Warn about sequential scans
	if plan.HasSeqScan {
		plan.Warnings = append(plan.Warnings, "Query performs sequential scan - consider adding appropriate indexes")
	}

	// Warn about high execution time
	if plan.ExecutionTime > 100 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("High execution time: %.2f ms (target: < 100 ms)", plan.ExecutionTime))
	}

	// Warn about high cost
	if plan.TotalCost > 10000 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("High query cost: %.2f", plan.TotalCost))
	}

	// Warn if no indexes used
	if len(plan.IndexesUsed) == 0 && !strings.Contains(strings.ToLower(plan.Query), "insert") &&
		!strings.Contains(strings.ToLower(plan.Query), "update") &&
		!strings.Contains(strings.ToLower(plan.Query), "delete") {
		plan.Warnings = append(plan.Warnings, "No indexes used - query may be slow on large datasets")
	}
}

// AnalyzeCommonQueries analyzes common Saga query patterns and returns their plans.
func (qa *QueryAnalyzer) AnalyzeCommonQueries(ctx context.Context) (map[string]*QueryPlan, error) {
	results := make(map[string]*QueryPlan)

	// Query 1: Get active sagas by state
	query1 := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE state IN (1, 2, 4)
		ORDER BY created_at DESC
		LIMIT 100
	`, qa.storage.instancesTable)

	plan1, err := qa.ExplainQuery(ctx, query1)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query 1: %w", err)
	}
	results["get_active_sagas_by_state"] = plan1

	// Query 2: Get sagas by definition and state
	query2 := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE definition_id = $1 AND state IN (1, 2)
		ORDER BY created_at DESC
		LIMIT 100
	`, qa.storage.instancesTable)

	plan2, err := qa.ExplainQuery(ctx, query2, "test-definition")
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query 2: %w", err)
	}
	results["get_sagas_by_definition_state"] = plan2

	// Query 3: Get sagas by time range
	query3 := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC
		LIMIT 100
	`, qa.storage.instancesTable)

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)
	plan3, err := qa.ExplainQuery(ctx, query3, dayAgo, now)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query 3: %w", err)
	}
	results["get_sagas_by_time_range"] = plan3

	// Query 4: Get sagas with metadata filter
	query4 := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE metadata @> $1::jsonb
		ORDER BY created_at DESC
		LIMIT 100
	`, qa.storage.instancesTable)

	metadataJSON := `{"tenant":"test"}`
	plan4, err := qa.ExplainQuery(ctx, query4, metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query 4: %w", err)
	}
	results["get_sagas_by_metadata"] = plan4

	// Query 5: Get step states for a saga
	query5 := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE saga_id = $1
		ORDER BY step_index ASC
	`, qa.storage.stepsTable)

	plan5, err := qa.ExplainQuery(ctx, query5, "test-saga-id")
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query 5: %w", err)
	}
	results["get_step_states"] = plan5

	// Query 6: Count sagas by state
	query6 := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE state IN (1, 2, 4)
	`, qa.storage.instancesTable)

	plan6, err := qa.ExplainQuery(ctx, query6)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query 6: %w", err)
	}
	results["count_sagas_by_state"] = plan6

	return results, nil
}

// GetIndexUsageStats returns statistics about index usage on Saga tables.
func (qa *QueryAnalyzer) GetIndexUsageStats(ctx context.Context) ([]map[string]interface{}, error) {
	qa.storage.mu.RLock()
	defer qa.storage.mu.RUnlock()

	if err := qa.storage.checkClosed(); err != nil {
		return nil, err
	}

	query := `
		SELECT
			schemaname,
			tablename,
			indexname,
			idx_scan as scans,
			idx_tup_read as tuples_read,
			idx_tup_fetch as tuples_fetched,
			pg_size_pretty(pg_relation_size(indexrelid)) as index_size
		FROM pg_stat_user_indexes
		WHERE tablename IN ('saga_instances', 'saga_steps', 'saga_events')
		ORDER BY idx_scan DESC
	`

	queryCtx, cancel := context.WithTimeout(ctx, qa.storage.config.QueryTimeout)
	defer cancel()

	rows, err := qa.storage.db.QueryContext(queryCtx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get index usage stats: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var schemaname, tablename, indexname, indexSize string
		var scans, tuplesRead, tuplesFetched sql.NullInt64

		err := rows.Scan(&schemaname, &tablename, &indexname, &scans, &tuplesRead, &tuplesFetched, &indexSize)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index stats: %w", err)
		}

		result := map[string]interface{}{
			"schema":         schemaname,
			"table":          tablename,
			"index":          indexname,
			"scans":          scans.Int64,
			"tuples_read":    tuplesRead.Int64,
			"tuples_fetched": tuplesFetched.Int64,
			"size":           indexSize,
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading index stats: %w", err)
	}

	return results, nil
}

// GetTableStats returns statistics about Saga tables.
func (qa *QueryAnalyzer) GetTableStats(ctx context.Context) ([]map[string]interface{}, error) {
	qa.storage.mu.RLock()
	defer qa.storage.mu.RUnlock()

	if err := qa.storage.checkClosed(); err != nil {
		return nil, err
	}

	query := `
		SELECT
			tablename,
			n_live_tup as live_tuples,
			n_dead_tup as dead_tuples,
			last_vacuum,
			last_autovacuum,
			last_analyze,
			last_autoanalyze,
			seq_scan,
			idx_scan
		FROM pg_stat_user_tables
		WHERE tablename IN ('saga_instances', 'saga_steps', 'saga_events')
	`

	queryCtx, cancel := context.WithTimeout(ctx, qa.storage.config.QueryTimeout)
	defer cancel()

	rows, err := qa.storage.db.QueryContext(queryCtx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get table stats: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var tablename string
		var liveTuples, deadTuples, seqScan, idxScan sql.NullInt64
		var lastVacuum, lastAutovacuum, lastAnalyze, lastAutoanalyze sql.NullTime

		err := rows.Scan(&tablename, &liveTuples, &deadTuples, &lastVacuum, &lastAutovacuum,
			&lastAnalyze, &lastAutoanalyze, &seqScan, &idxScan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table stats: %w", err)
		}

		result := map[string]interface{}{
			"table":             tablename,
			"live_tuples":       liveTuples.Int64,
			"dead_tuples":       deadTuples.Int64,
			"seq_scan":          seqScan.Int64,
			"idx_scan":          idxScan.Int64,
			"last_vacuum":       lastVacuum.Time,
			"last_autovacuum":   lastAutovacuum.Time,
			"last_analyze":      lastAnalyze.Time,
			"last_autoanalyze":  lastAutoanalyze.Time,
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading table stats: %w", err)
	}

	return results, nil
}

