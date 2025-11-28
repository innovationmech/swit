/**
 * k6 Load Testing Script for Security Components
 *
 * This script performs comprehensive load testing for the security middleware stack,
 * including JWT authentication, OPA policy evaluation, and OAuth2 token operations.
 *
 * Target Performance Metrics:
 * - 10,000 RPS sustained load
 * - < 15ms P99 latency for full middleware stack
 * - < 1% error rate
 * - < 5% performance impact
 *
 * Usage:
 *   k6 run tests/load/k6-script.js
 *   k6 run --vus 100 --duration 30s tests/load/k6-script.js
 *   k6 run -e BASE_URL=http://localhost:8080 tests/load/k6-script.js
 *
 * Environment Variables:
 *   BASE_URL - Base URL of the service (default: http://localhost:8080)
 *   JWT_SECRET - JWT signing secret for token generation
 *   SCENARIO - Scenario to run: smoke, load, stress, stability (default: load)
 *
 * Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// ============================================================================
// Configuration
// ============================================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const JWT_SECRET = __ENV.JWT_SECRET || 'test-secret-key-for-load-testing';
const SCENARIO = __ENV.SCENARIO || 'load';

// Pre-generated JWT tokens for testing (in production, these would be generated dynamically)
// These are example tokens - in real tests, use properly signed tokens
const TEST_TOKENS = [
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMSIsInVzZXJuYW1lIjoidGVzdHVzZXIxIiwiZW1haWwiOiJ0ZXN0MUBleGFtcGxlLmNvbSIsInJvbGVzIjpbImFkbWluIiwidXNlciJdLCJleHAiOjE3MzU2ODk2MDAsImlhdCI6MTcwNDE1MzYwMH0.placeholder1',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMiIsInVzZXJuYW1lIjoidGVzdHVzZXIyIiwiZW1haWwiOiJ0ZXN0MkBleGFtcGxlLmNvbSIsInJvbGVzIjpbImFkbWluIiwidXNlciJdLCJleHAiOjE3MzU2ODk2MDAsImlhdCI6MTcwNDE1MzYwMH0.placeholder2',
  'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMyIsInVzZXJuYW1lIjoidGVzdHVzZXIzIiwiZW1haWwiOiJ0ZXN0M0BleGFtcGxlLmNvbSIsInJvbGVzIjpbImFkbWluIiwidXNlciJdLCJleHAiOjE3MzU2ODk2MDAsImlhdCI6MTcwNDE1MzYwMH0.placeholder3',
];

// ============================================================================
// Custom Metrics
// ============================================================================

// Counters
const totalRequests = new Counter('security_total_requests');
const successfulRequests = new Counter('security_successful_requests');
const failedRequests = new Counter('security_failed_requests');
const authErrors = new Counter('security_auth_errors');
const policyErrors = new Counter('security_policy_errors');

// Rates
const errorRate = new Rate('security_error_rate');
const authSuccessRate = new Rate('security_auth_success_rate');

// Trends (latency tracking)
const authLatency = new Trend('security_auth_latency', true);
const policyLatency = new Trend('security_policy_latency', true);
const totalLatency = new Trend('security_total_latency', true);

// Gauges
const activeVUs = new Gauge('security_active_vus');

// ============================================================================
// Test Options (Scenarios)
// ============================================================================

const scenarios = {
  // Smoke test: Quick validation that the system works
  smoke: {
    executor: 'constant-vus',
    vus: 1,
    duration: '30s',
  },

  // Load test: Target 10,000 RPS
  load: {
    executor: 'ramping-arrival-rate',
    startRate: 100,
    timeUnit: '1s',
    preAllocatedVUs: 200,
    maxVUs: 500,
    stages: [
      { duration: '30s', target: 1000 },   // Ramp up to 1000 RPS
      { duration: '1m', target: 5000 },    // Ramp up to 5000 RPS
      { duration: '2m', target: 10000 },   // Reach target 10000 RPS
      { duration: '3m', target: 10000 },   // Sustain 10000 RPS
      { duration: '1m', target: 5000 },    // Ramp down
      { duration: '30s', target: 100 },    // Cool down
    ],
  },

  // Stress test: Push beyond normal limits
  stress: {
    executor: 'ramping-arrival-rate',
    startRate: 1000,
    timeUnit: '1s',
    preAllocatedVUs: 500,
    maxVUs: 1000,
    stages: [
      { duration: '30s', target: 5000 },
      { duration: '1m', target: 15000 },   // Beyond target
      { duration: '2m', target: 20000 },   // Stress level
      { duration: '1m', target: 15000 },
      { duration: '30s', target: 5000 },
    ],
  },

  // Stability test: Long-running sustained load
  stability: {
    executor: 'constant-arrival-rate',
    rate: 5000,
    timeUnit: '1s',
    duration: '30m',
    preAllocatedVUs: 200,
    maxVUs: 400,
  },
};

// Select scenario based on environment variable
const selectedScenario = scenarios[SCENARIO] || scenarios.load;

export const options = {
  scenarios: {
    security_load_test: selectedScenario,
  },

  thresholds: {
    // HTTP request thresholds
    http_req_duration: [
      'p(50)<10',     // 50% of requests should be below 10ms
      'p(90)<15',     // 90% of requests should be below 15ms
      'p(99)<30',     // 99% of requests should be below 30ms
      'max<100',      // Maximum should be below 100ms
    ],
    http_req_failed: ['rate<0.01'],  // Error rate should be below 1%

    // Custom security metrics thresholds
    security_error_rate: ['rate<0.01'],
    security_auth_latency: ['p(99)<10'],
    security_policy_latency: ['p(99)<5'],
    security_total_latency: ['p(99)<15'],
    security_auth_success_rate: ['rate>0.99'],
  },

  // Output configuration
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

// ============================================================================
// Setup and Teardown
// ============================================================================

export function setup() {
  console.log(`Starting ${SCENARIO} test against ${BASE_URL}`);
  console.log(`Pre-allocated VUs: ${selectedScenario.preAllocatedVUs || 'N/A'}`);

  // Verify service is available
  const healthCheck = http.get(`${BASE_URL}/health`);
  if (healthCheck.status !== 200) {
    console.warn(`Health check returned status ${healthCheck.status}`);
  }

  return {
    startTime: new Date().toISOString(),
    baseUrl: BASE_URL,
    scenario: SCENARIO,
  };
}

export function teardown(data) {
  console.log(`Test completed. Started at: ${data.startTime}`);
  console.log(`Scenario: ${data.scenario}`);
}

// ============================================================================
// Test Functions
// ============================================================================

// Main test function
export default function (data) {
  activeVUs.add(__VU);

  // Randomly select test scenarios
  const testCase = randomIntBetween(1, 100);

  if (testCase <= 40) {
    testAuthenticatedEndpoint();
  } else if (testCase <= 70) {
    testProtectedResource();
  } else if (testCase <= 85) {
    testTokenValidation();
  } else {
    testPolicyEvaluation();
  }

  // Small sleep to prevent overwhelming the system
  sleep(0.01);
}

// Test authenticated endpoint with JWT
function testAuthenticatedEndpoint() {
  group('Authenticated Endpoint', function () {
    const token = TEST_TOKENS[randomIntBetween(0, TEST_TOKENS.length - 1)];

    const params = {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
        'X-Request-ID': randomString(16),
      },
      tags: { name: 'authenticated_endpoint' },
    };

    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/v1/users/me`, params);
    const duration = Date.now() - startTime;

    totalRequests.add(1);
    totalLatency.add(duration);

    const success = check(response, {
      'status is 200 or 401': (r) => r.status === 200 || r.status === 401,
      'response time < 50ms': (r) => r.timings.duration < 50,
      'has response body': (r) => r.body && r.body.length > 0,
    });

    if (success && response.status === 200) {
      successfulRequests.add(1);
      authSuccessRate.add(1);
      errorRate.add(0);
    } else {
      failedRequests.add(1);
      authSuccessRate.add(0);
      errorRate.add(1);

      if (response.status === 401) {
        authErrors.add(1);
      }
    }

    authLatency.add(response.timings.duration);
  });
}

// Test protected resource with policy evaluation
function testProtectedResource() {
  group('Protected Resource', function () {
    const token = TEST_TOKENS[randomIntBetween(0, TEST_TOKENS.length - 1)];
    const resourceId = randomIntBetween(1, 1000);

    const params = {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
        'X-Request-ID': randomString(16),
      },
      tags: { name: 'protected_resource' },
    };

    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/v1/resources/${resourceId}`, params);
    const duration = Date.now() - startTime;

    totalRequests.add(1);
    totalLatency.add(duration);

    const success = check(response, {
      'status is 200, 401, or 403': (r) => [200, 401, 403].includes(r.status),
      'response time < 50ms': (r) => r.timings.duration < 50,
    });

    if (success && response.status === 200) {
      successfulRequests.add(1);
      errorRate.add(0);
    } else {
      failedRequests.add(1);
      errorRate.add(1);

      if (response.status === 403) {
        policyErrors.add(1);
      }
    }

    policyLatency.add(response.timings.duration);
  });
}

// Test token validation endpoint
function testTokenValidation() {
  group('Token Validation', function () {
    const token = TEST_TOKENS[randomIntBetween(0, TEST_TOKENS.length - 1)];

    const params = {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      tags: { name: 'token_validation' },
    };

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/api/v1/auth/validate`, null, params);
    const duration = Date.now() - startTime;

    totalRequests.add(1);
    authLatency.add(duration);

    const success = check(response, {
      'status is 200 or 401': (r) => r.status === 200 || r.status === 401,
      'response time < 20ms': (r) => r.timings.duration < 20,
    });

    if (success && response.status === 200) {
      successfulRequests.add(1);
      authSuccessRate.add(1);
      errorRate.add(0);
    } else {
      failedRequests.add(1);
      authSuccessRate.add(0);
      errorRate.add(1);
    }
  });
}

// Test policy evaluation endpoint
function testPolicyEvaluation() {
  group('Policy Evaluation', function () {
    const token = TEST_TOKENS[randomIntBetween(0, TEST_TOKENS.length - 1)];

    const payload = JSON.stringify({
      action: 'read',
      resource: {
        type: 'document',
        id: `doc-${randomIntBetween(1, 10000)}`,
      },
    });

    const params = {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      tags: { name: 'policy_evaluation' },
    };

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/api/v1/authz/check`, payload, params);
    const duration = Date.now() - startTime;

    totalRequests.add(1);
    policyLatency.add(duration);

    const success = check(response, {
      'status is 200 or 403': (r) => r.status === 200 || r.status === 403,
      'response time < 30ms': (r) => r.timings.duration < 30,
    });

    if (success) {
      successfulRequests.add(1);
      errorRate.add(0);
    } else {
      failedRequests.add(1);
      errorRate.add(1);
    }
  });
}

// ============================================================================
// Additional Test Scenarios
// ============================================================================

// Batch request test
export function batchRequests() {
  group('Batch Requests', function () {
    const token = TEST_TOKENS[0];

    const requests = [];
    for (let i = 0; i < 10; i++) {
      requests.push({
        method: 'GET',
        url: `${BASE_URL}/api/v1/resources/${randomIntBetween(1, 1000)}`,
        params: {
          headers: {
            'Authorization': `Bearer ${token}`,
          },
        },
      });
    }

    const responses = http.batch(requests);

    check(responses, {
      'all requests completed': (r) => r.length === 10,
      'all requests successful': (r) => r.every((resp) => resp.status === 200 || resp.status === 401),
    });
  });
}

// Health check test
export function healthCheck() {
  const response = http.get(`${BASE_URL}/health`);

  check(response, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 10ms': (r) => r.timings.duration < 10,
  });
}

// ============================================================================
// Helper Functions
// ============================================================================

// Generate a mock JWT token (for testing purposes only)
function generateMockToken(userId, roles) {
  // Note: In production, tokens should be generated by the actual auth service
  // This is a simplified mock for load testing purposes
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(JSON.stringify({
    sub: userId,
    username: `user${userId}`,
    email: `user${userId}@example.com`,
    roles: roles,
    exp: Math.floor(Date.now() / 1000) + 3600,
    iat: Math.floor(Date.now() / 1000),
  }));
  // Note: This signature is not valid - in real tests, use properly signed tokens
  const signature = 'mock_signature';
  return `${header}.${payload}.${signature}`;
}

// Base64 encode (browser-compatible)
function btoa(str) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
  let output = '';

  for (let i = 0; i < str.length; i += 3) {
    const a = str.charCodeAt(i);
    const b = i + 1 < str.length ? str.charCodeAt(i + 1) : 0;
    const c = i + 2 < str.length ? str.charCodeAt(i + 2) : 0;

    const index1 = a >> 2;
    const index2 = ((a & 3) << 4) | (b >> 4);
    const index3 = i + 1 < str.length ? ((b & 15) << 2) | (c >> 6) : 64;
    const index4 = i + 2 < str.length ? c & 63 : 64;

    output += chars[index1] + chars[index2] + chars[index3] + chars[index4];
  }

  return output;
}

// ============================================================================
// Custom Summary Handler
// ============================================================================

export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    scenario: SCENARIO,
    baseUrl: BASE_URL,
    metrics: {
      total_requests: data.metrics.security_total_requests?.values?.count || 0,
      successful_requests: data.metrics.security_successful_requests?.values?.count || 0,
      failed_requests: data.metrics.security_failed_requests?.values?.count || 0,
      error_rate: data.metrics.security_error_rate?.values?.rate || 0,
      auth_success_rate: data.metrics.security_auth_success_rate?.values?.rate || 0,
      latency: {
        avg: data.metrics.http_req_duration?.values?.avg || 0,
        p50: data.metrics.http_req_duration?.values['p(50)'] || 0,
        p90: data.metrics.http_req_duration?.values['p(90)'] || 0,
        p95: data.metrics.http_req_duration?.values['p(95)'] || 0,
        p99: data.metrics.http_req_duration?.values['p(99)'] || 0,
        max: data.metrics.http_req_duration?.values?.max || 0,
      },
      auth_latency: {
        avg: data.metrics.security_auth_latency?.values?.avg || 0,
        p99: data.metrics.security_auth_latency?.values['p(99)'] || 0,
      },
      policy_latency: {
        avg: data.metrics.security_policy_latency?.values?.avg || 0,
        p99: data.metrics.security_policy_latency?.values['p(99)'] || 0,
      },
    },
    thresholds: {
      passed: Object.entries(data.thresholds || {}).filter(([, v]) => v.ok).length,
      failed: Object.entries(data.thresholds || {}).filter(([, v]) => !v.ok).length,
    },
  };

  // Generate report
  const report = `
================================================================================
                        SECURITY LOAD TEST REPORT
================================================================================

Test Configuration:
  - Scenario: ${summary.scenario}
  - Base URL: ${summary.baseUrl}
  - Timestamp: ${summary.timestamp}

Request Summary:
  - Total Requests: ${summary.metrics.total_requests}
  - Successful Requests: ${summary.metrics.successful_requests}
  - Failed Requests: ${summary.metrics.failed_requests}
  - Error Rate: ${(summary.metrics.error_rate * 100).toFixed(4)}%
  - Auth Success Rate: ${(summary.metrics.auth_success_rate * 100).toFixed(2)}%

Latency Metrics (ms):
  - Average: ${summary.metrics.latency.avg.toFixed(2)}
  - P50: ${summary.metrics.latency.p50.toFixed(2)}
  - P90: ${summary.metrics.latency.p90.toFixed(2)}
  - P95: ${summary.metrics.latency.p95.toFixed(2)}
  - P99: ${summary.metrics.latency.p99.toFixed(2)}
  - Max: ${summary.metrics.latency.max.toFixed(2)}

Security-Specific Latency (ms):
  - Auth Latency (P99): ${summary.metrics.auth_latency.p99.toFixed(2)}
  - Policy Latency (P99): ${summary.metrics.policy_latency.p99.toFixed(2)}

Threshold Results:
  - Passed: ${summary.thresholds.passed}
  - Failed: ${summary.thresholds.failed}

Performance Targets:
  [${summary.metrics.latency.p99 <= 15 ? '✓' : '✗'}] P99 Latency < 15ms (Actual: ${summary.metrics.latency.p99.toFixed(2)}ms)
  [${summary.metrics.error_rate <= 0.01 ? '✓' : '✗'}] Error Rate < 1% (Actual: ${(summary.metrics.error_rate * 100).toFixed(4)}%)
  [${summary.metrics.auth_latency.p99 <= 10 ? '✓' : '✗'}] Auth Latency P99 < 10ms (Actual: ${summary.metrics.auth_latency.p99.toFixed(2)}ms)
  [${summary.metrics.policy_latency.p99 <= 5 ? '✓' : '✗'}] Policy Latency P99 < 5ms (Actual: ${summary.metrics.policy_latency.p99.toFixed(2)}ms)

================================================================================
`;

  return {
    'stdout': report,
    'tests/load/results/summary.json': JSON.stringify(summary, null, 2),
    'tests/load/results/report.txt': report,
  };
}


