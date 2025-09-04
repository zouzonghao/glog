package models

import (
	"html/template"
	"time"
)

type Post struct {
	ID          uint `gorm:"primarykey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt time.Time `gorm:"index"`
	Title       string    `gorm:"not null" json:"title" form:"title"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Content     string    `gorm:"type:text;not null" json:"content" form:"content"`
	ContentHTML string    `gorm:"type:text" json:"content_html"`
	Excerpt     string    `json:"excerpt"`
	IsPrivate   bool      `gorm:"index:idx_pub;default:false" json:"is_private" form:"is_private"`
}

// RenderedPost is a view model for displaying a post with rendered HTML content.
type RenderedPost struct {
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt time.Time
	Title       string
	Slug        string
	Summary     template.HTML // Rendered HTML of the content before <!--more-->
	Body        template.HTML // Rendered HTML of the content after <!--more-->
	Excerpt     string        // Plain text excerpt for lists
	IsPrivate   bool
}

// PostBackup is a simplified struct for backup and restore operations.
type PostBackup struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	IsPrivate   bool      `json:"is_private"`
	PublishedAt time.Time `json:"published_at"`
}

// SiteBackup defines the structure for a full site backup, including posts and settings.
type SiteBackup struct {
	Posts    []PostBackup      `json:"posts"`
	Settings map[string]string `json:"settings"`
}
