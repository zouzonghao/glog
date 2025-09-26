# Glog - 一个前后端分离的现代化 Go 博客系统

Glog 是一个使用 Go 语言和 Vue 重新构建的轻量级博客系统。它设计简洁、易于部署，并内置了 AI 辅助功能。

## 特性

-   **前后端分离**: 前端使用 Vue 3 + Vite，后端使用 Go + Gin，架构清晰，易于维护。
-   **高性能**: 后端基于 Gin 框架，性能卓越；前端通过 Vite 构建，加载迅速。
-   **易于部署**: 后端支持 Docker 和二进制文件部署，前端可托管于任何静态 Web 服务器。
-   **Markdown 编辑器**: 内置强大的 Markdown 编辑器，支持实时预览。
-   **AI 辅助**: 可选集成 OpenAI API，自动生成文章摘要和标题。
-   **数据备份**: 支持本地备份、GitHub 和 WebDAV 自动备份。
-   **全文搜索**: 内置简单的全文搜索功能。
-   **API**: 提供 RESTful API 用于文章的增删改查。

## 架构

Glog 采用前后端分离架构。

-   **后端 (`/backend`)**:
    -   **Web 框架**: 使用 [Gin](https://github.com/gin-gonic/gin) 作为核心 Web 框架，提供 JSON API。
    -   **数据库**: 使用 [SQLite](https://www.sqlite.org/) 作为默认数据库，通过 [GORM](https://gorm.io/) 进行对象关系映射（ORM）。
-   **前端 (`/frontend`)**:
    -   **框架**: 使用 [Vue 3](https://vuejs.org/) 和 [Vite](https://vitejs.dev/) 构建的单页面应用 (SPA)。
    -   **路由**: 使用 `vue-router` 管理页面路由。
    -   **API 请求**: 使用 `axios` 与后端 API 通信。

### 项目结构

```
.
├── backend/         # 后端 Go 项目
│   ├── internal/
│   ├── main.go
│   ├── go.mod
│   ├── Makefile
│   └── Dockerfile
├── frontend/        # 前端 Vue 项目
│   ├── src/
│   ├── public/
│   ├── index.html
│   └── vite.config.js
├── .github/         # CI/CD 工作流
└── DEPLOYMENT.md    # 详细部署指南
```

## 部署

本项目需要分别部署后端和前端。

-   **后端**: 可以通过 Docker 或直接运行二进制文件来部署。
-   **前端**: 可以部署在 Vercel, Cloudflare Pages, Netlify 或任何静态文件服务器上。

**详细的部署步骤请参考 [DEPLOYMENT.md](./DEPLOYMENT.md)。**

## 开发

### 环境要求

-   Go 1.23+
-   Node.js 18+
-   Make

### 本地开发

本地开发需要同时运行后端和前端两个服务。

1.  **克隆仓库**
    ```bash
    git clone https://github.com/your_username/glog.git
    cd glog
    ```

2.  **运行后端服务**
    ```bash
    cd backend
    make run
    ```
    后端服务将启动在 `http://localhost:37371`。

3.  **运行前端服务** (在另一个终端中)
    ```bash
    cd frontend
    npm install
    npm run dev
    ```
    前端开发服务器将启动在 `http://localhost:5173` (或另一个可用端口)。

### 配置 API 地址

前端应用需要知道后端的 API 地址。这是通过 Vite 的环境变量和代理来管理的。

#### 1. 本地开发环境

在本地开发时，我们使用 Vite 的代理功能来解决跨域问题。

-   **文件**: `frontend/vite.config.js`
-   **配置**:
    ```javascript
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:37371', // 后端服务地址
          changeOrigin: true,
        },
      },
    },
    ```
    这个配置会将前端发出的所有 `/api` 开头的请求转发到 `http://localhost:37371`。因此，本地开发时**无需**额外配置。

#### 2. 生产部署环境

在将前端部署到生产环境时，你需要告诉它后端 API 的公网地址。

-   **方式**: 通过创建一个 `.env.production` 文件来设置环境变量。
-   **步骤**:
    1.  在 `frontend` 目录下创建一个名为 `.env.production` 的文件。
    2.  在该文件中添加以下内容：
        ```
        VITE_API_BASE_URL=https://your-backend-api.com
        ```
        将 `https://your-backend-api.com` 替换为你的后端服务的实际公网地址。
    3.  运行 `npm run build` 时，Vite 会自动读取这个环境变量，并将其打包到前端代码中。

    **注意**: 如果你使用 Vercel 或 Cloudflare Pages 等平台部署，可以直接在其网站的控制面板中设置名为 `VITE_API_BASE_URL` 的环境变量，效果相同。

### 构建

你可以使用 `Makefile` 来构建后端的二进制文件。

-   **构建 Linux 发行版**:
    ```bash
    cd backend
    make release
    ```
    构建产物 `glog-linux-amd64` 将出现在 `backend` 目录中。

## API 文档

Glog 提供了 RESTful API 用于文章管理。所有 API 都在 `/api` 路径下。

### 认证

后台管理相关 API 需要通过 Cookie Session 进行认证。外部工具调用的 API (`/api/v1/*`) 则需要通过 `Authorization: Bearer <token>` 进行认证。

### API 端点

请参考 `backend/main.go` 中的路由定义以获取完整的 API 端点列表。