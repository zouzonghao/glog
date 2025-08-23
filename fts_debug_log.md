# Glog 全文搜索（FTS）深度调试与修复日志

本文档记录了一系列关于 Glog 项目全文搜索功能的深度调试过程，旨在追踪和解决从搜索结果不一致到程序报错等多个复杂问题。

## 核心经验总结：正确实现 Go + FTS5 + gse

### 1. 理解 FTS5 的两种模式，避免“幽灵索引”

-   **普通模式 (本项目最终选择)**: FTS 表存储分词后内容的副本。当 `CREATE VIRTUAL TABLE` 语句**不包含** `content=` 选项时启用。这是在应用层进行自定义分词的**正确选择**。FTS 引擎会自动创建 `_content` 等后台表来存储副本。
-   **Contentless 模式**: FTS 表只存储索引，不存储内容。当 `CREATE VIRTUAL TABLE` 语句**包含** `content='主表名'` 选项时启用。在这种模式下，FTS 引擎会**完全忽略**应用层传入的分词内容，而是返回主表读取原文，并使用**其内置的分词器**。
-   **陷阱**: `Contentless` 模式 + 数据库触发器会导致应用层的分词逻辑（如 `gse`）完全失效，产生难以调试的“幽灵索引”问题。

### 2. `gse` 嵌入式词典的最佳实践

为了创建无外部文件依赖的、可独立部署的 Go 应用，嵌入式词典是最佳选择。

**Step 1: 在 Go 文件中嵌入词典**
在 `internal/utils/segmenter.go` 中，使用 `//go:embed` 指令，并确保导入 `embed` 包。

```go
package utils

import (
	_ "embed" // 必须使用空导入
	// ...
)

//go:embed dict/simplified.txt
var simplifiedDict string

//go:embed dict/stop_word.txt
var stopWords string
```

**Step 2: 初始化分词器并加载停用词**
在 `init()` 函数中，分两步进行：

```go
func init() {
	// 1. 使用 NewEmbed 创建分词器，加载主词典和启用其他语言
	// "zh," + simplifiedDict 表示加载默认中文词典并追加自定义词典
	// "en" 表示启用内置的英文分词器
	seg, err = gse.NewEmbed("zh,"+simplifiedDict, "en")
	if err != nil {
		log.Fatalf("Failed to create segmenter: %v", err)
	}

	// 2. 使用 LoadStopEmbed 单独加载停用词
	err = seg.LoadStopEmbed(stopWords)
	if err != nil {
		log.Fatalf("Failed to load stop words: %v", err)
	}
}
```

**Step 3: 在分词流程中应用停用词**
加载停用词列表后，必须在分词流程中显式调用 `seg.Trim()` 来应用它。这是一个容易被忽略的关键步骤。

```go
func SegmentTextForIndex(text string) string {
	// 1. 分词
	words := seg.Cut(text, true)

	// 2. 应用停用词列表（同时也会移除空格等）
	trimmedWords := seg.Trim(words)

	// 3. 拼接成最终的字符串
	return strings.Join(trimmedWords, " ")
}
```

### 3. 保持索引和查询逻辑的绝对一致

用于建立索引的函数 (`SegmentTextForIndex`) 和用于处理用户搜索的函数 (`SegmentTextForQuery`) 必须拥有**完全相同**的处理流水线 (`Cut` -> `Trim`)，以确保用户搜索的词能精确匹配到索引中的词。

### 4. 处理无效查询的边界情况

用户的输入经过分词和停用词过滤后，可能变为空字符串（例如，只输入了标点符号）。必须在将查询发送到数据库之前进行检查，否则 `MATCH ""` 会导致 FTS5 语法错误。

```go
// In post_service.go
func (s *PostService) SearchPublishedPostsPage(...) {
    // ...
	ftsQuery := utils.SegmentTextForQuery(query)
	if ftsQuery == "" {
		return []models.Post{}, 0, nil // 直接返回空结果，避免数据库错误
	}
    // ...
}
```

---
## 详细调试历史

### 问题一：新旧文章搜索效果不一致
... (省略，内容同之前版本) ...

### 问题二：旧文章索引无法被新分词引擎覆盖（核心问题）
... (省略，内容同之前版本) ...

### 问题三：修复后，特定词语（如“单线程”）无法搜索
... (省略，内容同之前版本) ...

### 问题四：搜索内容包含英文逗号时程序报错
... (省略，内容同之前版本) ...

### 问题五：修复问题四时引入的新 Bug
... (省略，内容同之前版本) ...

### 问题六：只搜索标点符号时程序报错（最终 Bug）
... (省略，内容同之前版本) ...