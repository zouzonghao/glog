package models

import (
	"html/template"
	"time"
)

// PostListResponse is the standardized API response for a post in a list.
// It uses camelCase JSON tags and contains only summary fields.
type PostListResponse struct {
	ID          uint      `json:"id"`
	PublishedAt time.Time `json:"publishedAt"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Cover       string    `json:"cover"`
	Excerpt     string    `json:"excerpt,omitempty"`
	IsPrivate   bool      `json:"isPrivate"`
}

// PostDetailResponse is the standardized API response for a single post.
// It uses camelCase JSON tags and includes the full content.
type PostDetailResponse struct {
	ID          uint          `json:"id"`
	PublishedAt time.Time     `json:"publishedAt"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug"`
	Cover       string        `json:"cover"`
	Content     string        `json:"content,omitempty"` // For editor
	Body        template.HTML `json:"body,omitempty"`    // For rendered view
	IsPrivate   bool          `json:"isPrivate"`
}
