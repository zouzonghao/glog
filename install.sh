#!/bin/sh

# ==============================================================================
# glog 服务安装脚本 for POSIX sh (兼容 Debian/Ubuntu)
# ==============================================================================

# -- 脚本配置 --
SERVICE_NAME="glog"
INSTALL_DIR="/opt/glog"
EXECUTABLE_NAME="glog"
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

# -- 检查操作系统 --
check_os() {
    if [ -f /etc/os-release ]; then
        # shellcheck source=/dev/null
        . /etc/os-release
        if [ "$ID" = "ubuntu" ] || [ "$ID" = "debian" ] || echo "$ID_LIKE" | grep -q "debian"; then
            info "检测到兼容的操作系统: $PRETTY_NAME"
            return
        fi
    fi
    die "此脚本仅支持 Debian 或 Ubuntu 系统。"
}

# 安装或更新函数
install_service() {
    info "开始安装或更新 ${SERVICE_NAME} 服务..."

    # -- 动态获取最新版本信息 --
    info "正在获取最新版本信息..."
    LATEST_VERSION=$(curl -s https://api.github.com/repos/zouzonghao/glog/releases | grep '"tag_name":' | grep '"v[0-9]' | head -n 1 | cut -d '"' -f 4)
    if [ -z "$LATEST_VERSION" ]; then
        die "无法获取最新的正式版本号 (v*.*.*)。"
    fi
    info "找到最新的正式版本: ${LATEST_VERSION}"
    API_RESPONSE=$(curl -s "https://api.github.com/repos/zouzonghao/glog/releases/tags/${LATEST_VERSION}")
    DOWNLOAD_URL=$(echo "$API_RESPONSE" | grep "browser_download_url" | grep "glog-linux-amd64.tar.gz" | cut -d '"' -f 4)

    if [ -z "$DOWNLOAD_URL" ] || [ -z "$LATEST_VERSION" ]; then
        die "无法获取最新版本信息，请检查网络或 API 限制。"
    fi
    info "最新版本为: ${LATEST_VERSION}"

    # -- 检查是否已安装 --
    if [ -f "$SERVICE_FILE_PATH" ]; then
        warn "${SERVICE_NAME} 服务似乎已经安装。"
        printf "是否要覆盖安装? (y/n): "
        read -r REPLY
        case "$REPLY" in
            [Yy]*) 
                info "正在停止服务以进行覆盖安装..."
                systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true
                ;;
            *) 
                info "安装已取消。"
                exit 0 
                ;;
        esac
    fi

    # -- 数据库处理 --
    DATABASE_FILE="${INSTALL_DIR}/glog.db"
    if [ -f "$DATABASE_FILE" ]; then
        printf "是否要重置数据库? (警告: 这将删除所有现有数据) (y/n): "
        read -r REPLY
        case "$REPLY" in
            [Yy]*) 
                info "正在重置数据库..."
                rm -f "$DATABASE_FILE" || die "删除数据库失败。"
                ;;
            *) info "保留现有数据库。" ;;
        esac
    fi

    command -v curl >/dev/null 2>&1 || die "需要 curl 命令来下载文件，请先安装 (sudo apt install curl)。"
    command -v tar >/dev/null 2>&1 || die "需要 tar 命令来解压文件，请先安装 (sudo apt install tar)。"

    info "创建安装目录: ${INSTALL_DIR}"
    mkdir -p "$INSTALL_DIR" || die "无法创建安装目录 ${INSTALL_DIR}。"

    TEMP_FILE="/tmp/glog-download.tar.gz"
    info "从 ${DOWNLOAD_URL} 下载文件..."
    curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL" || die "下载文件失败。"

    info "清空旧文件并解压新文件到 ${INSTALL_DIR}..."
    find "$INSTALL_DIR" -mindepth 1 ! -name 'glog.db' -exec rm -rf {} +
    tar -xzf "$TEMP_FILE" -C "$INSTALL_DIR" || die "解压文件失败。"
    info "设置文件所有权为 root..."
    chown -R root:root "$INSTALL_DIR" || die "设置文件所有权失败。"

    ORIGINAL_EXECUTABLE="${INSTALL_DIR}/glog-linux-amd64"
    TARGET_EXECUTABLE="${INSTALL_DIR}/${EXECUTABLE_NAME}"
    
    if [ ! -f "$ORIGINAL_EXECUTABLE" ]; then
        die "解压后未找到预期的文件: ${ORIGINAL_EXECUTABLE}"
    fi

    info "设置可执行文件..."
    mv "$ORIGINAL_EXECUTABLE" "$TARGET_EXECUTABLE" || die "重命名文件失败。"
    chmod +x "$TARGET_EXECUTABLE" || die "设置执行权限失败。"

    info "设置目录权限为 777..."
    chmod -R 777 "$INSTALL_DIR" || die "设置目录权限失败。"

    info "创建或更新 systemd 服务文件..."
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
    systemctl daemon-reload
    
    info "启用并启动 ${SERVICE_NAME} 服务..."
    systemctl enable "${SERVICE_NAME}"
    systemctl start "${SERVICE_NAME}"
    
    # -- 记录版本号 --
    echo "$LATEST_VERSION" > "${INSTALL_DIR}/.version" || warn "无法写入版本文件。"
    
    rm -f "$TEMP_FILE"

    info "----------------------------------------------------"
    printf "${GREEN}操作成功! (${LATEST_VERSION})${NC}\n"
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

    # -- 数据库处理 --
    DATABASE_FILE="${INSTALL_DIR}/glog.db"
    DELETE_DB=1
    if [ -f "$DATABASE_FILE" ]; then
        printf "是否要删除数据库文件? (警告: 数据将无法恢复) (y/n): "
        read -r REPLY
        case "$REPLY" in
            [Yy]*) info "数据库将被删除。" ;;
            *) info "将保留数据库文件: ${DATABASE_FILE}"; DELETE_DB=0 ;;
        esac
    fi

    info "停止并禁用服务..."
    systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true
    systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
    
    info "删除 systemd 服务文件..."
    rm -f "$SERVICE_FILE_PATH"
    systemctl daemon-reload

    info "删除安装目录..."
    if [ "$DELETE_DB" -eq 1 ]; then
        rm -rf "$INSTALL_DIR"
    else
        # 只删除程序文件，保留数据库
        rm -f "${INSTALL_DIR}/${EXECUTABLE_NAME}"
        rm -f "${INSTALL_DIR}/.version"
        # 如果目录为空，也删除它
    check_os
        if [ -z "$(ls -A "$INSTALL_DIR")" ]; then
            rm -rf "$INSTALL_DIR"
        fi
    fi
    
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
