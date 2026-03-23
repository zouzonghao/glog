# Glog - 一个简洁、高效的 Go 博客系统

Glog 是一个使用 Go 语言编写的轻量级博客系统。它设计简洁，易于部署和使用，专注于核心博客功能，同时提供 REST API 供外部程序集成。

## 特性

-   **轻量级**: 基于 Gin 框架，性能卓越，资源占用少。
-   **易于部署**: 支持 Docker 和二进制文件直接部署。
-   **Markdown 编辑器**: 内置 Markdown 编辑器，支持实时预览。
-   **数据备份**: 支持本地备份、GitHub 和 WebDAV 自动备份。
-   **全文搜索**: 内置简单的全文搜索功能。
-   **REST API**: 提供 API 用于外部程序集成（如 AI 摘要、封面生成等）。

## 架构

Glog 采用经典的分层架构，清晰地分离了不同模块的职责。

-   **Web 框架**: 使用 [Gin](https://github.com/gin-gonic/gin) 作为核心 Web 框架，负责路由和 HTTP 请求处理。
-   **数据库**: 使用 [SQLite](https://www.sqlite.org/) 作为默认数据库，通过 [GORM](https://gorm.io/) 进行对象关系映射（ORM）。SQLite 的使用使得 Glog 无需外部数据库依赖，简化了部署。
-   **模板引擎**: 使用 Go 内置的 `html/template`，并通过 `gin-contrib/multitemplate` 集成到 Gin 中，用于渲染前端页面。
-   **Markdown 渲染**: 使用 [Goldmark](https://github.com/yuin/goldmark) 将 Markdown 内容转换为 HTML。

### 项目结构

```
.
├── internal/
│   ├── handlers/    # HTTP 请求处理器
│   ├── services/    # 业务逻辑层
│   ├── repository/  # 数据访问层
│   ├── models/      # 数据模型定义
│   ├── tasks/       # 定时任务（如自动备份）
│   └── utils/       # 工具函数
├── static/          # 静态资源 (CSS, JS, images)
├── templates/       # HTML 模板
├── main.go          # 程序入口
├── Dockerfile       # Docker 配置文件
├── Makefile         # 项目构建脚本
└── install.sh       # Debian/Ubuntu 安装脚本
```

## 部署

Glog 提供多种部署方式。

### 1. 使用 Docker

```yml
version: '3.8'

services:
  glog:
    image: sanqi37/glog:latest
    container_name: glog
    restart: unless-stopped
    environment:
      - DB_PATH=/app/db/glog.db
    ports:
      - "37371:37371"
    volumes:
      - "$(pwd)/glog_db:/app/db" 
    # 如果启用了反向代理通过 https 访问，则注释下面这一行。否则无法登录
    command: ["/app/glog", "--unsafe"]
```


### 2. 使用脚本安装（推荐）

#### 安装命令
```bash
bash <(curl -sL https://zfff.de/glogsh) install
```

#### 卸载命令
```bash
bash <(curl -sL https://zfff.de/glogsh) uninstall
```

#### 默认路径

*   **安装目录**: `/opt/glog`
*   **数据文件**: `/opt/glog/glog.db` 

#### 服务管理 (Systemd)

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


### 3. 二进制文件手动部署

你也可以直接在服务器上运行预编译的二进制文件。

1.  前往 [Releases](https://github.com/your_username/glog/releases) 页面下载对应你服务器操作系统和架构的最新版本。
2.  解压下载的文件。
3.  直接运行二进制文件：

    ```bash
    ./glog-linux-amd64
    ```

    程序将在前台运行。建议使用 `systemd` 或 `supervisor` 等工具来管理进程。

## 开发

### 环境要求

-   Go 1.23+
-   Make

### 本地运行

1.  克隆仓库：

    ```bash
    git clone https://github.com/your_username/glog.git
    cd glog
    ```

2.  安装依赖：

    ```bash
    go mod tidy
    ```

3.  运行项目：

    ```bash
    make run
    ```

    或者直接运行：

    ```bash
    go run .
    ```

    > **注意**: 请使用 `go run .` 而不是 `go run main.go`，因为项目使用了编译标签来区分开发和生产模式的资源加载。

    服务将启动在 `http://localhost:37371`。

### 构建

你可以使用 `Makefile` 来构建不同平台的二进制文件。

-   **构建所有发布版本 (Linux, Windows, macOS)**:

    ```bash
    make release-all
    ```

-   **构建特定平台版本**:

    ```bash
    # 构建 Linux 版本
    GOOS=linux GOARCH=amd64 make build-platform-with-cleanup

    # 构建 Windows 版本
    GOOS=windows GOARCH=amd64 make build-platform-with-cleanup

    # 构建 macOS (Apple Silicon) 版本
    GOOS=darwin GOARCH=arm64 make build-platform-with-cleanup
    ```

    构建产物将出现在项目根目录。

## API 文档

Glog 提供 REST API 用于外部程序集成。你可以使用这些 API 实现自动生成文章摘要、封面图片等功能。

### 认证

所有 API 请求都需要通过 `Authorization` 请求头进行认证。认证方式为 `Bearer Token`，其中 `Token` 是你在后台设置的站点密码。

**示例:**

```
Authorization: Bearer your_site_password
```

如果认证失败，API 将返回 `401 Unauthorized` 错误。

### API 端点

#### 1. 获取文章列表

获取所有文章列表，包含摘要和封面状态信息。

-   **URL**: `/api/v1/posts`
-   **Method**: `GET`
-   **查询参数**:
    -   `page` (可选): 页码，默认为 `1`。
    -   `page_size` (可选): 每页数量，默认为 `10`。
-   **响应示例**:

    ```json
    {
      "posts": [
        {
          "id": 1,
          "title": "文章标题",
          "slug": "article-slug",
          "excerpt": "文章摘要...",
          "cover": "https://example.com/cover.jpg",
          "has_cover": true,
          "is_private": false,
          "published_at": "2025-01-15T10:00:00+08:00"
        }
      ],
      "total": 100,
      "page": 1,
      "page_size": 10
    }
    ```

#### 2. 获取文章详情

获取单篇文章的完整内容。

-   **URL**: `/api/v1/posts/:id`
-   **Method**: `GET`
-   **响应示例**:

    ```json
    {
      "id": 1,
      "title": "文章标题",
      "slug": "article-slug",
      "content": "完整的文章内容...",
      "excerpt": "文章摘要...",
      "cover": "https://example.com/cover.jpg",
      "is_private": false,
      "published_at": "2025-01-15T10:00:00+08:00"
    }
    ```

#### 3. 更新文章摘要

更新指定文章的摘要内容。

-   **URL**: `/api/v1/posts/:id/excerpt`
-   **Method**: `PUT`
-   **请求体**:

    ```json
    {
      "excerpt": "新的文章摘要"
    }
    ```

-   **响应示例**:

    ```json
    {
      "status": "success",
      "message": "摘要已更新"
    }
    ```

#### 4. 更新文章封面

更新指定文章的封面图片 URL。

-   **URL**: `/api/v1/posts/:id/cover`
-   **Method**: `PUT`
-   **请求体**:

    ```json
    {
      "cover": "https://example.com/new-cover.jpg"
    }
    ```

-   **响应示例**:

    ```json
    {
      "status": "success",
      "message": "封面已更新"
    }
    ```

### 使用示例

以下是一个使用 API 自动生成文章摘要的 Python 示例：

```python
import requests

API_BASE = "http://localhost:37371/api/v1"
API_KEY = "your_site_password"

headers = {"Authorization": f"Bearer {API_KEY}"}

# 获取文章列表
response = requests.get(f"{API_BASE}/posts", headers=headers)
posts = response.json()["posts"]

for post in posts:
    if not post["excerpt"]:
        # 获取文章内容
        detail = requests.get(f"{API_BASE}/posts/{post['id']}", headers=headers).json()
        content = detail["content"]
        
        # 调用 AI 生成摘要（示例）
        summary = generate_summary_with_ai(content)
        
        # 更新摘要
        requests.put(
            f"{API_BASE}/posts/{post['id']}/excerpt",
            headers=headers,
            json={"excerpt": summary}
        )
```

## 许可证

MIT License
