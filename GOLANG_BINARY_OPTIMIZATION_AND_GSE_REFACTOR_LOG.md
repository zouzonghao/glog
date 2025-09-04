# Go 项目二进制文件瘦身与 `gse` 分词库内部化重构实战

## 1. 背景：一个 60MB 的“Hello World”

项目初始目标是在 `glog` 博客引擎中集成全文搜索功能。我们选择了 `gse` 作为中文分词库。然而，在嵌入了一个仅 5MB 大小的词典后，编译出的 Go 二进制文件大小异常地膨胀到了 **60MB**。这引发了一场从简单的编译优化到深度的代码重构的探索之旅。

本文档详细记录了整个问题的排查、分析、解决和重构过程，旨在为未来的 Go 项目性能优化和依赖管理提供一份宝贵的实战经验。

---

## 2. 排查与解决过程

### 第一阶段：初步诊断与常规优化

面对二进制文件过大的问题，我们首先尝试了 Go 语言中最常规的优化手段。

#### 2.1 编译标志：`-ldflags="-s -w"`

我们修改了 `Makefile`，在 `go build` 命令中加入了 `-ldflags="-s -w"` 标志，用于去除调试信息和符号表。

**`Makefile` (修改前):**
```makefile
build:
	go build -o glog main.go
```

**`Makefile` (修改后):**
```makefile
build:
	go build -ldflags="-s -w" -o glog main.go
```

**结果：** 文件大小从 60MB 降至 53MB。有效果，但显然没有触及问题的根源。

#### 2.2 Gzip 压缩嵌入

我们猜测嵌入的 5MB 词典是主要原因，因此尝试在构建时先用 Gzip 压缩词典，然后在程序启动时解压。

**`build.go` (关键代码):**
```go
// build.go
func compressFiles() error {
	for _, path := range filesToCompress {
		// ... (打开文件)
		gz := gzip.NewWriter(out)
		// ... (写入压缩数据)
	}
	return nil
}
```

**`api.go` (当时的文件):**
```go
//go:embed dict/simplified.txt.gz
var simplifiedDictGz []byte

func init() {
    simplifiedDict, err := decompressGzip(simplifiedDictGz)
    // ...
}
```

**结果：** 文件大小几乎没有变化。这表明问题并非源于我们自己嵌入的 5MB 词典。

### 第二阶段：深入分析，定位根源

常规优化手段失效后，我们意识到问题可能更深层。我们使用 macOS 的 `otool` 工具来分析二进制文件的内部结构。

```bash
otool -lV ./glog
```

**发现：** `otool` 的输出结果令人震惊。在 `__TEXT` 段中，有一个名为 `__rodata` (read-only data) 的节，其大小达到了惊人的 **35MB**！

**结论：** 这几乎可以肯定是 `gse` 库自身的问题。我们推断，无论我们如何使用 `gse`，它都会默认将一个巨大的内置词典编译进最终的二进制文件中。

### 第三阶段：最终方案：代码内部化与重构

为了彻底解决这个问题并获得完全的控制权，我们决定采取一个更彻底的方案：**将 `gse` 库的核心代码直接整合进我们的项目中，作为一个内部包。**

#### 3.1 迁移与整合

我们将 `gse` 库中必要的 `.go` 源文件（如 `segmenter.go`, `dictionary.go`, `dag.go`, `hmm.go` 等）复制到了项目的一个新目录 `internal/utils/segmenter` 中。

#### 3.2 架构调整

最初，我们将多个文件直接放在新包中，但这导致了大量的交叉引用和编译错误（例如 `undefined: Segmenter`），因为 Go 的编译器无法保证正确的初始化顺序。

最终的架构是将所有核心算法和数据结构合并到一个 `engine.go` 文件中，而将特定于本应用的初始化、`//go:embed` 指令和公共 API 放入一个独立的 `api.go` 文件中。

#### 3.3 漫长的调试之旅

这是整个过程中最艰难但最有价值的部分。

1.  **`redeclared` 编译错误**: 在合并文件后，我们遇到了大量的“变量重定义”错误。这通常是由于旧的文件被删除后，Go 的构建缓存或 IDE 索引没有及时更新。
    *   **解决方案**: 运行 `go clean -modcache` 和 `go mod tidy` 清理缓存和依赖。

