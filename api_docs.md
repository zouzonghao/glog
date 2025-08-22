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

### 文章

#### 1. 创建文章

创建一个新的文章。

*   **URL**: `/api/v1/posts`
*   **Method**: `POST`
*   **Headers**:
    *   `Authorization: Bearer <token>`
    *   `Content-Type: application/json`
*   **Body**:

    ```json
    {
      "title": "文章标题",
      "content": "文章内容",
      "published": true,
      "is_private": false,
      "with_ai": false
    }
    ```

*   **成功响应 (201 Created)**:

    ```json
    {
        "ID": 1,
        "CreatedAt": "2023-10-27T10:00:00Z",
        "UpdatedAt": "2023-10-27T10:00:00Z",
        "DeletedAt": null,
        "title": "文章标题",
        "slug": "article-title",
        "content": "文章内容",
        "excerpt": "文章摘要",
        "published": true,
        "is_private": false,
        "published_at": "2023-10-27T10:00:00Z"
    }
    ```

#### 2. 查找文章

查找文章，支持多关键字搜索和分页。

*   **URL**: `/api/v1/posts`
*   **Method**: `GET`
*   **Headers**:
    *   `Authorization: Bearer <token>`
*   **查询参数**:
    *   `q` (可选): 搜索关键字，多个关键字用逗号分隔。例如: `golang,api`
    *   `page` (可选): 页码，默认为 `1`。
    *   `pageSize` (可选): 每页数量，默认为 `15`。

*   **成功响应 (200 OK)**:

    ```json
    {
        "posts": [
            {
                "ID": 1,
                "CreatedAt": "2023-10-27T10:00:00Z",
                "UpdatedAt": "2023-10-27T10:00:00Z",
                "DeletedAt": null,
                "title": "文章标题",
                "slug": "article-title",
                "content": "文章内容",
                "excerpt": "文章摘要",
                "published": true,
                "is_private": false,
                "published_at": "2023-10-27T10:00:00Z"
            }
        ],
        "total": 1
    }