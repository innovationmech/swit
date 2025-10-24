#!/usr/bin/env bash

# Saga 示例应用启动脚本
# 用于运行各种 Saga 示例，支持不同的执行模式和配置选项
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
EXAMPLES_DIR="$PROJECT_ROOT/pkg/saga/examples"

# 配置变量
EXAMPLE_NAME=""
TEST_MODE=false
VERBOSE=false
COVERAGE=false
RACE_DETECTOR=false
TIMEOUT="30s"
SPECIFIC_TEST=""
DRY_RUN=false

# 可用的示例列表
declare -A EXAMPLES=(
    ["order"]="订单处理 Saga"
    ["payment"]="支付处理 Saga"
    ["inventory"]="库存管理 Saga"
    ["user"]="用户注册 Saga"
    ["e2e"]="端到端测试"
    ["all"]="所有示例"
)

# 函数：打印日志
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

log_step() {
    echo -e "${CYAN}==>${NC} $1"
}

# 函数：显示帮助信息
show_help() {
    cat << EOF
Saga 示例应用启动脚本

用法:
    $0 [选项] <示例名称>

示例名称:
    order       运行订单处理 Saga 示例
    payment     运行支付处理 Saga 示例
    inventory   运行库存管理 Saga 示例
    user        运行用户注册 Saga 示例
    e2e         运行端到端测试
    all         运行所有示例测试

选项:
    -h, --help          显示此帮助信息
    -t, --test          测试模式（运行单元测试）
    -v, --verbose       详细输出模式
    -c, --coverage      生成覆盖率报告
    -r, --race          启用竞态检测
    --timeout DURATION  设置测试超时时间（默认: 30s）
    --specific TEST     运行特定的测试函数
    -n, --dry-run       试运行模式（显示命令但不执行）

环境变量:
    SAGA_LOG_LEVEL      日志级别（debug, info, warn, error）
    SAGA_STORAGE_TYPE   存储类型（memory, redis, postgres）
    SAGA_TRACE_ENABLED  启用分布式追踪（true/false）

示例:
    $0 order                                # 运行订单处理示例测试
    $0 --test order                         # 运行订单处理单元测试
    $0 --coverage all                       # 运行所有测试并生成覆盖率
    $0 --verbose --race payment             # 详细模式运行支付处理测试（含竞态检测）
    $0 --specific TestOrderSagaSuccess order # 运行特定测试函数
    $0 e2e                                  # 运行端到端测试

EOF
}

# 函数：列出所有可用示例
list_examples() {
    log_info "可用的 Saga 示例："
    echo ""
    for key in "${!EXAMPLES[@]}"; do
        printf "  ${GREEN}%-12s${NC} %s\n" "$key" "${EXAMPLES[$key]}"
    done
    echo ""
}

# 函数：验证示例名称
validate_example() {
    local name=$1
    if [[ ! ${EXAMPLES[$name]+_} ]]; then
        log_error "未知的示例名称: $name"
        echo ""
        list_examples
        exit 1
    fi
}

