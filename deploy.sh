#!/bin/bash

# SNMP Trap 部署脚本
# 用于安装、配置和管理 SNMP Trap 服务

set -e

# 配置参数
INSTALL_DIR="/opt/snmptrap"
SERVICE_FILE="snmptrap.service"
CONFIG_FILE="config.json"
EXECUTABLE="snmptrap"

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

# 检查是否以 root 身份运行
check_root() {
    if [ "$(id -u)" != "0" ]; then
        log_error "此脚本需要以 root 身份运行"
        exit 1
    fi
}

# 安装服务
install_service() {
    log_info "开始安装 SNMP Trap 服务..."
    
    # 创建安装目录
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "创建安装目录: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
    
    # 复制可执行文件
    if [ -f "./$EXECUTABLE" ]; then
        log_info "复制可执行文件到 $INSTALL_DIR"
        cp "./$EXECUTABLE" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$EXECUTABLE"
    else
        log_error "可执行文件 $EXECUTABLE 不存在，请先编译项目"
        exit 1
    fi
    
    # 复制配置文件
    if [ -f "./$CONFIG_FILE" ]; then
        log_info "复制配置文件到 $INSTALL_DIR"
        cp "./$CONFIG_FILE" "$INSTALL_DIR/"
    else
        log_warning "配置文件 $CONFIG_FILE 不存在，请手动创建"
    fi
    
    # 复制服务文件
    if [ -f "./$SERVICE_FILE" ]; then
        log_info "复制服务文件到 /etc/systemd/system/"
        cp "./$SERVICE_FILE" "/etc/systemd/system/"
        chmod 644 "/etc/systemd/system/$SERVICE_FILE"
    else
        log_error "服务文件 $SERVICE_FILE 不存在"
        exit 1
    fi
    
    # 重新加载 systemd 配置
    log_info "重新加载 systemd 配置"
    systemctl daemon-reload
    
    # 启用服务
    log_info "启用 SNMP Trap 服务"
    systemctl enable "$SERVICE_FILE"
    
    # 启动服务
    log_info "启动 SNMP Trap 服务"
    systemctl start "$SERVICE_FILE"
    
    # 检查服务状态
    log_info "检查服务状态"
    systemctl status "$SERVICE_FILE" --no-pager
    
    log_info "SNMP Trap 服务安装完成!"
}

# 卸载服务
uninstall_service() {
    log_info "开始卸载 SNMP Trap 服务..."
    
    # 停止服务
    if systemctl is-active --quiet "$SERVICE_FILE"; then
        log_info "停止 SNMP Trap 服务"
        systemctl stop "$SERVICE_FILE"
    fi
    
    # 禁用服务
    if systemctl is-enabled --quiet "$SERVICE_FILE"; then
        log_info "禁用 SNMP Trap 服务"
        systemctl disable "$SERVICE_FILE"
    fi
    
    # 删除服务文件
    if [ -f "/etc/systemd/system/$SERVICE_FILE" ]; then
        log_info "删除服务文件"
        rm -f "/etc/systemd/system/$SERVICE_FILE"
    fi
    
    # 重新加载 systemd 配置
    log_info "重新加载 systemd 配置"
    systemctl daemon-reload
    
    # 删除安装目录
    if [ -d "$INSTALL_DIR" ]; then
        log_info "删除安装目录: $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
    fi
    
    log_info "SNMP Trap 服务卸载完成!"
}

# 显示帮助信息
show_help() {
    echo "SNMP Trap 部署脚本"
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  install    安装 SNMP Trap 服务"
    echo "  uninstall  卸载 SNMP Trap 服务"
    echo "  status     查看服务状态"
    echo "  restart    重启服务"
    echo "  help       显示此帮助信息"
    echo ""
}

# 查看服务状态
check_status() {
    log_info "查看 SNMP Trap 服务状态..."
    systemctl status "$SERVICE_FILE" --no-pager
}

# 重启服务
restart_service() {
    log_info "重启 SNMP Trap 服务..."
    systemctl restart "$SERVICE_FILE"
    systemctl status "$SERVICE_FILE" --no-pager
}

# 主函数
main() {
    check_root
    
    case "$1" in
        "install")
            install_service
            ;;
        "uninstall")
            uninstall_service
            ;;
        "status")
            check_status
            ;;
        "restart")
            restart_service
            ;;
        "help" | "--help" | "-h")
            show_help
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"