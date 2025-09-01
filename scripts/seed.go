package main

import (
	"fmt"
	"glog/internal/models"
	"glog/internal/utils"
	"log"
	"regexp"
	"time"

	"github.com/gosimple/slug"
)

const (
	TotalPosts = 1000
	Content    = `
# Glog 性能测试文章

这是一篇由脚本生成的文章，用于 Glog 博客系统的性能和负载测试。

## Markdown 特性测试

### 列表

- 项目一
- 项目二
- 项目三

### 引用

> “性能测试是确保应用程序在高负载下稳定运行的关键步骤。”

### 代码块

` + "```go" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
` + "```" + `

## 长文本模拟

为了模拟真实的文章长度，这里会重复一段文本。Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed non risus. Suspendisse lectus tortor, dignissim sit amet, adipiscing nec, ultricies sed, dolor. Cras elementum ultrices diam. Maecenas ligula massa, varius a, semper congue, euismod non, mi. Proin porttitor, orci nec nonummy molestie, enim est eleifend mi, non fermentum diam nisl sit amet erat. Duis semper. Duis arcu massa, scelerisque vitae, consequat in, pretium a, enim. Pellentesque congue. Ut in risus volutpat libero pharetra tempor. Cras vestibulum bibendum augue. Praesent eas leo in pede.

<!--more-->

这是摘要分割线之后的内容。这部分内容应该只在文章详情页显示，而不会出现在首页的文章列表中。我们将在这里填充更多的重复文本来增加文章的体积，从而更好地测试 Markdown 渲染和数据库查询的性能。

Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Vestibulum tortor quam, feugiat vitae, ultricies eget, tempor sit amet, ante. Donec eu libero sit amet quam egestas semper. Aenean ultricies mi vitae est. Mauris placerat eleifend leo. Quisque sit amet est et sapien ullamcorper pharetra. Vestibulum erat wisi, condimentum sed, commodo vitae, ornare sit amet, wisi. Aenean fermentum, elit eget tincidunt condimentum, eros ipsum rutrum orci, sagittis tempus lacus enim ac dui. Donec non enim in turpis pulvinar facilisis. Ut felis. Praesent dapibus, neque id cursus faucibus, tortor neque egestas augue, eu vulputate magna eros eu erat. Aliquam erat volutpat. Nam dui mi, tincidunt quis, accansan porttitor, facilisis luctus, metus.
`
)

// processAndRenderContent is a simplified, standalone version for the seed script.
func processAndRenderContent(md string) (string, error) {
	separatorRegex := regexp.MustCompile(`<!--\s*more\s*-->`)
	parts := separatorRegex.Split(md, 2)

	if len(parts) > 1 {
		summaryMd := parts[0]
		bodyMd := parts[1]
		summaryHtml, err := utils.RenderMarkdown(summaryMd)
		if err != nil {
			return "", fmt.Errorf("摘要渲染失败: %w", err)
		}
		bodyHtml, err := utils.RenderMarkdown(bodyMd)
		if err != nil {
			return "", fmt.Errorf("正文渲染失败: %w", err)
		}
		return fmt.Sprintf("<blockquote class=\"post-summary\">%s</blockquote>%s", summaryHtml, bodyHtml), nil
	}

	fullHtml, err := utils.RenderMarkdown(md)
	if err != nil {
		return "", fmt.Errorf("全文渲染失败: %w", err)
	}
	return string(fullHtml), nil
}

func main() {
	log.Println("开始连接数据库...")
	db, err := utils.InitDatabase("")
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("数据库连接成功。")

	log.Println("清空旧的文章数据...")
	if err := db.Exec("DELETE FROM posts").Error; err != nil {
		log.Fatalf("清空 posts 表失败: %v", err)
	}
	if err := db.Exec("VACUUM").Error; err != nil {
		log.Printf("VACUUM failed: %v", err)
	}
	log.Println("旧数据已清空。")

	log.Printf("准备生成 %d 篇文章...\n", TotalPosts)

	for i := 1; i <= TotalPosts; i++ {
		title := fmt.Sprintf("性能测试文章 %d", i)
		content := fmt.Sprintf("这是文章 %d 的内容。\n\n%s", i, Content)

		// Pre-render HTML content
		htmlContent, err := processAndRenderContent(content)
		if err != nil {
			log.Printf("渲染文章失败 %d: %v\n", i, err)
			continue
		}

		post := models.Post{
			Title:       title,
			Slug:        slug.Make(title),
			Content:     content,
			ContentHTML: htmlContent, // Save the pre-rendered HTML
			IsPrivate:   false,
			PublishedAt: time.Now(),
		}

		result := db.Create(&post)
		if result.Error != nil {
			log.Printf("创建文章失败 %d: %v\n", i, result.Error)
		}

		if i%100 == 0 {
			log.Printf("已生成 %d/%d 篇文章...\n", i, TotalPosts)
		}
	}

	log.Printf("成功生成 %d 篇文章。\n", TotalPosts)
}