# 函数：检查依赖
check_dependencies() {
    log_step "检查依赖..."
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "未找到 Go 命令，请先安装 Go 1.23+"
        exit 1
    fi
    
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go 版本: $go_version"
    
    # 检查项目目录
    if [[ ! -d "$EXAMPLES_DIR" ]]; then
        log_error "示例目录不存在: $EXAMPLES_DIR"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 函数：构建测试命令
build_test_command() {
    local example=$1
    local cmd="go test"
    
    # 添加基本标志
    if [[ "$VERBOSE" == true ]]; then
        cmd="$cmd -v"
    fi
    
    if [[ "$COVERAGE" == true ]]; then
        cmd="$cmd -cover -coverprofile=coverage_${example}.out"
    fi
    
    if [[ "$RACE_DETECTOR" == true ]]; then
        cmd="$cmd -race"
    fi
    
    cmd="$cmd -timeout $TIMEOUT"
    
    # 添加测试过滤
    if [[ -n "$SPECIFIC_TEST" ]]; then
        cmd="$cmd -run $SPECIFIC_TEST"
    else
        case $example in
            "order")
                cmd="$cmd -run 'Test(CreateOrder|ReserveInventory|ProcessPayment|ConfirmOrder|OrderProcessing)'"
                ;;
            "payment")
                cmd="$cmd -run 'Test(ValidateAccounts|FreezeAmount|ExecuteTransfer|ReleaseFreeze|CreateAuditRecord|SendTransferNotification|PaymentProcessing|GenerateValidationCode|GenerateTransferID|GenerateUnfreezeID)'"
                ;;
            "inventory")
                cmd="$cmd -run 'Test(CheckInventory|ReserveMultiWarehouseInventory|AllocateMultiWarehouseInventory|ReleaseReservation|CreateInventoryAuditRecord|SendInventoryNotification|InventoryManagement|GenerateReleaseID|ErrorTypes)'"
                ;;
            "user")
                cmd="$cmd -run 'Test(CreateUserAccount|SendVerificationEmail|InitializeUserConfig|AllocateResources|TrackRegistrationEvent|UserRegistration|ValidationFunctions|InvalidDataTypes)'"
                ;;
            "e2e")
                cmd="$cmd -run 'Test(OrderProcessingSaga_E2E|PaymentProcessingSaga_E2E|InventoryManagementSaga_E2E|UserRegistrationSaga_E2E|MultipleSagas_E2E)'"
                ;;
            "all")
                # 运行所有测试
                ;;
        esac
    fi
    
    echo "$cmd"
}

# 函数：运行示例测试
run_example_test() {
    local example=$1
    local example_name="${EXAMPLES[$example]}"
    
    log_step "运行 $example_name..."
    
    # 切换到示例目录
    cd "$EXAMPLES_DIR"
    
    # 构建测试命令
    local cmd=$(build_test_command "$example")
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "试运行模式 - 将执行的命令："
        echo "  cd $EXAMPLES_DIR"
        echo "  $cmd"
        return 0
    fi
    
    # 设置环境变量
    export SAGA_LOG_LEVEL="${SAGA_LOG_LEVEL:-info}"
    export SAGA_STORAGE_TYPE="${SAGA_STORAGE_TYPE:-memory}"
    export SAGA_TRACE_ENABLED="${SAGA_TRACE_ENABLED:-false}"
    
    log_info "执行命令: $cmd"
    echo ""
    
    # 执行测试
    if eval "$cmd"; then
        log_success "$example_name 运行成功"
        
        # 如果生成了覆盖率报告，显示摘要
        if [[ "$COVERAGE" == true && -f "coverage_${example}.out" ]]; then
            echo ""
            log_info "覆盖率摘要:"
            go tool cover -func="coverage_${example}.out" | tail -n 1
            log_info "详细覆盖率报告: coverage_${example}.out"
            log_info "生成 HTML 报告: go tool cover -html=coverage_${example}.out"
        fi
        
        return 0
    else
        log_error "$example_name 运行失败"
        return 1
    fi
}

# 函数：运行所有示例
run_all_examples() {
    log_step "运行所有 Saga 示例测试..."
    
    local failed=0
    local total=0
    
    for example in "order" "payment" "inventory" "user" "e2e"; do
        total=$((total + 1))
        echo ""
        echo "----------------------------------------"
        
        if run_example_test "$example"; then
            log_success "✓ ${EXAMPLES[$example]} 通过"
        else
            log_error "✗ ${EXAMPLES[$example]} 失败"
            failed=$((failed + 1))
        fi
    done
    
    echo ""
    echo "========================================"
    log_info "测试完成: $((total - failed))/$total 通过"
    
    if [[ $failed -gt 0 ]]; then
        log_error "$failed 个测试失败"
        return 1
    else
        log_success "所有测试通过！"
        return 0
    fi
}

