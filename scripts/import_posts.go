package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PostBackup corresponds to the structure in internal/models/post.go
type PostBackup struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	IsPrivate   bool      `json:"is_private"`
	PublishedAt time.Time `json:"published_at"`
}

// FrontMatter defines the structure of the YAML front matter in Markdown files.
type FrontMatter struct {
	Title       string      `yaml:"title"`
	PublishDate interface{} `yaml:"publishDate"` // Use interface{} to handle multiple date formats
	Draft       bool        `yaml:"draft"`
}

var frontMatterRegex = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n`)

func main() {
	postsDir := "/Users/macm4/code/cactus/src/content/post"
	outputFile := "backup.json"

	var backups []PostBackup

	err := filepath.Walk(postsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			fmt.Printf("Processing file: %s\n", path)

			contentBytes, err := ioutil.ReadFile(path)
			if err != nil {
				fmt.Printf("Error reading file %s: %v\n", path, err)
				return nil // Continue with next file
			}
			contentStr := string(contentBytes)

			matches := frontMatterRegex.FindStringSubmatch(contentStr)
			if len(matches) < 2 {
				fmt.Printf("Warning: Could not find front matter in %s\n", path)
				return nil // Continue with next file
			}

			frontMatterStr := matches[1]
			postContent := strings.TrimSpace(contentStr[len(matches[0]):])

			var fm FrontMatter
			err = yaml.Unmarshal([]byte(frontMatterStr), &fm)
			if err != nil {
				fmt.Printf("Error unmarshalling YAML from %s: %v\n", path, err)
				return nil // Continue with next file
			}

			var publishedAt time.Time
			// Type assertion to handle different date formats
			switch v := fm.PublishDate.(type) {
			case string:
				publishedAt, err = time.Parse("2006-01-02", v)
				if err != nil {
					fmt.Printf("Error parsing date string from %s: %v\n", path, err)
					publishedAt = time.Now()
				}
			case time.Time:
				publishedAt = v
			default:
				fmt.Printf("Warning: Unsupported date format in %s. Using current time.\n", path)
				publishedAt = time.Now()
			}

			backup := PostBackup{
				Title:       fm.Title,
				Content:     postContent,
				IsPrivate:   fm.Draft,
				PublishedAt: publishedAt,
			}
			backups = append(backups, backup)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %s: %v\n", postsDir, err)
		return
	}

	jsonData, err := json.MarshalIndent(backups, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling to JSON: %v\n", err)
		return
	}

	err = ioutil.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to file %s: %v\n", outputFile, err)
		return
	}

	fmt.Printf("Successfully created backup file: %s with %d posts.\n", outputFile, len(backups))
}
