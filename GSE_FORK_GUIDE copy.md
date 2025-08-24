# 指南：如何创建一个“纯净版”的 gse 分词库

本文档指导您如何通过 Fork `github.com/go-ego/gse` 并对其进行修改，来创建一个不包含任何内置字典的、轻量级的“纯净”版本，以解决 Go 二进制文件体积过大的问题。

## 步骤 1: Fork gse 仓库

1.  访问 `gse` 的官方 GitHub 仓库：[https://github.com/go-ego/gse](https://github.com/go-ego/gse)
2.  点击页面右上角的 "Fork" 按钮，将该仓库复刻到您自己的 GitHub 账户下。

## 步骤 2: 克隆您 Fork 的仓库到本地

将您刚刚 Fork 的仓库克隆到您的本地开发环境。请将 `[YourUsername]` 替换为您的 GitHub 用户名。

```bash
git clone https://github.com/[YourUsername]/gse.git
cd gse
```

## 步骤 3: 移除不必要的字典文件和代码

这是最关键的一步。我们需要删除 `gse` 库中所有与内置字典相关的文件和代码。

### 3.1 删除物理字典文件

执行以下命令，删除所有内置的字典数据。

```bash
rm -rf data/
```

### 3.2 清空 `dict_embed.go`

这个文件负责使用 `//go:embed` 指令嵌入字典。我们需要清空它。

用以下内容覆盖 `dict_embed.go` 文件：

```go
//go:build go1.16 && !ne
// +build go1.16,!ne

package gse

// This file is intentionally left blank to prevent embedding of default dictionaries.
```

### 3.3 “阉割” `dict_1.16.go`

这个文件包含了加载内置字典的硬编码逻辑。我们需要修改它，让这些逻辑失效。

用以下内容覆盖 `dict_1.16.go` 文件：

```go
// Copyright 2016 The go-ego Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-ego/gse/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

package gse

import (
	"strings"
)

// NewEmbed is modified to prevent loading any default embedded dictionaries.
// It now only handles the alpha/en flag.
func NewEmbed(dict ...string) (seg Segmenter, err error) {
	if len(dict) > 1 && (dict[1] == "alpha" || dict[1] == "en") {
		seg.AlphaNum = true
	}
	return
}

func (seg *Segmenter) loadZh() error {
	// Do nothing to prevent loading default dictionaries.
	return nil
}

func (seg *Segmenter) loadZhST(d string) (begin int, err error) {
	// Do nothing to prevent loading default dictionaries.
	return
}

// LoadDictEmbed is modified to only load dictionaries passed as a string,
// and ignore keywords like "zh", "ja", etc.
func (seg *Segmenter) LoadDictEmbed(dict ...string) (err error) {
	if len(dict) > 0 {
		d := dict[0]
		// Ignore keywords that would load default dictionaries.
		switch d {
		case "ja", "zh", "zh_s", "zh_t":
			return nil
		}

		if strings.Contains(d, ", ") && seg.DictSep != "," {
			s := strings.Split(d, ", ")
			for i := 0; i < len(s); i++ {
				err = seg.LoadDictStr(s[i])
			}
			return
		}

		err = seg.LoadDictStr(d)
		return
	}
	return nil
}

// LoadDictStr load the dictionary from dict path
func (seg *Segmenter) LoadDictStr(dict string) error {
	if seg.Dict == nil {
		seg.Dict = NewDict()
		seg.Init()
	}

	arr := strings.Split(dict, "\n")
	for i := 0; i < len(arr); i++ {
		s1 := strings.Split(arr[i], seg.DictSep+" ")
		size := len(s1)
		if size == 0 {
			continue
		}
		text := strings.TrimSpace(s1[0])

		freqText := ""
		if len(s1) > 1 {
			freqText = strings.TrimSpace(s1[1])
		}

		freq := seg.Size(size, text, freqText)
		if freq == 0.0 {
			continue
		}

		pos := ""
		if size > 2 {
			pos = strings.TrimSpace(strings.Trim(s1[2], "\n"))
		}

		words := seg.SplitTextToWords([]byte(text))
		token := Token{text: words, freq: freq, pos: pos}
		seg.Dict.AddToken(token)
	}

	seg.CalcToken()
	return nil
}

// LoadStopEmbed is modified to only load stop words passed as a string,
// and ignore keywords like "zh".
func (seg *Segmenter) LoadStopEmbed(dict ...string) (err error) {
	if len(dict) > 0 {
		d := dict[0]
		if strings.Contains(d, ", ") {
			s := strings.Split(d, ", ")
			// Ignore "zh" keyword
			start := 0
			if s[0] == "zh" {
				start = 1
			}
			for i := start; i < len(s); i++ {
				err = seg.LoadStopStr(s[i])
			}
			return
		}

		err = seg.LoadStopStr(d)
		return
	}
	return nil
}

// LoadStopStr load the stop dictionary from dict path
func (seg *Segmenter) LoadStopStr(dict string) error {
	if seg.StopWordMap == nil {
		seg.StopWordMap = make(map[string]bool)
	}

	arr := strings.Split(dict, "\n")
	for i := 0; i < len(arr); i++ {
		key := strings.TrimSpace(arr[i])
		if key != "" {
			seg.StopWordMap[key] = true
		}
	}

	return nil
}
```

## 步骤 4: 提交并推送您的修改

将您的修改提交到您 Fork 的仓库中。

```bash
git add .
git commit -m "feat: remove all embedded dictionaries to create a pure version"
git push origin master
```

## 步骤 5: 在您的项目中使用 Fork 后的版本

现在，回到您自己的项目（例如 `glog`）中。

### 5.1 修改 `go.mod` 文件

在 `go.mod` 文件中使用 `replace` 指令，将原始的 `gse` 模块指向您 Fork 的版本。

```go
replace github.com/go-ego/gse => github.com/[YourUsername]/gse [version]
```
*   `[YourUsername]`：您的 GitHub 用户名。
*   `[version]`：您刚刚提交的 commit 的哈希值，或者一个您创建的 tag (例如 `v0.80.3-pure`)。推荐使用 commit 哈希。您可以通过 `git rev-parse HEAD` 在您 Fork 的仓库中获取最新的 commit 哈希。

一个完整的例子可能看起来像这样：
```go
replace github.com/go-ego/gse => github.com/MyUser/gse v0.0.0-20250824052600-a1b2c3d4e5f6
```
(这里的 `v0.0.0-...` 是 Go Modules 使用 `replace` 配合 commit 哈希的标准格式)

### 5.2 整理依赖

运行 `go mod tidy`，Go 会自动下载并使用您 Fork 的版本。

```bash
go mod tidy
```

## 步骤 6: 在您的项目中集成自定义字典

现在，当您编译您的项目时，Go 将会使用您修改过的、不含任何内置字典的 `gse` 库。您需要像下面这样在自己的项目中嵌入并加载您的字典。

### 6.1 准备您的字典文件

确保您的字典文件（例如 `simplified.txt` 和 `stop_word.txt`）放在您的项目目录中。一个推荐的结构是：
```
your-project/
├── internal/
│   └── utils/
│       ├── dict/
│       │   ├── simplified.txt
│       │   └── stop_word.txt
│       ├── dict.go
│       └── segmenter.go
├── go.mod
└── main.go
```

**重要提示**: 如果您的字典文件很大，强烈建议先用 `gzip` 等工具将其压缩（例如 `simplified.txt.gz`），然后在代码中嵌入压缩后的版本，并在运行时解压。这可以进一步减小最终二进制文件的大小。

### 6.2 创建 `dict.go` 来嵌入字典

创建一个文件（例如 `internal/utils/dict.go`）专门用来嵌入您的字典数据。

```go
// internal/utils/dict.go
package utils

import _ "embed"

//go:embed dict/simplified.txt.gz
var simplifiedDictGz []byte

//go:embed dict/stop_word.txt.gz
var stopWordsGz []byte
```

### 6.3 创建 `segmenter.go` 来加载字典

创建分词器文件（例如 `internal/utils/segmenter.go`），在 `init()` 函数中初始化分词器并加载您嵌入的字典。

```go
// internal/utils/segmenter.go
package utils

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"strings"

	"github.com/go-ego/gse" // 这里将使用您 Fork 的版本
)

var seg gse.Segmenter

func init() {
	log.Println("Loading embedded dictionary and stop words...")

	// Decompress the simplified dictionary from dict.go
	simplifiedDict, err := decompressGzip(simplifiedDictGz)
	if err != nil {
		log.Fatalf("Failed to decompress embedded simplified dictionary: %v", err)
	}

	// Decompress the stop words from dict.go
	stopWords, err := decompressGzip(stopWordsGz)
	if err != nil {
		log.Fatalf("Failed to decompress embedded stop words: %v", err)
	}

	// Create a new segmenter instance directly to bypass any default dictionary loading.
	seg = gse.Segmenter{}

	// Load our custom dictionary from the decompressed string.
	err = seg.LoadDictEmbed(simplifiedDict)
	if err != nil {
		log.Fatalf("Failed to load embedded dictionary: %v", err)
	}

	// Load custom stop words from the decompressed string.
	err = seg.LoadStopEmbed(stopWords)
	if err != nil {
		log.Fatalf("Failed to load embedded stop words: %v", err)
	}
	log.Println("Custom dictionary and stop words loaded successfully.")
}

// decompressGzip takes a gzipped byte slice and returns the decompressed string.
func decompressGzip(data []byte) (string, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer r.Close()

	uncompressed, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(uncompressed), nil
}

// ... 您的分词函数 (SegmentTextForIndex, SegmentTextForQuery, etc.)
```

## 完成！

遵循以上步骤，您就可以在任何项目中使用这个“纯净版”的 `gse` 库，同时保持对字典的完全控制，并确保最终的二进制文件尽可能小。

最后，请记得手动删除您项目中的 `vendor` 目录（如果之前生成过）。
```bash
rm -rf vendor