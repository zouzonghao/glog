package models

import (
	"html/template"
	"time"
)

// PostListResponse is the standardized API response for a post in a list.
// It uses snake_case JSON tags and contains only summary fields.
type PostListResponse struct {
	ID          uint      `json:"id"`
	PublishedAt time.Time `json:"published_at"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Cover       string    `json:"cover"`
	Excerpt     string    `json:"excerpt,omitempty"`
	IsPrivate   bool      `json:"is_private"`
}

// PostDetailResponse is the standardized API response for a single post.
// It uses snake_case JSON tags and includes the full content.
type PostDetailResponse struct {
	ID          uint          `json:"id"`
	PublishedAt time.Time     `json:"published_at"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug"`
	Cover       string        `json:"cover"`
	Content     string        `json:"content,omitempty"` // For editor
	Body        template.HTML `json:"body,omitempty"`    // For rendered view
	IsPrivate   bool          `json:"is_private"`
}

// Pagination defines the structure for pagination information in API responses.
type Pagination struct {
	CurrentPage  int  `json:"current_page"`
	TotalPages   int  `json:"total_pages"`
	TotalRecords int  `json:"total_records"`
	PageSize     int  `json:"page_size"`
	HasPrev      bool `json:"has_prev"`
	HasNext      bool `json:"has_next"`
}
