# Glog API 文档

本文档描述了 Glog 博客系统的 API。

## 认证

所有 API 请求都需要通过 `Authorization` 请求头进行认证。认证方式为 `Bearer Token`，其中 `Token` 是你的站点密码。

**示例:**

```
Authorization: Bearer your_site_password
```

如果认证失败，API 将返回 `401 Unauthorized` 错误。

## API 端点

### 1. 获取文章列表

获取文章列表，支持分页。

*   **URL**: `/api/v1/posts`
*   **Method**: `GET`
*   **Headers**:
    *   `Authorization: Bearer <token>`
*   **查询参数**:
    *   `page` (可选): 页码，默认为 `1`
    *   `page_size` (可选): 每页数量，默认为 `10`

*   **成功响应 (200 OK)**:

    ```json
    {
        "posts": [
            {
                "id": 1,
                "title": "文章标题",
                "slug": "article-title",
                "excerpt": "文章摘要描述",
                "has_cover": true,
                "cover": "https://example.com/cover.jpg",
                "published_at": "2025-01-01 12:00:00",
                "is_private": false
            }
        ],
        "total": 100,
        "page": 1,
        "page_size": 10
    }
    ```

### 2. 获取文章详情

获取单篇文章的完整内容。

*   **URL**: `/api/v1/posts/:id`
*   **Method**: `GET`
*   **Headers**:
    *   `Authorization: Bearer <token>`
*   **路径参数**:
    *   `id`: 文章 ID

*   **成功响应 (200 OK)**:

    ```json
    {
        "id": 1,
        "title": "文章标题",
        "slug": "article-title",
        "content": "文章原始 Markdown 内容",
        "cover": "https://example.com/cover.jpg",
        "published_at": "2025-01-01 12:00:00"
    }
    ```

*   **错误响应**:
    *   `400 Bad Request`: 无效的文章 ID
    *   `404 Not Found`: 文章不存在

### 3. 更新文章摘要

更新指定文章的摘要描述。

*   **URL**: `/api/v1/posts/:id/excerpt`
*   **Method**: `PUT`
*   **Headers**:
    *   `Authorization: Bearer <token>`
    *   `Content-Type: application/json`
*   **路径参数**:
    *   `id`: 文章 ID
*   **Body**:

    ```json
    {
        "excerpt": "新的文章摘要，最多 500 字符"
    }
    ```

*   **成功响应 (200 OK)**:

    ```json
    {
        "status": "success"
    }
    ```

*   **错误响应**:
    *   `400 Bad Request`: 无效的文章 ID 或请求格式错误
    *   `500 Internal Server Error`: 更新失败

### 4. 更新文章封面

更新指定文章的封面图片 URL。

*   **URL**: `/api/v1/posts/:id/cover`
*   **Method**: `PUT`
*   **Headers**:
    *   `Authorization: Bearer <token>`
    *   `Content-Type: application/json`
*   **路径参数**:
    *   `id`: 文章 ID
*   **Body**:

    ```json
    {
        "cover": "https://example.com/new-cover.jpg"
    }
    ```

*   **成功响应 (200 OK)**:

    ```json
    {
        "status": "success"
    }
    ```

*   **错误响应**:
    *   `400 Bad Request`: 无效的文章 ID 或请求格式错误
    *   `500 Internal Server Error`: 更新失败

## 错误响应格式

所有错误响应遵循以下格式：

```json
{
    "error": "错误描述信息"
}
```

## 使用场景

这些 API 设计用于与外部 AI 服务集成：

1. **获取文章列表** - 外部程序可以获取所有文章的基本信息，包括摘要和是否有封面图
2. **获取文章内容** - 外部程序可以读取文章的原始 Markdown 内容进行分析
3. **更新摘要** - AI 服务生成摘要后，通过 API 更新到文章
4. **更新封面** - AI 服务生成封面图后，通过 API 更新封面 URL
