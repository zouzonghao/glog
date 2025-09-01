# glog

## 安装与卸载

### 安装命令
```bash
bash <(curl -sL https://cdn.jsdelivr.net/gh/zouzonghao/glog@install/glog.sh) install
```

### 卸载命令
```bash
bash <(curl -sL https://cdn.jsdelivr.net/gh/zouzonghao/glog@install/glog.sh) uninstall
```

## 默认路径

*   **安装目录**: `/opt/glog`
*   **数据文件**: `/opt/glog/glog.db` 

## 服务管理 (Systemd)

使用 `systemctl` 管理 glog 服务。

*   **启动服务**:
    ```bash
    sudo systemctl start glog
    ```
*   **停止服务**:
    ```bash
    sudo systemctl stop glog
    ```
*   **重启服务**:
    ```bash
    sudo systemctl restart glog
    ```
*   **查看服务状态**:
    ```bash
    sudo systemctl status glog
    ```
*   **设置开机自启**:
    ```bash
    sudo systemctl enable glog
    ```
*   **取消开机自启**:
    ```bash
    sudo systemctl disable glog
    ```
*   **查看实时日志**:
    ```bash
    sudo journalctl -u glog -f
