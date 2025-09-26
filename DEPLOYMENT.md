# 部署指南

本项目已改造为前后端分离架构，需要分别部署后端服务和前端应用。

## 后端部署 (Go API)

后端是一个 Go 语言编写的纯 API 服务。

### 1. 编译

首先，进入 `backend` 目录：
```bash
cd backend
```

然后，你可以使用 `Makefile` 来方便地构建或运行。

#### 使用 Makefile (推荐)

构建 `linux/amd64` 发行版：
```bash
make release
```
这将在 `backend` 目录下生成 `glog-linux-amd64` 二进制文件。

在本地运行开发服务器：
```bash
make run
```

#### 手动编译

如果你想手动编译，可以使用以下命令。以 Linux x86_64 为例：
```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o glog-server .
```
**注意**: 由于使用了 `sqlite3`，需要启用 `CGO_ENABLED=1`。

### 2. 运行

将编译后的二进制文件（例如 `glog-linux-amd64` 或 `glog-server`）和 `glog.db` 数据库文件上传到你的服务器的同一目录。

在服务器上运行：
```bash
./glog-linux-amd64
```

服务将默认在 `:37371` 端口启动。建议使用 `systemd` 或 `supervisor` 等工具来管理进程，确保服务在后台持续运行。

### 3. CORS 配置

在生产环境中，为了安全起见，你需要修改 `backend/main.go` 中的 CORS 配置，将 `AllowOrigins` 设置为你的前端应用的实际域名。

```go
// backend/main.go

// ...
	config := cors.DefaultConfig()
	// 将 "*" 替换为你的前端域名，例如 "https://your-frontend.com"
	config.AllowOrigins = []string{"https://your-frontend.com"} 
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
// ...
```

## 前端部署 (Vue App)

前端是一个使用 Vue 和 Vite 构建的静态单页面应用 (SPA)，非常适合部署在 Vercel、Cloudflare Pages 或 Netlify 等静态托管平台。

### 1. 构建项目

在 `frontend` 目录下，运行以下命令来构建生产版本的静态文件：

```bash
cd frontend
npm run build
```

这将在 `frontend/dist` 目录下生成所有需要的 HTML, CSS, 和 JavaScript 文件。

### 2. 部署到 Vercel / Cloudflare Pages

1.  将你的整个项目（包括 `frontend` 目录）推送到一个 GitHub/GitLab 仓库。
2.  在 Vercel 或 Cloudflare Pages 上，选择导入你的 Git 仓库。
3.  在构建设置中，进行如下配置：
    *   **构建命令**: `npm run build`
    *   **输出目录**: `frontend/dist`
    *   **根目录**: `frontend`
4.  添加一个环境变量，用于告诉前端应用后端的 API 地址：
    *   **Key**: `VITE_API_BASE_URL`
    *   **Value**: `https://your-backend-api.com` (你的后端服务的公开访问地址)
5.  开始部署。

部署完成后，你的前端应用就可以通过静态托管平台提供的域名进行访问了。