# 函数：显示示例信息
show_example_info() {
    local example=$1
    
    log_info "示例信息: ${EXAMPLES[$example]}"
    echo ""
    
    case $example in
        "order")
            cat << EOF
${CYAN}订单处理 Saga${NC}
  演示电商订单处理的完整流程，包括：
  - 创建订单
  - 预留库存
  - 处理支付
  - 确认订单
  - 发送通知

  适用场景:
  - 电商下单流程
  - 多步骤事务协调
  - 资源预留与释放
  - 订单状态管理

  文档: docs/order_saga.md
EOF
            ;;
        "payment")
            cat << EOF
${CYAN}支付处理 Saga${NC}
  演示跨账户资金转账的完整流程，包括：
  - 账户验证
  - 资金冻结
  - 执行转账
  - 风险检测
  - 生成凭证

  适用场景:
  - 金融转账业务
  - 资金安全管理
  - 风控系统集成
  - 账户余额操作

  文档: docs/payment_saga.md
EOF
            ;;
        "inventory")
            cat << EOF
${CYAN}库存管理 Saga${NC}
  演示多仓库库存协调的完整流程，包括：
  - 查询可用仓库
  - 库存可用性检查
  - 分配库存
  - 锁定库存
  - 生成提货单

  适用场景:
  - 多仓库管理
  - 库存分配策略
  - 供应链协调
  - 库存预留机制

  文档: docs/inventory_saga.md
EOF
            ;;
        "user")
            cat << EOF
${CYAN}用户注册 Saga${NC}
  演示用户注册和初始化的完整流程，包括：
  - 创建用户账户
  - 发送验证邮件
  - 初始化配置
  - 分配资源配额
  - 欢迎邮件

  适用场景:
  - 用户注册流程
  - 账户初始化
  - 资源分配
  - 邮件验证

  文档: docs/user_registration_saga.md
EOF
            ;;
        "e2e")
            cat << EOF
${CYAN}端到端测试${NC}
  运行完整的端到端测试套件，包括：
  - 所有示例的集成测试
  - 多示例交互场景
  - 性能和压力测试
  - 故障恢复测试

  文档: E2E_TESTING.md
EOF
            ;;
    esac
    
    echo ""
}

# 主函数
main() {
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -t|--test)
                TEST_MODE=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -r|--race)
                RACE_DETECTOR=true
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --specific)
                SPECIFIC_TEST="$2"
                shift 2
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -l|--list)
                list_examples
                exit 0
                ;;
            -*)
                log_error "未知选项: $1"
                echo ""
                show_help
                exit 1
                ;;
            *)
                EXAMPLE_NAME="$1"
                shift
                ;;
        esac
    done
    
    # 检查是否提供了示例名称
    if [[ -z "$EXAMPLE_NAME" ]]; then
        log_error "请指定要运行的示例名称"
        echo ""
        list_examples
        exit 1
    fi
    
    # 验证示例名称
    validate_example "$EXAMPLE_NAME"
    
    # 显示横幅
    echo ""
    echo "========================================"
    echo "  Saga 示例应用 - ${EXAMPLES[$EXAMPLE_NAME]}"
    echo "========================================"
    echo ""
    
    # 显示示例信息
    if [[ "$VERBOSE" == true ]]; then
        show_example_info "$EXAMPLE_NAME"
    fi
    
    # 检查依赖
    check_dependencies
    
    echo ""
    
    # 运行示例
    if [[ "$EXAMPLE_NAME" == "all" ]]; then
        run_all_examples
    else
        run_example_test "$EXAMPLE_NAME"
    fi
    
    local exit_code=$?
    
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        log_success "完成！"
    else
        log_error "运行失败，退出码: $exit_code"
    fi
    
    exit $exit_code
}

# 执行主函数
main "$@"

