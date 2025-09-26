package models

import (
	"html/template"
	"time"
)

type Post struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PublishedAt time.Time `gorm:"index" json:"published_at"`
	Title       string    `gorm:"not null" json:"title" form:"title"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Cover       string    `json:"cover" form:"cover"` // 新增封面图字段
	Content     string    `gorm:"type:text;not null" json:"content" form:"content"`
	ContentHTML string    `gorm:"type:text" json:"content_html"`
	Excerpt     string    `json:"excerpt"`
	IsPrivate   bool      `gorm:"index:idx_pub;default:false" json:"is_private" form:"is_private"`
}

// RenderedPost is a view model for displaying a post with rendered HTML content.
type RenderedPost struct {
	ID          uint          `json:"id"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	PublishedAt time.Time     `json:"published_at"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug"`
	Cover       string        `json:"cover"`
	Summary     template.HTML `json:"summary,omitempty"`
	Body        template.HTML `json:"body,omitempty"`
	Excerpt     string        `json:"excerpt"`
	IsPrivate   bool          `json:"is_private"`
}

// PostBackup is a simplified struct for backup and restore operations.
type PostBackup struct {
	Title       string    `json:"title"`
	Cover       string    `json:"cover"` // 备份时也包含封面
	Content     string    `json:"content"`
	IsPrivate   bool      `json:"is_private"`
	PublishedAt time.Time `json:"published_at"`
}

// SiteBackup defines the structure for a full site backup, including posts and settings.
type SiteBackup struct {
	Posts    []PostBackup      `json:"posts"`
	Settings map[string]string `json:"settings"`
}
