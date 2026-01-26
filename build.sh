#!/bin/bash

# SNMP Trap 构建脚本
# 用于编译不同架构的可执行文件

set -e

# 配置参数
EXECUTABLE="snmptrap"
BUILD_DIR="./build"

# 颜色定义
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Go 环境
check_go_env() {
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.16+"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "检测到 Go 版本: $GO_VERSION"
}

# 编译 x86 架构 (默认)
build_x86() {
    log_info "开始编译 x86 架构..."
    
    # 创建构建目录
    if [ ! -d "$BUILD_DIR" ]; then
        mkdir -p "$BUILD_DIR"
    fi
    
    # 设置环境变量
    export GOOS=linux
    export GOARCH=amd64
    export CGO_ENABLED=0
    
    # 编译（添加优化选项减小体积）
    log_info "执行编译命令..."
    go build -ldflags="-s -w" -trimpath -o "$BUILD_DIR/${EXECUTABLE}_x86" .
    
    # 复制到当前目录
    cp "$BUILD_DIR/${EXECUTABLE}_x86" "./$EXECUTABLE"
    chmod +x "./$EXECUTABLE"
    
    log_info "x86 架构编译完成: $EXECUTABLE"
}

# 编译 ARM 架构
build_arm() {
    log_info "开始编译 ARM 架构..."
    
    # 创建构建目录
    if [ ! -d "$BUILD_DIR" ]; then
        mkdir -p "$BUILD_DIR"
    fi
    
    # 设置环境变量
    export GOOS=linux
    export GOARCH=arm64
    export CGO_ENABLED=0
    
    # 编译（添加优化选项减小体积）
    log_info "执行编译命令..."
    go build -ldflags="-s -w" -trimpath -o "$BUILD_DIR/${EXECUTABLE}_arm" .
    
    log_info "ARM 架构编译完成: $BUILD_DIR/${EXECUTABLE}_arm"
}

# 编译所有架构
build_all() {
    log_info "开始编译所有架构..."
    
    # 编译 x86
    build_x86
    
    # 编译 ARM
    build_arm
    
    log_info "所有架构编译完成!"
    log_info "构建产物:"
    ls -la "$BUILD_DIR/"
}

# 清理构建产物
clean() {
    log_info "清理构建产物..."
    
    if [ -d "$BUILD_DIR" ]; then
        rm -rf "$BUILD_DIR"
        log_info "删除构建目录: $BUILD_DIR"
    fi
    
    if [ -f "./$EXECUTABLE" ]; then
        rm -f "./$EXECUTABLE"
        log_info "删除可执行文件: $EXECUTABLE"
    fi
    
    log_info "清理完成!"
}

# 显示帮助信息
show_help() {
    echo "SNMP Trap 构建脚本"
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  x86        编译 x86 架构 (默认)"
    echo "  arm        编译 ARM 架构"
    echo "  all        编译所有架构"
    echo "  clean      清理构建产物"
    echo "  help       显示此帮助信息"
    echo ""
}

# 主函数
main() {
    check_go_env
    
    case "$1" in
        "x86")
            build_x86
            ;;
        "arm")
            build_arm
            ;;
        "all")
            build_all
            ;;
        "clean")
            clean
            ;;
        "help" | "--help" | "-h")
            show_help
            ;;
        "")
            # 默认编译 x86
            build_x86
            ;;
        *)
            log_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"