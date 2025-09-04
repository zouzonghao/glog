# 全文搜索（FTS）模块代码审查报告

## 1. 总结

经过一系列修复，搜索模块的核心功能已恢复正常且逻辑正确。本次审查的目标是识别残余的潜在风险、性能瓶颈和可维护性问题。

审查发现，当前实现虽然功能可用，但在**搜索结果相关性排序**、**后台索引任务的健壮性**和**数据库查询效率**方面有显著的提升空间。

---

## 2. 分模块审查详情

### 2.1. `internal/utils/database.go` (数据库初始化)

-   **现状**: FTS 表被正确地初始化为普通虚拟表，依赖应用层进行分词和同步。
-   **潜在隐患**:
    -   **索引重建机制缺失**: 当前，修复索引问题的唯一方法是手动删除 `blog.db` 文件。这在生产环境中是不可接受的。如果未来需要更换分词词典或修复分词 Bug，没有一个内置的、安全的方式来重建所有文章的索引。
-   **优化建议**:
    -   **增加索引重建接口**: 在管理后台增加一个“重建全站索引”的按钮。其后端逻辑为：
        1.  清空 `posts_fts` 表 (`DELETE FROM posts_fts`)。
        2.  遍历 `posts` 表中的所有文章。
        3.  为每一篇文章调用 `UpdateFtsIndex` 方法，重新生成索引。
        -   这提供了一个安全、可控的方式来刷新索引，而无需删除整个数据库。

### 2.2. `internal/utils/segmenter.go` (分词器)

-   **现状**: 索引和查询使用统一的 `gse.Cut` 分词，过滤逻辑基于 `unicode` 标准，实现正确且健壮。
-   **潜在隐患**:
    -   **词典不可扩展**: 当前使用的是 `gse` 的内置默认词典。对于特定领域的博客（如技术、医学），可能会有内置词典无法正确切分的专有词汇，影响搜索精度。
-   **优化建议**:
    -   **支持自定义词典**: 增加从外部文件（如 `user_dict.txt`）加载自定义词典的功能。`gse` 库支持 `seg.LoadDict("user_dict.txt")` 这样的调用。这能极大地提升对特定领域词汇的分词准确性。

### 2.3. `internal/services/post_service.go` (业务逻辑)

-   **现状**: 异步索引更新逻辑正确，空查询已处理。
-   **潜在隐患**:
    -   **后台任务失败静默**: `asyncPostSaveOperations` 是一个 `go` 协程，即“发射后不管”。如果在这个函数中，更新索引的操作 `UpdateFtsIndex` 因为任何原因（如数据库锁定、磁盘空间满）失败了，它只会在后台打印一条日志。用户和系统都无法感知到这次失败，导致对应文章的索引变得陈旧或缺失，且问题难以排查。**这是目前最大的一个健壮性隐患**。
-   **优化建议**:
    -   **引入失败重试或状态标记**:
        -   **方案A (简单)**: 在 `post` 模型中增加一个字段，如 `fts_state` (枚举值: `synced`, `pending`, `failed`)。当异步任务失败时，将状态设为 `failed`。后台可以有一个定时任务或管理界面来重试失败的索引任务。
        -   **方案B (更专业)**: 引入一个轻量级的后台任务队列（如 `Redis` 或基于数据库的队列），将索引任务放入队列。由专门的 worker 来处理，可以提供失败重试、错误告警等更完善的机制。

### 2.4. `internal/repository/post_repo.go` (数据库查询)

-   **现状**: 使用 `IN (SELECT ...)` 子查询的方式来获取匹配的文章。
-   **潜在隐患**:
    1.  **性能问题**: 在文章数量巨大时，`IN` 子查询的性能可能劣于 `JOIN`。
    2.  **严重缺陷 - 未使用相关性排序**: 当前的搜索结果是按照 `published_at desc` (发布时间) 排序的。这完全浪费了 FTS5 引擎计算出的**相关性分数 (rank)**。一个好的搜索引擎，最应该做的是把最匹配用户查询的结果排在最前面，而不是最新的。
-   **优化建议**:
    -   **使用 `JOIN` 替代 `IN`**: 将查询语句修改为 `JOIN`，通常数据库的查询优化器能更好地处理 `JOIN`。
    -   **按 `rank` 排序 (最高优先级)**: 修改查询，使其按 FTS5 的 `rank` 值进行排序。`rank` 值越高，代表文章与查询的匹配度越高。

    **建议的 `SearchPage` 函数修改示例**:
    ```go
    // SearchPage searches published posts with pagination using FTS, respecting login status.
    func (r *PostRepository) SearchPage(ftsQuery string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
        var posts []models.Post

        // 使用 JOIN 并按 rank 排序
        dbQuery := r.db.Table("posts").
            Select("posts.*, posts_fts.rank").
            Joins("JOIN posts_fts ON posts.id = posts_fts.rowid").
            Where("posts_fts MATCH ?", ftsQuery).
            Where("posts.published = ?", true)

        if !isLoggedIn {
            dbQuery = dbQuery.Where("posts.is_private = ?", false)
        }

        offset := (page - 1) * pageSize
        // ORDER BY rank (descending) 是 FTS 搜索的核心
        err := dbQuery.Order("posts_fts.rank DESC").Offset(offset).Limit(pageSize).Find(&posts).Error
        return posts, err
    }
    ```
    **注意**: `CountByQuery` 函数可以保持不变，因为它只关心数量，不关心排序。

---

## 3. 总结与优先级建议

1.  **最高优先级**: **按 `rank` 排序** (`post_repo.go`)。这是对搜索功能“质量”的根本性提升，能让用户体验得到巨大改善。
2.  **高优先级**: **处理后台索引失败** (`post_service.go`)。这解决了系统健壮性的核心隐患，避免了“静默失败”导致的数据不一致。
3.  **中优先级**: **增加索引重建功能** (`database.go` 及管理后台)。这为未来的系统维护提供了极大的便利。
4.  **低优先级**: **支持自定义词典** (`segmenter.go`)。这是一个功能增强，可以根据实际需求来决定何时实现。