2.  **逻辑错误：单字符分词**: 修复编译错误后，测试显示所有分词结果都被切成了单个汉字。
    *   **调试**: 通过添加大量日志，我们发现词典本身是加载成功的。
    *   **根源**: 问题出在 Viterbi 算法的实现上。我们错误地使用了 `math.Log(freq)` 来计算路径权重，而应该使用预先计算好的 `token.distance` (即负对数概率)。算法的目标是寻找**最小距离**路径，而不是最大频率路径。

    **`engine.go` (错误的核心逻辑):**
    ```go
    // routes = append(routes, route{freq: math.Log(freq) + routeMap[i+1].freq, index: i})
    // sort.Slice(routes, func(i, j int) bool {
    // 	return routes[i].freq > routes[j].freq
    // })
    ```

    **`engine.go` (修正后的核心逻辑):**
    ```go
    token, ok := seg.Dict.Find(toLower([]byte(string(word))))
    if ok && token != nil {
        routes = append(routes, route{distance: token.distance + routeMap[i+1].distance, index: i})
    }
    // ...
    sort.Slice(routes, func(i, j int) bool {
        return routes[i].distance < routes[j].distance
    })
    ```

3.  **逻辑错误：停用词过滤失效**: 核心分词功能正常后，我们发现停用词没有被过滤。
    *   **调试**: 日志显示停用词词典已加载，但在 map 中查找 "的"、"是" 等词时返回 `false`。
    *   **根源 (由用户发现！)**: 这是一个经典的数据问题。我们检查了 `stop_word.txt` 文件，发现里面根本没有 "这" 和 "是" 这两个词。这是一个深刻的教训：**当代码看起来正确时，一定要检查你的数据！**
    *   **解决方案**: 在 `internal/utils/segmenter/dict/stop_word.txt` 中添加缺失的停用词。

#### 3.4 移除 Gzip 依赖

根据用户的最终要求，我们移除了 Gzip 压缩步骤，以避免为最终的 UPX 压缩增加不必要的复杂性。

*   **修改 `build.go`**: 删除了所有 `compress/gzip` 相关的代码。
*   **修改 `api.go`**: 修改 `//go:embed` 指令，直接嵌入 `.txt` 文件，并删除了所有解压逻辑。

**`api.go` (最终版本):**
```go
package segmenter

import (
	"bufio"
	_ "embed"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
)

//go:embed dict/simplified.txt
var simplifiedDict []byte

//go:embed dict/stop_word.txt
var stopWords []byte

//go:embed hmm/prob_emit.go
var probEmit []byte

// ... init() 和其他函数 ...

func init() {
	// ...
	totalFreq := loadDictFromString(&seg, string(simplifiedDict))
	recalculateTokenDistances(seg.Dict, totalFreq)
	loadStopFromString(&seg, string(stopWords))
	// ...
}
```

### 第四阶段：大功告成

在修复了所有问题后，我们更新了项目中所有调用分词函数的地方 (`post_service.go`, `admin.go`, `main.go`)，使其指向新的 `internal/utils/segmenter` 包。

最终，`make release` 命令成功执行，生成了一个功能完整、高度优化且无外部二进制依赖的分词器。

---

## 3. 总结与经验

这次从 60MB 到最终优化版本的旅程，给我们带来了许多宝贵的经验：

1.  **警惕第三方库的“黑盒”**: 许多库为了开箱即用，会包含大量默认资源，这可能导致二进制文件异常膨胀。
2.  **善用原生工具分析问题**: `otool` (Linux 上的 `nm` 或 `objdump`) 是深入理解二进制文件结构、定位问题的强大武器。
3.  **代码内部化是终极控制手段**: 当你无法通过常规手段控制一个库的行为时，将其代码直接纳入你的项目（Internalization）可以让你获得完全的控制权，尽管这需要更多的工作。
4.  **系统性调试至关重要**: 面对复杂问题，通过“假设-验证-排除”的循环，并辅以详细的日志，是找到问题根源的唯一途径。
5.  **代码没错时，检查你的数据**: 这是本次调试中最深刻的教训之一。一个看似复杂的逻辑问题，最终可能只是源于一个简单的数据缺失。
6.  **重构是一项系统工程**: 大规模的代码移动和重构，必须仔细检查并更新所有调用点，否则会导致编译失败。
7.  **理解并利用 Go 工具链**: `go mod tidy` 和 `go clean -modcache` 是解决 Go 模块和缓存问题的有力工具。

通过这次实践，我们不仅解决了一个棘手的性能问题，还获得了一个完全由我们掌控、高度优化的内部分词库，并积累了宝贵的 Go 项目工程经验。