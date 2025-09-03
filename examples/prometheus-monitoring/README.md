# Comprehensive Prometheus Monitoring Example

This example demonstrates a complete monitoring stack with Prometheus metrics collection, Grafana visualization, and Alertmanager for alerting. It showcases advanced business metrics, custom KPIs, and comprehensive monitoring best practices.

## 🏗️ Architecture

```text
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Business App   │────│   Prometheus    │────│    Grafana      │
│  (Port 8080)    │    │  (Port 9090)    │    │  (Port 3000)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         └────────────────────────┼────────────────────────┘
                                  │
                         ┌─────────────────┐
                         │  Alertmanager   │
                         │  (Port 9093)    │
                         └─────────────────┘
```

## 📊 Business Metrics Collected

### E-commerce KPIs
- **Revenue Metrics**: Total revenue, regional revenue, average order value
- **Order Metrics**: Orders created, order value distribution, order processing times
- **User Metrics**: Active users, conversion rates, regional user activity
- **Payment Metrics**: Payment success/failure rates, payment method usage
- **Inventory Metrics**: Stock levels, low stock alerts

### Performance Metrics
- **Request Metrics**: Response times, request rates, error rates
- **Cache Metrics**: Hit/miss ratios, cache performance
- **Queue Metrics**: Processing queue sizes, job processing times
- **System Metrics**: CPU usage, memory usage, connections

### Custom Business Dimensions
- **Regional Segmentation**: Performance by geographic region
- **User Segmentation**: Premium, standard, basic user tiers
- **Product Categories**: Electronics, accessories, other
- **Payment Methods**: Credit card, PayPal, digital wallets

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```bash
# Start the complete monitoring stack
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the stack
docker-compose down
```

### Manual Setup

```bash
# Run the business service
go run main.go

# Access the service
curl http://localhost:8080/api/v1/health
```

## 🎯 Endpoints

### Business API Endpoints

#### E-commerce Operations
```bash
# Create an order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "product_id": "laptop",
    "quantity": 2,
    "price": 999.99,
    "region": "us-east-1"
  }'

# Process a payment
curl -X POST http://localhost:8080/api/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "order_123",
    "method": "credit_card",
    "amount": 1999.98
  }'

# View product details
curl http://localhost:8080/api/v1/products/laptop?user_id=user123

# User login
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "region": "us-east-1"
  }'
```

#### Analytics Endpoints
```bash
# Revenue analytics
curl http://localhost:8080/api/v1/analytics/revenue

# User analytics
curl http://localhost:8080/api/v1/analytics/users

# Product analytics
curl http://localhost:8080/api/v1/analytics/products
```

## 📈 Accessing the Monitoring Stack

### Prometheus
- **URL**: http://localhost:9090
- **Purpose**: Metrics collection and querying
- **Key Queries**:
  ```promql
  # Total orders per hour
  rate(swit_business_orders_created_total[1h]) * 3600
  
  # Average order value by region
  avg by (region) (swit_business_order_value_dollars)
  
  # Cache hit rate
  swit_business_cache_hit_rate * 100
  
  # Payment failure rate
  rate(swit_business_payments_processed_total{status="failed"}[5m])
  ```

### Grafana
- **URL**: http://localhost:3000
- **Login**: admin/admin
- **Dashboards**: Pre-configured business metrics dashboard
- **Features**:
  - Revenue overview and trends
  - Regional performance comparison
  - Payment method distribution
  - Inventory monitoring
  - Cache performance metrics

### Alertmanager
- **URL**: http://localhost:9093
- **Purpose**: Alert routing and notification
- **Alerts Configured**:
  - High order error rates
  - Low conversion rates
  - Inventory low stock warnings
  - Payment processing issues
  - System performance degradation

## 🔔 Alert Rules

### Business KPI Alerts
```yaml
# High order error rate
- alert: HighOrderErrorRate
  expr: rate(swit_business_order_errors_total[5m]) > 0.1
  
# Low conversion rate
- alert: LowConversionRate
  expr: avg(swit_business_conversion_rate_percent) < 2.0
  
# Low inventory
- alert: InventoryLowStock
  expr: swit_business_inventory_levels < 10
```

### Performance Alerts
```yaml
# High request latency
- alert: HighRequestLatency
  expr: histogram_quantile(0.95, rate(swit_business_request_duration_seconds_bucket[5m])) > 1.0
  
# Low cache hit rate
- alert: LowCacheHitRate
  expr: swit_business_cache_hit_rate < 0.8
```

## 📊 Sample Metrics Collected

### Counter Metrics
```text
swit_business_orders_created_total{region="us-east-1",product="laptop"} 145
swit_business_payments_processed_total{method="credit_card",status="success"} 892
swit_business_user_logins_total{status="success",region="us-east-1"} 1203
```

