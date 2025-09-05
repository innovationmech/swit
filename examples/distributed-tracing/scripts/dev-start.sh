#!/bin/bash

# 分布式追踪开发环境启动脚本
# Copyright (c) 2024 SWIT Framework Authors

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/scripts/config/demo-config.env"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo -e "${PURPLE}[DEV-START]${NC} $1"
}

log_dev() {
    echo -e "${CYAN}[DEV]${NC} $1"
}

# 显示欢迎信息
show_welcome() {
    echo "======================================================================"
    echo -e "${PURPLE}🚀 SWIT 框架分布式追踪开发环境启动器${NC}"
    echo "======================================================================"
    echo -e "${YELLOW}本脚本提供完整的开发环境搭建和管理：${NC}"
    echo "  🔧 自动环境检查和依赖安装"
    echo "  📦 智能服务编排和热重载"
    echo "  📊 实时监控和日志聚合"
    echo "  🔍 开发工具集成和调试支持"
    echo "  🎯 一键测试数据生成和验证"
    echo
}

# 显示使用说明
show_usage() {
    echo "Usage: $0 [options] [services...]"
    echo
    echo "Options:"
    echo "  -h, --help              显示帮助信息"
    echo "  -m, --mode=MODE         开发模式 (默认: full)"
    echo "                          可选: minimal, standard, full, debug"
    echo "  --hot-reload            启用代码热重载 (需要支持的服务)"
    echo "  --debug-port=PORT       开启调试端口 (默认: 不启用)"
    echo "  --profile               启用性能分析"
    echo "  --mock-external         模拟外部依赖"
    echo "  --seed-data             自动生成种子数据"
    echo "  --auto-test             启动后自动运行测试"
    echo "  --dashboard             启动开发者仪表板"
    echo "  --logs-tail             实时显示服务日志"
    echo "  --wait-time=SECONDS     服务启动等待时间 (默认: 60秒)"
    echo "  --cleanup               启动前清理旧数据"
    echo "  --verbose               详细输出"
    echo "  --quiet                 静默模式"
    echo
    echo "Development Modes:"
    echo "  minimal     只启动 Jaeger (用于外部服务开发)"
    echo "  standard    启动 Jaeger + 核心服务"
    echo "  full        启动所有服务 + 监控 + 工具"
    echo "  debug       完整环境 + 调试配置 + 详细日志"
    echo
    echo "Services (可选，默认根据模式启动):"
    echo "  jaeger              Jaeger 追踪后端"
    echo "  order-service       订单服务"
    echo "  payment-service     支付服务"
    echo "  inventory-service   库存服务"
    echo "  monitoring          监控服务 (Prometheus, Grafana)"
    echo
    echo "Examples:"
    echo "  $0                                    # 完整开发环境"
    echo "  $0 -m minimal --hot-reload           # 最小环境，启用热重载"
    echo "  $0 -m debug --debug-port=5005        # 调试模式，开启调试端口"
    echo "  $0 jaeger order-service --seed-data  # 只启动指定服务并生成测试数据"
    echo "  $0 --dashboard --logs-tail            # 启用仪表板和日志跟踪"
}

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        log_info "正在加载配置文件: $CONFIG_FILE"
        source "$CONFIG_FILE"
    else
        log_warning "配置文件不存在: $CONFIG_FILE，创建默认配置"
        create_default_config
    fi
    
    # 开发环境默认配置
    export JAEGER_UI_PORT=${JAEGER_UI_PORT:-16686}
    export JAEGER_COLLECTOR_PORT=${JAEGER_COLLECTOR_PORT:-14268}
    export ORDER_SERVICE_PORT=${ORDER_SERVICE_PORT:-8081}
    export PAYMENT_SERVICE_PORT=${PAYMENT_SERVICE_PORT:-8082}
    export INVENTORY_SERVICE_PORT=${INVENTORY_SERVICE_PORT:-8083}
    export DEV_DASHBOARD_PORT=${DEV_DASHBOARD_PORT:-3000}
}

