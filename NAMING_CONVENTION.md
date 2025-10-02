# Glog 项目全栈命名规范

本文档旨在为 Glog 项目建立一套统一、清晰且跨语言的命名规范，以提高代码的可读性、可维护性，并从根本上避免因命名不一致导致的数据解析问题。

所有新代码都应严格遵守此规范。现有代码将根据本文档附带的重构计划进行统一。

---

## 核心原则

**在任何跨语言、跨服务的边界（例如 API），数据交换格式必须统一使用 `snake_case`（下划线命名法）。**

- **适用范围**: 所有 RESTful API 的 JSON 请求体、响应体、URL 查询参数。
- **示例**: `user_id`, `published_at`, `has_next`。
- **理由**: `snake_case` 是 Go 和 Web API 中广泛接受的惯例，能有效避免因大小写处理不当（如 `HasNext` vs `hasNext`）引发的难以调试的 bug。

---

## 各层级具体规范

### 1. Go 后端

- **结构体字段 (Struct Fields)**:
  - **命名**: 遵循 Go 语言官方推荐，使用 `PascalCase`（大驼峰命名法）。
  - **示例**: `PublishedAt`, `IsPrivate`。
  - **JSON 标签**: **必须**为所有需要通过 API 暴露的字段添加 `json:"snake_case"` 标签，以确保对外接口的统一。
    ```go
    type Post struct {
        PublishedAt time.Time `json:"published_at"`
        IsPrivate   bool      `json:"is_private"`
    }
    ```

- **变量与函数 (Variables & Functions)**:
  - **命名**: 遵循 Go 语言官方推荐，使用 `camelCase`（小驼峰命名法）。
  - **示例**: `getPosts`, `currentUser`。

### 2. TypeScript / Svelte 前端

- **类型与接口 (Types & Interfaces)**:
  - **命名**: 使用 `PascalCase`。
  - **示例**: `interface Post`, `type UserProfile`。
  - **字段命名**: **必须**使用 `snake_case`，以与后端 API 的 JSON 输出完全匹配。
    ```typescript
    export interface Post {
        published_at: string;
        is_private: boolean;
    }
    ```

- **变量与函数 (Variables & Functions)**:
  - **命名**: 使用 `camelCase`。
  - **示例**: `getPosts`, `currentUser`。

- **组件 Props (Svelte/Astro)**:
  - **命名**: 使用 `camelCase`。
  - **示例**: `<CardView initialPosts={posts} />`

### 3. 数据库

- **表名 (Table Names)**:
  - **命名**: 使用复数的 `snake_case`。
  - **示例**: `posts`, `user_settings`。

- **列名 (Column Names)**:
  - **命名**: 使用单数的 `snake_case`。
  - **示例**: `published_at`, `is_private`。
  - **备注**: GORM 会自动将 Go 结构体的 `PascalCase` 字段映射为 `snake_case` 的列名，因此只要遵循 Go 的规范即可。

---

## 重构清单

以下是根据此规范需要进行重构的关键点：

1.  **`backend/internal/handlers/admin.go`**:
    -   **目标**: 将 `ListPosts` 处理器中所有本地定义的、使用 `camelCase` JSON 标签的结构体，全部替换为 `models` 包中已有的、使用 `snake_case` 标签的全局模型。
    -   **具体修改**:
        -   移除本地的 `PostResponseForAdmin` 结构体，直接使用 `models.Post` 或 `models.RenderedPost`。
        -   移除本地的 `PaginationResponse` 结构体，直接使用 `models.Pagination`。
        -   调整 `AdminPostsResponse` 以引用正确的模型。

2.  **前端 Admin 相关组件 (例如 `AdminManager.svelte`)**:
    -   **目标**: 在前端，所有调用 `/api/admin/posts` 接口并处理其数据的地方，都需要将对 `camelCase` 字段（如 `currentPage`）的引用，修改为对 `snake_case` 字段（如 `current_page`）的引用。
