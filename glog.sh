#!/bin/sh

# ==============================================================================
# glog 服务安装脚本 for POSIX sh (兼容 Debian/Ubuntu)
# ==============================================================================

# -- 脚本配置 --
SERVICE_NAME="glog"
INSTALL_DIR="/opt/glog"
EXECUTABLE_NAME="glog"
DOWNLOAD_URL="https://github.com/zouzonghao/glog/releases/download/v1.0.0/glog-linux-amd64.tar.gz"
SERVICE_FILE_PATH="/etc/systemd/system/${SERVICE_NAME}.service"

# -- 颜色定义 (自动检测终端) --
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    NC='\033[0m'
else
    GREEN=''
    RED=''
    YELLOW=''
    NC=''
fi

# -- 函数定义 (使用 printf 替代 echo -e) --

die() {
    printf "${RED}错误: %s${NC}\n" "$1" >&2
    exit 1
}

info() {
    printf "${GREEN}信息: %s${NC}\n" "$1"
}

warn() {
    printf "${YELLOW}警告: %s${NC}\n" "$1"
}

# 安装函数
install_service() {
    info "开始安装 ${SERVICE_NAME} 服务..."

    # 1. 检查是否已安装
    if [ -f "$SERVICE_FILE_PATH" ]; then
        warn "${SERVICE_NAME} 服务似乎已经安装。"
        
        # 使用 POSIX 兼容的方式获取用户输入
        printf "是否要覆盖安装? (y/n): "
        read -r REPLY
        
        case "$REPLY" in
            [Yy]* )
                # 用户同意，继续执行
                uninstall_service
                info "旧版本已卸载，现在开始重新安装..."
                ;;
            * )
                # 其他所有情况都视为取消
                info "安装已取消。"
                exit 0
                ;;
        esac
    fi

    command -v curl >/dev/null 2>&1 || die "需要 curl 命令来下载文件，请先安装 (sudo apt install curl)。"
    command -v tar >/dev/null 2>&1 || die "需要 tar 命令来解压文件，请先安装 (sudo apt install tar)。"

    info "创建安装目录: ${INSTALL_DIR}"
    mkdir -p "$INSTALL_DIR" || die "无法创建安装目录 ${INSTALL_DIR}。"

    TEMP_FILE="/tmp/glog-download.tar.gz"
    info "从 ${DOWNLOAD_URL} 下载文件..."
    curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL" || die "下载文件失败。"

    info "解压文件到 ${INSTALL_DIR}..."
    tar -xzf "$TEMP_FILE" -C "$INSTALL_DIR" || die "解压文件失败。"

    ORIGINAL_EXECUTABLE="${INSTALL_DIR}/glog-linux-amd64"
    TARGET_EXECUTABLE="${INSTALL_DIR}/${EXECUTABLE_NAME}"
    
    if [ ! -f "$ORIGINAL_EXECUTABLE" ]; then
        die "解压后未找到预期的文件: ${ORIGINAL_EXECUTABLE}"
    fi

    info "设置可执行文件..."
    mv "$ORIGINAL_EXECUTABLE" "$TARGET_EXECUTABLE" || die "重命名文件失败。"
    chmod +x "$TARGET_EXECUTABLE" || die "设置执行权限失败。"

    info "创建 systemd 服务文件: ${SERVICE_FILE_PATH}"
    cat << EOF > "$SERVICE_FILE_PATH"
[Unit]
Description=GLog Service
After=network.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${TARGET_EXECUTABLE}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    info "重新加载 systemd 配置..."
    systemctl daemon-reload

    info "启用并启动 ${SERVICE_NAME} 服务..."
    systemctl enable "${SERVICE_NAME}"
    systemctl start "${SERVICE_NAME}"
    
    rm -f "$TEMP_FILE"

    info "----------------------------------------------------"
    printf "${GREEN}安装成功!${NC}\n"
    printf "服务状态检查: ${YELLOW}sudo systemctl status ${SERVICE_NAME}${NC}\n"
    printf "查看服务日志: ${YELLOW}sudo journalctl -u ${SERVICE_NAME} -f${NC}\n"
    printf "程序监听端口: ${YELLOW}37371${NC}\n"
    info "----------------------------------------------------"
}

# 卸载函数
uninstall_service() {
    info "开始卸载 ${SERVICE_NAME} 服务..."
    if [ ! -f "$SERVICE_FILE_PATH" ]; then
        warn "${SERVICE_NAME} 服务未安装。"
        return
    fi
    info "停止并禁用服务..."
    systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true
    systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
    info "删除 systemd 服务文件..."
    rm -f "$SERVICE_FILE_PATH"
    systemctl daemon-reload
    info "删除安装目录: ${INSTALL_DIR}"
    rm -rf "$INSTALL_DIR"
    info "${GREEN}卸载完成！${NC}"
}

# 主逻辑
main() {
    if [ "$(id -u)" -ne 0 ]; then
        die "此脚本需要以 root 权限运行。请使用 sudo。"
    fi
    
    case "$1" in
        install)
            install_service
            ;;
        uninstall)
            uninstall_service
            ;;
        *)
            printf "用法: %s {install|uninstall}\n" "$0"
            exit 1
            ;;
    esac
}

main "$@"