# 创建默认配置
create_default_config() {
    mkdir -p "$(dirname "$CONFIG_FILE")"
    
    cat > "$CONFIG_FILE" << 'EOF'
# SWIT 框架分布式追踪开发环境配置
# Copyright (c) 2024 SWIT Framework Authors

# 服务端口配置
JAEGER_UI_PORT=16686
JAEGER_COLLECTOR_PORT=14268
ORDER_SERVICE_PORT=8081
PAYMENT_SERVICE_PORT=8082
INVENTORY_SERVICE_PORT=8083

# 开发环境配置
DEV_DASHBOARD_PORT=3000
DEBUG_PORT=5005
LOG_LEVEL=debug
HOT_RELOAD_ENABLED=true

# Jaeger 配置
JAEGER_ENDPOINT=http://localhost:14268/api/traces
JAEGER_SAMPLING_RATE=1.0
JAEGER_AGENT_HOST=localhost
JAEGER_AGENT_PORT=6831

# 数据库配置 (开发环境)
DB_HOST=localhost
DB_PORT=5432
DB_NAME=swit_dev
DB_USER=swit_dev
DB_PASSWORD=dev_password

# Redis 配置 (开发环境)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# 开发工具配置
ENABLE_PROFILING=true
ENABLE_METRICS=true
GENERATE_SEED_DATA=true
AUTO_TEST_ON_START=false

# 日志配置
LOG_FORMAT=json
LOG_OUTPUT=stdout
MAX_LOG_SIZE=100MB
MAX_LOG_BACKUPS=5
EOF
    
    log_success "已创建默认配置文件: $CONFIG_FILE"
}

