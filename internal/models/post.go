package models

import (
	"html/template"
	"time"

	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Title       string    `gorm:"not null" json:"title" form:"title"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Content     string    `gorm:"type:text;not null" json:"content" form:"content"`
	Excerpt     string    `json:"excerpt"`
	Published   bool      `gorm:"default:false" json:"published" form:"published"`
	PublishedAt time.Time `json:"published_at"`
}

// RenderedPost is a view model for displaying a post with rendered HTML content.
type RenderedPost struct {
	gorm.Model
	Title       string
	Slug        string
	Content     template.HTML // Use template.HTML to prevent escaping
	Excerpt     string
	Published   bool
	PublishedAt time.Time
}