### Gauge Metrics
```text
swit_business_active_users_total 1547
swit_business_inventory_levels{product="laptop",warehouse="us-east-1"} 89
swit_business_cache_hit_rate{cache_type="product_details"} 0.85
```

### Histogram Metrics
```text
swit_business_order_value_dollars_bucket{le="100.0"} 234
swit_business_request_duration_seconds_bucket{endpoint="/api/v1/orders"} 1456
swit_business_payment_processing_duration_seconds_bucket{method="credit_card"} 892
```

## 🎛️ Configuration

### Service Configuration (`swit.yaml`)
```yaml
prometheus:
  enabled: true
  endpoint: "/metrics"
  namespace: "swit"
  subsystem: "business"
  cardinality_limit: 50000
  buckets:
    duration: [0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10, 30, 60]
    size: [1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000]
```

### Prometheus Configuration
- Scrape interval: 5s for business metrics
- Retention: 30 days
- Storage: 1GB limit
- Alerting rules enabled

### Environment Variables
```bash
HTTP_PORT=8080
PROMETHEUS_ENABLED=true
CONFIG_PATH=swit.yaml
```

## 🧪 Load Testing

Generate realistic load to see metrics in action:

```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Generate order load
hey -n 1000 -c 10 -m POST -H "Content-Type: application/json" \
  -d '{"user_id":"user123","product_id":"laptop","quantity":1,"price":999.99,"region":"us-east-1"}' \
  http://localhost:8080/api/v1/orders

# Generate payment load
hey -n 500 -c 5 -m POST -H "Content-Type: application/json" \
  -d '{"order_id":"order_123","method":"credit_card","amount":999.99}' \
  http://localhost:8080/api/v1/payments

# Generate product view load
hey -n 2000 -c 20 \
  http://localhost:8080/api/v1/products/laptop?user_id=user123
```

## 🚨 Monitoring Best Practices Demonstrated

### 1. Metric Naming Convention
- **Namespace**: `swit_business_*`
- **Clear naming**: `orders_created_total`, `payment_processing_duration_seconds`
- **Consistent labels**: `region`, `product`, `method`, `status`

### 2. Cardinality Management
- Limited label values to prevent metric explosion
- User segmentation instead of individual user IDs
- Product categories instead of individual products

### 3. Business Context
- Metrics tied to business KPIs
- Regional and user segment breakdown
- Revenue and conversion tracking

### 4. Alert Strategy
- Business-impact focused alerts
- Different severity levels
- Alert fatigue prevention with inhibit rules

### 5. Dashboard Design
- Executive overview with key KPIs
- Operational dashboards for different teams
- Drill-down capabilities

## 🛠️ Development

### Adding New Metrics

1. **Define the metric in the handler**:
```go
h.trackMetric("new_business_metric_total", 1, map[string]string{
    "category": "example",
    "region": region,
})
```

2. **Add alerting rules** in `alerting/rules.yml`

3. **Update Grafana dashboard** in `grafana/dashboards/`

### Testing Metrics

```bash
# Check metric availability
curl http://localhost:8080/metrics | grep swit_business

# Validate Prometheus targets
curl http://localhost:9090/api/v1/targets

# Test alert rules
curl http://localhost:9090/api/v1/rules
```

## 📚 Further Reading

- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Grafana Dashboard Design](https://grafana.com/docs/grafana/latest/dashboards/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Business Metrics Strategy](https://sre.google/workbook/implementing-slos/)

## 🤝 Integration

This monitoring setup can be integrated with:
- **Kubernetes**: Using ServiceMonitor and PrometheusRule CRDs
- **Cloud Platforms**: AWS CloudWatch, GCP Monitoring, Azure Monitor
- **APM Tools**: Jaeger for tracing, New Relic for additional insights
- **Notification Systems**: Slack, PagerDuty, email, webhooks

## 🔍 Troubleshooting

### Common Issues

1. **Metrics not appearing**:
   - Check service is running: `curl http://localhost:8080/metrics`
   - Verify Prometheus targets: `http://localhost:9090/targets`

2. **High cardinality warnings**:
   - Review metric labels for high-cardinality values
   - Adjust `cardinality_limit` in configuration

3. **Grafana dashboards not loading**:
   - Check datasource configuration
   - Verify Prometheus connectivity

4. **Alerts not firing**:
   - Check alerting rules syntax
   - Verify Alertmanager configuration
   - Test with `amtool` CLI tool

This comprehensive example demonstrates production-ready monitoring with business-focused metrics, providing a foundation for monitoring real-world e-commerce and business applications.