# 检查开发环境依赖
check_dev_dependencies() {
    log_header "检查开发环境依赖"
    
    local missing_tools=()
    local optional_missing=()
    
    # 基础工具
    local required_tools=("docker" "docker-compose" "curl" "jq" "bc")
    for tool in "${required_tools[@]}"; do
        if ! command -v $tool &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    # 开发工具
    local dev_tools=("git" "make" "go" "node" "python3")
    for tool in "${dev_tools[@]}"; do
        if ! command -v $tool &> /dev/null; then
            optional_missing+=("$tool")
        fi
    done
    
    # 性能和分析工具
    local analysis_tools=("hey" "wrk" "ab" "htop" "iostat")
    for tool in "${analysis_tools[@]}"; do
        if ! command -v $tool &> /dev/null; then
            optional_missing+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "缺少必需工具: ${missing_tools[*]}"
        echo
        echo "安装方法："
        echo "  macOS: brew install ${missing_tools[*]}"
        echo "  Ubuntu: sudo apt-get install ${missing_tools[*]}"
        echo "  CentOS: sudo yum install ${missing_tools[*]}"
        exit 1
    fi
    
    if [ ${#optional_missing[@]} -gt 0 ] && [ "$VERBOSE" = "true" ]; then
        log_warning "缺少可选开发工具: ${optional_missing[*]}"
        log_info "这些工具可以提供更好的开发体验"
    fi
    
    # 检查 Docker 服务状态
    if ! docker info &> /dev/null; then
        log_error "Docker 服务未运行"
        log_info "请启动 Docker 服务后重试"
        exit 1
    fi
    
    # 检查 Docker Compose 版本
    local compose_version
    if command -v docker-compose &> /dev/null; then
        compose_version=$(docker-compose --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
    elif docker compose version &> /dev/null; then
        compose_version=$(docker compose version --short)
    else
        log_error "Docker Compose 未安装或版本不兼容"
        exit 1
    fi
    
    log_success "依赖检查完成"
    if [ "$VERBOSE" = "true" ]; then
        log_info "Docker Compose 版本: $compose_version"
    fi
}

# 环境清理
cleanup_environment() {
    if [ "$CLEANUP" != "true" ]; then
        return 0
    fi
    
    log_header "清理开发环境"
    
    # 停止现有服务
    cd "$PROJECT_ROOT"
    
    if docker-compose ps -q | grep -q .; then
        log_info "停止现有服务..."
        docker-compose down -v 2>/dev/null || true
    fi
    
    # 清理旧的容器和镜像
    local old_containers
    old_containers=$(docker ps -a --filter "name=distributed-tracing" --filter "name=order-service" --filter "name=payment-service" --filter "name=inventory-service" --filter "name=jaeger" -q)
    
    if [ -n "$old_containers" ]; then
        log_info "清理旧容器..."
        docker rm -f $old_containers 2>/dev/null || true
    fi
    
    # 清理未使用的资源
    log_info "清理未使用的 Docker 资源..."
    docker system prune -f --volumes &> /dev/null || true
    
    # 清理日志目录
    if [ -d "$PROJECT_ROOT/logs" ]; then
        rm -rf "$PROJECT_ROOT/logs"
        log_info "已清理日志目录"
    fi
    
    # 清理临时文件
    find "$PROJECT_ROOT" -name "*.tmp" -o -name "*.log.*" -o -name "core.*" -delete 2>/dev/null || true
    
    log_success "环境清理完成"
}

# 准备开发环境
prepare_dev_environment() {
    log_header "准备开发环境"
    
    cd "$PROJECT_ROOT"
    
    # 创建必要的目录
    local dev_dirs=("logs" "tmp" "reports" "benchmark-results")
    for dir in "${dev_dirs[@]}"; do
        mkdir -p "$dir"
    done
    
    # 创建开发用的 docker-compose 覆盖文件
    if [ ! -f "docker-compose.dev.yml" ]; then
        create_dev_compose_override
    fi
    
    # 设置环境变量文件
    if [ ! -f ".env" ]; then
        create_dev_env_file
    fi
    
    # 准备开发数据库
    if [ "$MODE" = "full" ] || [ "$MODE" = "debug" ]; then
        prepare_dev_database
    fi
    
    log_success "开发环境准备完成"
}

# 创建开发环境 Docker Compose 覆盖文件
create_dev_compose_override() {
    log_info "创建开发环境 Docker Compose 覆盖配置..."
    
    cat > "docker-compose.dev.yml" << 'EOF'
# 开发环境 Docker Compose 覆盖配置
version: '3.8'

services:
  jaeger:
    environment:
      - JAEGER_DISABLED=false
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
      - SPAN_STORAGE_TYPE=memory
      - QUERY_MAX_TRACES=10000
    ports:
      - "16686:16686"  # Jaeger UI
      - "16687:16687"  # Jaeger gRPC
      - "14268:14268"  # Jaeger collector HTTP
      - "14250:14250"  # Jaeger gRPC collector
      - "6831:6831/udp"  # Jaeger agent UDP
      - "6832:6832/udp"  # Jaeger agent UDP binary

  order-service:
    build:
      context: ./services/order-service
      dockerfile: Dockerfile.dev
    volumes:
      - ./services/order-service:/app
      - go-mod-cache:/go/pkg/mod
    environment:
      - ENV=development
      - LOG_LEVEL=debug
      - HOT_RELOAD=true
    ports:
      - "8081:8080"
      - "8091:8090"  # metrics
    depends_on:
      - jaeger
      - dev-db

  payment-service:
    build:
      context: ./services/payment-service
      dockerfile: Dockerfile.dev
    volumes:
      - ./services/payment-service:/app
      - go-mod-cache:/go/pkg/mod
    environment:
      - ENV=development
      - LOG_LEVEL=debug
      - HOT_RELOAD=true
    ports:
      - "8082:8080"
      - "8092:8090"  # metrics
    depends_on:
      - jaeger
      - dev-db

  inventory-service:
    build:
      context: ./services/inventory-service
      dockerfile: Dockerfile.dev
    volumes:
      - ./services/inventory-service:/app
      - go-mod-cache:/go/pkg/mod
    environment:
      - ENV=development
      - LOG_LEVEL=debug
      - HOT_RELOAD=true
    ports:
      - "8083:8080"
      - "8093:8090"  # metrics
    depends_on:
      - jaeger
      - dev-db

  # 开发数据库
  dev-db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=swit_dev
      - POSTGRES_USER=swit_dev
      - POSTGRES_PASSWORD=dev_password
    ports:
      - "5432:5432"
    volumes:
      - dev-db-data:/var/lib/postgresql/data
      - ./scripts/sql:/docker-entrypoint-initdb.d

  # 开发 Redis
  dev-redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - dev-redis-data:/data

  # 开发仪表板 (可选)
  dev-dashboard:
    image: portainer/portainer-ce:latest
    ports:
      - "3000:9000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer-data:/data
    command: --admin-password='$$2y$$10$$N6Q2jOG.KYqpPzOOd5CIjOGhcNj7bF3NG5zTkdWy7Q4qxPjCg1jR.'

  # Prometheus (监控)
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'

  # Grafana (可视化)
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources

volumes:
  go-mod-cache:
  dev-db-data:
  dev-redis-data:
  portainer-data:
  prometheus-data:
  grafana-data:

networks:
  default:
    name: swit-dev-network
EOF
    
    log_success "开发环境 Docker Compose 覆盖文件已创建"
}

# 创建开发环境变量文件
create_dev_env_file() {
    log_info "创建开发环境变量文件..."
    
    cat > ".env" << EOF
# 开发环境变量
COMPOSE_PROJECT_NAME=swit-dev
COMPOSE_FILE=docker-compose.yml:docker-compose.dev.yml

# 服务端口
JAEGER_UI_PORT=16686
ORDER_SERVICE_PORT=8081
PAYMENT_SERVICE_PORT=8082
INVENTORY_SERVICE_PORT=8083
DEV_DASHBOARD_PORT=3000

# 开发模式配置
ENV=development
LOG_LEVEL=debug
HOT_RELOAD=true
DEBUG_PORT=${DEBUG_PORT:-5005}

# 数据库配置
DB_HOST=dev-db
DB_PORT=5432
DB_NAME=swit_dev
DB_USER=swit_dev
DB_PASSWORD=dev_password

# Redis 配置
REDIS_HOST=dev-redis
REDIS_PORT=6379

# Jaeger 配置
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
JAEGER_SAMPLING_RATE=1.0
JAEGER_AGENT_HOST=jaeger
JAEGER_AGENT_PORT=6831

# 开发工具
ENABLE_PROFILING=${PROFILE:-false}
ENABLE_METRICS=true
GENERATE_SEED_DATA=${SEED_DATA:-false}
AUTO_TEST_ON_START=${AUTO_TEST:-false}
EOF
    
    log_success "开发环境变量文件已创建"
}

# 准备开发数据库
prepare_dev_database() {
    log_info "准备开发数据库初始化脚本..."
    
    mkdir -p "$PROJECT_ROOT/scripts/sql"
    
    # 创建数据库初始化脚本
    cat > "$PROJECT_ROOT/scripts/sql/01-init.sql" << 'EOF'
-- 开发环境数据库初始化脚本
-- Copyright (c) 2024 SWIT Framework Authors

-- 创建开发用户和权限
CREATE USER IF NOT EXISTS 'dev_user'@'%' IDENTIFIED BY 'dev_password';
GRANT ALL PRIVILEGES ON swit_dev.* TO 'dev_user'@'%';
FLUSH PRIVILEGES;

-- 创建基础表结构 (示例)
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    customer_id VARCHAR(100) NOT NULL,
    product_id VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    trace_id VARCHAR(100),
    INDEX idx_customer_id (customer_id),
    INDEX idx_status (status),
    INDEX idx_trace_id (trace_id)
);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    amount DECIMAL(10,2) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    transaction_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    trace_id VARCHAR(100),
    INDEX idx_order_id (order_id),
    INDEX idx_status (status),
    INDEX idx_trace_id (trace_id)
);

CREATE TABLE IF NOT EXISTS inventory (
    id SERIAL PRIMARY KEY,
    product_id VARCHAR(100) UNIQUE NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    reserved INTEGER NOT NULL DEFAULT 0,
    price DECIMAL(10,2) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_product_id (product_id)
);
EOF

    # 创建种子数据脚本
    if [ "$SEED_DATA" = "true" ]; then
        cat > "$PROJECT_ROOT/scripts/sql/02-seed-data.sql" << 'EOF'
-- 开发环境种子数据
-- Copyright (c) 2024 SWIT Framework Authors

-- 插入测试产品数据
INSERT INTO inventory (product_id, quantity, price) VALUES
('product-001', 1000, 99.99),
('product-002', 500, 149.99),
('product-003', 200, 299.99),
('product-004', 2000, 49.99),
('product-005', 800, 199.99)
ON DUPLICATE KEY UPDATE 
    quantity = VALUES(quantity),
    price = VALUES(price);

-- 插入测试订单数据
INSERT INTO orders (customer_id, product_id, quantity, amount, status, trace_id) VALUES
('customer-001', 'product-001', 2, 199.98, 'completed', 'trace-dev-001'),
('customer-002', 'product-002', 1, 149.99, 'pending', 'trace-dev-002'),
('customer-003', 'product-003', 1, 299.99, 'completed', 'trace-dev-003');

-- 插入对应的支付数据
INSERT INTO payments (order_id, amount, payment_method, status, transaction_id, trace_id) VALUES
(1, 199.98, 'credit_card', 'completed', 'txn-dev-001', 'trace-dev-001'),
(3, 299.99, 'paypal', 'completed', 'txn-dev-003', 'trace-dev-003');
EOF
    fi
    
    log_success "开发数据库脚本已准备"
}

# 确定服务启动列表
determine_services() {
    local services=()
    
    if [ ${#SELECTED_SERVICES[@]} -gt 0 ]; then
        services=("${SELECTED_SERVICES[@]}")
        log_info "使用指定的服务: ${services[*]}"
    else
        case "$MODE" in
            "minimal")
                services=("jaeger")
                ;;
            "standard")
                services=("jaeger" "order-service" "payment-service" "inventory-service")
                ;;
            "full")
                services=("jaeger" "order-service" "payment-service" "inventory-service" "dev-db" "dev-redis")
                if [ "$DASHBOARD" = "true" ]; then
                    services+=("dev-dashboard")
                fi
                ;;
            "debug")
                services=("jaeger" "order-service" "payment-service" "inventory-service" "dev-db" "dev-redis" "prometheus" "grafana")
                if [ "$DASHBOARD" = "true" ]; then
                    services+=("dev-dashboard")
                fi
                ;;
        esac
        log_info "根据模式 '$MODE' 确定服务: ${services[*]}"
    fi
    
    printf '%s\n' "${services[@]}"
}

