# 部署指南

本文档将指导你如何在生产环境中部署和配置 Glog 前后端分离应用。

## 环境变量配置

为了确保应用在生产环境中的安全性和正确性，你需要配置以下环境变量。

### 1. 前端环境变量 (Astro)

在你的前端部署平台（如 Vercel, Netlify 等）上，你需要设置以下环境变量。通常，你可以在项目的设置页面中找到环境变量配置选项。

| 变量名 | 示例值 | 说明 |
| :--- | :--- | :--- |
| `PUBLIC_API_URL` | `https://api.your-domain.com` | **必须。** 指向你部署的 Go 后端服务的公开访问地址。 |

### 2. 后端环境变量 (Go)

在你的后端服务器或服务（如 Docker, Systemd, PM2 等）的启动环境中，你需要设置以下环境变量。

| 变量名 | 示例值 | 说明 |
| :--- | :--- | :--- |
| `FRONTEND_URL` | `https://www.your-domain.com` | **必须。** 你的前端应用的公开访问地址。用于 CORS 配置，确保后端只接受来自你前端的请求。 |
| `COOKIE_DOMAIN` | `.your-domain.com` | **必须。** 用于设置会话 Cookie 的域名。**注意**：前面的点 `.` 非常重要，它允许 Cookie 在所有子域名（如 `www` 和 `api`）之间共享。 |
| `SESSION_SECRET` | `a_very_long_and_random_string_here` | **必须。** 用于加密和签名会话 Cookie 的密钥。请使用一个长且无法预测的随机字符串以确保安全。你可以使用密码生成器来创建一个。 |
| `GIN_MODE` | `release` | **推荐。** 将 Gin 框架设置为生产模式，以获得更好的性能和更少的日志输出。 |

## 部署示例

假设你的域名是 `my-awesome-blog.com`。

#### 前端配置 (Vercel, Netlify, etc.)

- `PUBLIC_API_URL` = `https://api.my-awesome-blog.com`

#### 后端配置 (服务器, Docker, etc.)

- `FRONTEND_URL` = `https://www.my-awesome-blog.com`
- `COOKIE_DOMAIN` = `.my-awesome-blog.com`
- `SESSION_SECRET` = `use_a_strong_random_secret_here_12345`
- `GIN_MODE` = `release`

通过以上配置，你的前后端应用就可以在不同的服务器上独立部署，并通过跨域 Cookie 实现安全可靠的用户认证。