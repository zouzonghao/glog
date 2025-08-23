//go:build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

var (
	m = minify.New()
	// assetReplacements 定义了需要在 HTML 文件中进行的替换规则。
	// key 是原始文件名, value 是替换后的文件名。
	assetReplacements = map[string]string{
		// CSS Files
		"style.css": "style.min.css",
		"prism.css": "prism.min.css",
		"404.css":   "404.min.css",
		// JS Files
		"main.js":  "main.min.js",
		"prism.js": "prism.min.js",
	}
)

func init() {
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/javascript", js.Minify)
}

func main() {
	release := flag.Bool("release", false, "Process assets for release")
	clean := flag.Bool("clean", false, "Clean processed assets and restore original files")
	flag.Parse()

	if *release && *clean {
		log.Fatal("Cannot use -release and -clean flags simultaneously.")
	}

	if *release {
		fmt.Println("Processing assets for release...")
		if err := processAssets(false); err != nil {
			log.Fatalf("Failed to process assets for release: %v", err)
		}
		fmt.Println("Assets processed successfully.")
	} else if *clean {
		fmt.Println("Cleaning up processed assets...")
		if err := cleanupAssets(); err != nil {
			log.Fatalf("Failed to clean up assets: %v", err)
		}
		fmt.Println("Cleanup complete.")
	} else {
		fmt.Println("No action specified. Use -release to process assets or -clean to clean up.")
	}
}

// processAssets 处理静态资源。isClean 控制是替换为 .min 版本还是还原。
func processAssets(isClean bool) error {
	// 1. 压缩文件 (仅在非 clean 模式下)
	if !isClean {
		fmt.Println("Minifying CSS and JS files...")
		assetDirs := []string{"static/css", "static/js"}
		for _, dir := range assetDirs {
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					if strings.HasSuffix(path, ".css") && !strings.HasSuffix(path, ".min.css") {
						return minifyFile(path, "text/css")
					}
					if strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".min.js") {
						return minifyFile(path, "text/javascript")
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("error walking directory %s: %w", dir, err)
			}
		}
	}

	// 2. 更新 HTML 引用
	fmt.Println("Updating HTML references...")
	return updateHTMLReferences(isClean)
}

func cleanupAssets() error {
	// 1. 还原 HTML 引用
	fmt.Println("Restoring HTML references...")
	if err := updateHTMLReferences(true); err != nil {
		return fmt.Errorf("failed to restore HTML references: %w", err)
	}

	// 2. 删除 .min 文件
	fmt.Println("Deleting minified files...")
	assetDirs := []string{"static/css", "static/js"}
	for _, dir := range assetDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if strings.HasSuffix(path, ".min.css") || strings.HasSuffix(path, ".min.js") {
					fmt.Printf("  - Deleting %s\n", path)
					if err := os.Remove(path); err != nil {
						return fmt.Errorf("failed to delete %s: %w", path, err)
					}
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error cleaning directory %s: %w", dir, err)
		}
	}
	return nil
}

func minifyFile(path, mediaType string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	outPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".min" + filepath.Ext(path)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, in); err != nil {
		return err
	}

	if err := m.Minify(mediaType, out, &buf); err != nil {
		return fmt.Errorf("failed to minify %s: %w", path, err)
	}

	fmt.Printf("  - Minified %s -> %s\n", path, outPath)
	return nil
}

func updateHTMLReferences(isClean bool) error {
	return filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			originalContent := string(content)
			modifiedContent := originalContent

			for original, minified := range assetReplacements {
				if isClean {
					// 还原: .min.css -> .css
					modifiedContent = strings.ReplaceAll(modifiedContent, minified, original)
				} else {
					// 替换: .css -> .min.css
					modifiedContent = strings.ReplaceAll(modifiedContent, original, minified)
				}
			}

			if modifiedContent != originalContent {
				if isClean {
					fmt.Printf("  - Restoring references in %s\n", path)
				} else {
					fmt.Printf("  - Updating references in %s\n", path)
				}
				if err := os.WriteFile(path, []byte(modifiedContent), info.Mode()); err != nil {
					return fmt.Errorf("failed to write updated content to %s: %w", path, err)
				}
			}
		}
		return nil
	})
}