# 启动服务
start_services() {
    log_header "启动开发环境服务"
    
    cd "$PROJECT_ROOT"
    
    local services
    mapfile -t services < <(determine_services)
    
    # 构建 Docker Compose 命令
    local compose_files=("docker-compose.yml")
    
    # 添加开发环境覆盖文件
    if [ -f "docker-compose.dev.yml" ]; then
        compose_files+=("docker-compose.dev.yml")
    fi
    
    local compose_cmd="docker-compose"
    if command -v docker &> /dev/null && docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    fi
    
    # 构建完整命令
    local full_cmd="$compose_cmd"
    for file in "${compose_files[@]}"; do
        full_cmd="$full_cmd -f $file"
    done
    
    # 构建或拉取镜像
    log_info "准备服务镜像..."
    if $full_cmd build "${services[@]}" 2>/dev/null || true; then
        log_success "服务镜像构建完成"
    else
        log_info "拉取预构建镜像..."
        $full_cmd pull "${services[@]}" 2>/dev/null || true
    fi
    
    # 启动服务
    log_info "启动服务: ${services[*]}"
    if [ "$VERBOSE" = "true" ]; then
        $full_cmd up -d "${services[@]}"
    else
        $full_cmd up -d "${services[@]}" > /dev/null 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        log_success "服务启动完成"
    else
        log_error "服务启动失败"
        return 1
    fi
}

