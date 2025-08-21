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
	IsPrivate   bool      `gorm:"default:false" json:"is_private" form:"is_private"`
	PublishedAt time.Time `json:"published_at"`
}

// RenderedPost is a view model for displaying a post with rendered HTML content.
type RenderedPost struct {
	gorm.Model
	Title       string
	Slug        string
	Summary     template.HTML // Rendered HTML of the content before <!--more-->
	Body        template.HTML // Rendered HTML of the content after <!--more-->
	Excerpt     string        // Plain text excerpt for lists
	Published   bool
	IsPrivate   bool
	PublishedAt time.Time
}
