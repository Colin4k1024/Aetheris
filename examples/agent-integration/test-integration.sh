#!/bin/bash
#
# Agent Integration 测试脚本
#
# 用法：
#   ./test-integration.sh [command]
#
# 命令：
#   python    - 测试 Python Agent
#   langchain - 测试 LangChain Agent
#   all       - 测试所有
#   help      - 显示帮助

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 测试 Python Agent
test_python() {
    info "Testing Python Agent..."
    
    # 检查服务是否运行
    if ! curl -s http://localhost:9001/health > /dev/null 2>&1; then
        error "Python Agent is not running on port 9001"
        echo "Start it with: cd python-agent && python app.py"
        return 1
    fi
    
    # 测试健康检查
    info "Health check..."
    curl -s http://localhost:9001/health | jq .
    
    # 测试 invoke
    info "Testing invoke..."
    curl -s -X POST http://localhost:9001/invoke \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: test-$(date +%s)" \
      -d '{"message": "Hello from test!"}' | jq .
    
    info "Python Agent test completed!"
}

# 测试 LangChain Agent
test_langchain() {
    info "Testing LangChain Agent..."
    
    # 检查服务是否运行
    if ! curl -s http://localhost:9002/health > /dev/null 2>&1; then
        error "LangChain Agent is not running on port 9002"
        echo "Start it with: cd langchain-agent && python app.py"
        return 1
    fi
    
    # 测试健康检查
    info "Health check..."
    curl -s http://localhost:9002/health | jq .
    
    # 测试 invoke
    info "Testing invoke..."
    curl -s -X POST http://localhost:9002/invoke \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: test-$(date +%s)" \
      -d '{"message": "What time is it?"}' | jq .
    
    info "LangChain Agent test completed!"
}

# 测试 Aetheris 集成
test_aetheris() {
    info "Testing Aetheris integration..."
    
    # 检查 Aetheris 是否运行
    if ! curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
        error "Aetheris is not running on port 8080"
        echo "Start it with: make run-embedded"
        return 1
    fi
    
    # 测试提交任务
    info "Submitting job to Aetheris..."
    curl -s -X POST http://localhost:8080/api/agents/my_python_agent/message \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: aetheris-test-$(date +%s)" \
      -d '{"message": "Hello from Aetheris test!"}' | jq .
    
    info "Aetheris integration test completed!"
}

# 显示帮助
show_help() {
    echo "Agent Integration Test Script"
    echo ""
    echo "Usage: ./test-integration.sh [command]"
    echo ""
    echo "Commands:"
    echo "  python    - Test Python Agent (port 9001)"
    echo "  langchain - Test LangChain Agent (port 9002)"
    echo "  aetheris  - Test Aetheris integration (port 8080)"
    echo "  all       - Test all services"
    echo "  help      - Show this help"
    echo ""
    echo "Examples:"
    echo "  ./test-integration.sh python"
    echo "  ./test-integration.sh all"
}

# 主函数
main() {
    case "${1:-help}" in
        python)
            test_python
            ;;
        langchain)
            test_langchain
            ;;
        aetheris)
            test_aetheris
            ;;
        all)
            test_python
            echo ""
            test_langchain
            echo ""
            test_aetheris
            ;;
        help|*)
            show_help
            ;;
    esac
}

main "$@"