# 等待服务就绪
wait_for_services() {
    log_header "等待服务就绪"
    
    local start_time=$(date +%s)
    local timeout=${WAIT_TIME:-60}
    local all_ready=false
    
    # 定义健康检查端点
    local health_endpoints=(
        "http://localhost:$JAEGER_UI_PORT:Jaeger UI"
    )
    
    # 根据启动的服务添加健康检查
    mapfile -t started_services < <(determine_services)
    for service in "${started_services[@]}"; do
        case "$service" in
            "order-service")
                health_endpoints+=("http://localhost:$ORDER_SERVICE_PORT/health:Order Service")
                ;;
            "payment-service")
                health_endpoints+=("http://localhost:$PAYMENT_SERVICE_PORT/health:Payment Service")
                ;;
            "inventory-service")
                health_endpoints+=("http://localhost:$INVENTORY_SERVICE_PORT/health:Inventory Service")
                ;;
            "dev-dashboard")
                health_endpoints+=("http://localhost:$DEV_DASHBOARD_PORT:Dev Dashboard")
                ;;
        esac
    done
    
    while [ $all_ready = false ]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log_error "服务启动超时 (${timeout}秒)"
            return 1
        fi
        
        local ready_count=0
        local total_endpoints=${#health_endpoints[@]}
        
        for endpoint_info in "${health_endpoints[@]}"; do
            local endpoint="${endpoint_info%:*}"
            local name="${endpoint_info#*:}"
            
            if curl -f -s --connect-timeout 5 "$endpoint" > /dev/null 2>&1; then
                ((ready_count++))
            fi
        done
        
        if [ $ready_count -eq $total_endpoints ]; then
            all_ready=true
            log_success "所有服务已就绪 (用时: ${elapsed}秒)"
        else
            printf "\r等待服务就绪... (%d/%d) [%ds]" "$ready_count" "$total_endpoints" "$elapsed"
            sleep 2
        fi
    done
    
    echo  # 换行
}

# 生成种子数据
generate_seed_data() {
    if [ "$SEED_DATA" != "true" ]; then
        return 0
    fi
    
    log_header "生成种子数据"
    
    log_info "生成测试追踪数据..."
    "$PROJECT_ROOT/tools/trace-generator" -n 20 -c 3 -s mixed --quiet
    
    log_success "种子数据生成完成"
}

# 运行自动测试
run_auto_tests() {
    if [ "$AUTO_TEST" != "true" ]; then
        return 0
    fi
    
    log_header "运行自动测试"
    
    # 运行健康检查
    log_info "执行健康检查..."
    if "$PROJECT_ROOT/scripts/health-check.sh" --quiet; then
        log_success "健康检查通过"
    else
        log_warning "健康检查未完全通过"
    fi
    
    # 运行基础功能测试
    log_info "执行基础功能测试..."
    
    # 测试订单创建
    local order_response
    order_response=$(curl -s -w "%{http_code}" -X POST http://localhost:$ORDER_SERVICE_PORT/api/v1/orders \
        -H "Content-Type: application/json" \
        -d '{"customer_id": "test-dev", "product_id": "product-001", "quantity": 1, "amount": 99.99}' \
        -o /dev/null)
    
    if [[ "$order_response" =~ ^2[0-9][0-9]$ ]]; then
        log_success "订单服务测试通过"
    else
        log_warning "订单服务测试失败 (HTTP $order_response)"
    fi
    
    log_success "自动测试完成"
}

# 启动监控和日志
start_monitoring() {
    if [ "$LOGS_TAIL" != "true" ] && [ "$DASHBOARD" != "true" ]; then
        return 0
    fi
    
    log_header "启动监控和日志"
    
    # 启动日志跟踪 (后台)
    if [ "$LOGS_TAIL" = "true" ]; then
        log_info "启动日志跟踪..."
        
        # 创建日志跟踪脚本
        cat > "$PROJECT_ROOT/tmp/log-tail.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/.."
while true; do
    echo "=== 服务日志跟踪开始 $(date) ==="
    docker-compose logs --tail=50 -f 2>/dev/null || true
    sleep 5
done
EOF
        chmod +x "$PROJECT_ROOT/tmp/log-tail.sh"
        
        # 在后台启动日志跟踪
        "$PROJECT_ROOT/tmp/log-tail.sh" > "$PROJECT_ROOT/logs/dev-tail.log" 2>&1 &
        local log_tail_pid=$!
        echo $log_tail_pid > "$PROJECT_ROOT/tmp/log-tail.pid"
        
        log_success "日志跟踪已在后台启动 (PID: $log_tail_pid)"
        log_info "日志文件: $PROJECT_ROOT/logs/dev-tail.log"
    fi
    
    # 启动实时日志分析 (可选)
    if [ "$PROFILE" = "true" ]; then
        log_info "启动性能分析..."
        
        # 在后台启动性能监控
        cat > "$PROJECT_ROOT/tmp/perf-monitor.sh" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")/.."
while true; do
    ./tools/log-parser --performance --stats --quiet --output=json --output-file="reports/perf-$(date +%s).json"
    sleep 60
done
EOF
        chmod +x "$PROJECT_ROOT/tmp/perf-monitor.sh"
        
        "$PROJECT_ROOT/tmp/perf-monitor.sh" &
        local perf_pid=$!
        echo $perf_pid > "$PROJECT_ROOT/tmp/perf-monitor.pid"
        
        log_success "性能监控已启动 (PID: $perf_pid)"
    fi
}

# 显示开发环境信息
show_dev_info() {
    log_header "开发环境信息"
    
    echo -e "${GREEN}🎉 分布式追踪开发环境启动成功！${NC}"
    echo
    echo "🌐 服务访问地址："
    echo -e "  📊 ${BLUE}Jaeger UI:${NC}         http://localhost:$JAEGER_UI_PORT"
    
    mapfile -t started_services < <(determine_services)
    for service in "${started_services[@]}"; do
        case "$service" in
            "order-service")
                echo -e "  🛍️  ${BLUE}订单服务:${NC}          http://localhost:$ORDER_SERVICE_PORT"
                echo -e "  🔍 ${BLUE}订单服务健康检查:${NC}    http://localhost:$ORDER_SERVICE_PORT/health"
                ;;
            "payment-service")
                echo -e "  💳 ${BLUE}支付服务:${NC}          http://localhost:$PAYMENT_SERVICE_PORT"
                echo -e "  🔍 ${BLUE}支付服务健康检查:${NC}    http://localhost:$PAYMENT_SERVICE_PORT/health"
                ;;
            "inventory-service")
                echo -e "  📦 ${BLUE}库存服务:${NC}          http://localhost:$INVENTORY_SERVICE_PORT"
                echo -e "  🔍 ${BLUE}库存服务健康检查:${NC}    http://localhost:$INVENTORY_SERVICE_PORT/health"
                ;;
            "dev-dashboard")
                echo -e "  🎛️  ${BLUE}开发仪表板:${NC}        http://localhost:$DEV_DASHBOARD_PORT"
                ;;
            "prometheus")
                echo -e "  📈 ${BLUE}Prometheus:${NC}       http://localhost:9090"
                ;;
            "grafana")
                echo -e "  📊 ${BLUE}Grafana:${NC}          http://localhost:3001 (admin/admin)"
                ;;
        esac
    done
    
    echo
    echo "🛠️  开发工具："
    echo -e "  🎯 ${BLUE}生成测试数据:${NC}        ./tools/trace-generator -n 10 -s mixed"
    echo -e "  🔍 ${BLUE}分析日志:${NC}            ./tools/log-parser --analyze --stats"
    echo -e "  📊 ${BLUE}性能测试:${NC}            ./scripts/benchmark.sh -t throughput"
    echo -e "  🔧 ${BLUE}配置验证:${NC}            ./tools/config-validator"
    
    echo
    echo "📋 开发命令："
    echo -e "  ${BLUE}查看服务状态:${NC}         docker-compose ps"
    echo -e "  ${BLUE}查看服务日志:${NC}         docker-compose logs -f [service-name]"
    echo -e "  ${BLUE}重启服务:${NC}             docker-compose restart [service-name]"
    echo -e "  ${BLUE}停止环境:${NC}             ./scripts/stop.sh"
    echo -e "  ${BLUE}完全清理:${NC}             ./scripts/cleanup.sh"
    
    if [ "$DEBUG_PORT" != "" ]; then
        echo
        echo "🐛 调试信息："
        echo -e "  ${BLUE}调试端口:${NC}             localhost:$DEBUG_PORT"
        echo -e "  ${BLUE}调试模式:${NC}             已启用"
    fi
    
    echo
    echo "📁 重要目录："
    echo -e "  ${BLUE}日志目录:${NC}             $PROJECT_ROOT/logs/"
    echo -e "  ${BLUE}报告目录:${NC}             $PROJECT_ROOT/reports/"
    echo -e "  ${BLUE}配置文件:${NC}             $PROJECT_ROOT/scripts/config/"
    
    if [ "$LOGS_TAIL" = "true" ]; then
        echo
        echo "📡 实时监控："
        echo -e "  ${BLUE}日志跟踪:${NC}             tail -f $PROJECT_ROOT/logs/dev-tail.log"
    fi
}

# 设置开发环境清理
setup_cleanup_handlers() {
    # 创建清理脚本
    cat > "$PROJECT_ROOT/tmp/dev-cleanup.sh" << 'EOF'
#!/bin/bash
echo "清理开发环境..."

# 停止后台进程
if [ -f tmp/log-tail.pid ]; then
    kill $(cat tmp/log-tail.pid) 2>/dev/null || true
    rm -f tmp/log-tail.pid
fi

if [ -f tmp/perf-monitor.pid ]; then
    kill $(cat tmp/perf-monitor.pid) 2>/dev/null || true
    rm -f tmp/perf-monitor.pid
fi

# 停止 Docker 服务 (可选)
# docker-compose down

echo "开发环境清理完成"
EOF
    chmod +x "$PROJECT_ROOT/tmp/dev-cleanup.sh"
    
    # 注册退出处理器
    trap 'echo; echo "正在清理开发环境..."; $PROJECT_ROOT/tmp/dev-cleanup.sh; exit 0' SIGINT SIGTERM
}

# 主函数
main() {
    # 默认参数
    MODE="full"
    HOT_RELOAD="false"
    DEBUG_PORT=""
    PROFILE="false"
    MOCK_EXTERNAL="false"
    SEED_DATA="false"
    AUTO_TEST="false"
    DASHBOARD="false"
    LOGS_TAIL="false"
    WAIT_TIME=60
    CLEANUP="false"
    VERBOSE="false"
    QUIET="false"
    SELECTED_SERVICES=()
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -m|--mode)
                MODE="$2"
                shift 2
                ;;
            --mode=*)
                MODE="${1#*=}"
                shift
                ;;
            --hot-reload)
                HOT_RELOAD="true"
                shift
                ;;
            --debug-port=*)
                DEBUG_PORT="${1#*=}"
                shift
                ;;
            --profile)
                PROFILE="true"
                shift
                ;;
            --mock-external)
                MOCK_EXTERNAL="true"
                shift
                ;;
            --seed-data)
                SEED_DATA="true"
                shift
                ;;
            --auto-test)
                AUTO_TEST="true"
                shift
                ;;
            --dashboard)
                DASHBOARD="true"
                shift
                ;;
            --logs-tail)
                LOGS_TAIL="true"
                shift
                ;;
            --wait-time=*)
                WAIT_TIME="${1#*=}"
                shift
                ;;
            --cleanup)
                CLEANUP="true"
                shift
                ;;
            --verbose)
                VERBOSE="true"
                shift
                ;;
            --quiet)
                QUIET="true"
                shift
                ;;
            jaeger|order-service|payment-service|inventory-service|monitoring)
                SELECTED_SERVICES+=("$1")
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 验证参数
    if [[ ! "$MODE" =~ ^(minimal|standard|full|debug)$ ]]; then
        log_error "无效的开发模式: $MODE"
        log_info "支持的模式: minimal, standard, full, debug"
        exit 1
    fi
    
    if [ "$QUIET" != "true" ]; then
        show_welcome
    fi
    
    # 设置清理处理器
    setup_cleanup_handlers
    
    # 执行开发环境搭建流程
    load_config
    check_dev_dependencies
    cleanup_environment
    prepare_dev_environment
    start_services
    wait_for_services
    generate_seed_data
    run_auto_tests
    start_monitoring
    
    if [ "$QUIET" != "true" ]; then
        show_dev_info
        
        if [ "$LOGS_TAIL" = "true" ]; then
            echo
            log_info "按 Ctrl+C 停止实时日志并清理环境"
            echo
            log_dev "实时日志跟踪中..."
            tail -f "$PROJECT_ROOT/logs/dev-tail.log" 2>/dev/null || sleep infinity
        else
            echo
            log_success "开发环境已启动！使用上述地址访问服务"
            log_info "使用 ./scripts/stop.sh 停止服务"
        fi
    fi
}

# 执行主函数
main "$@"