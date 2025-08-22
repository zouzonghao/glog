package repository

import (
	"glog/internal/models"
	"strings"

	"gorm.io/gorm"
)

type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *PostRepository) Update(post *models.Post) error {
	return r.db.Save(post).Error
}

func (r *PostRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&models.Post{}, id).Error
}

func (r *PostRepository) FindByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.First(&post, id).Error
	return &post, err
}

func (r *PostRepository) FindBySlug(slug string, isLoggedIn bool) (*models.Post, error) {
	var post models.Post
	query := r.db.Where("slug = ? AND published = ?", slug, true)
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	err := query.First(&post).Error
	return &post, err
}

func (r *PostRepository) CheckSlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Unscoped().Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) CheckSlugExistsForOtherPost(slug string, id uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Unscoped().Where("slug = ? AND id != ?", slug, id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) FindAllPublished(isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db.Where("published = ?", true)
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	err := query.Order("published_at desc").Find(&posts).Error
	return posts, err
}

func (r *PostRepository) FindPublishedPage(page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db.Where("published = ?", true)
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	offset := (page - 1) * pageSize
	err := query.Order("published_at desc").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountPublished(isLoggedIn bool) (int64, error) {
	var count int64
	query := r.db.Model(&models.Post{}).Where("published = ?", true)
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *PostRepository) FindAll(page, pageSize int, query, status string) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Order("created_at desc")

	if query != "" {
		searchQuery := "%" + strings.ToLower(query) + "%"
		dbQuery = dbQuery.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", searchQuery, searchQuery)
	}

	switch status {
	case "published":
		dbQuery = dbQuery.Where("published = ?", true)
	case "draft":
		dbQuery = dbQuery.Where("published = ?", false)
	}

	offset := (page - 1) * pageSize
	err := dbQuery.Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountAll(query, status string) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{})

	if query != "" {
		searchQuery := "%" + strings.ToLower(query) + "%"
		dbQuery = dbQuery.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", searchQuery, searchQuery)
	}

	switch status {
	case "published":
		dbQuery = dbQuery.Where("published = ?", true)
	case "draft":
		dbQuery = dbQuery.Where("published = ?", false)
	}

	err := dbQuery.Count(&count).Error
	return count, err
}

// SearchPage searches published posts with pagination using FTS, respecting login status.
func (r *PostRepository) SearchPage(ftsQuery string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post

	subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", ftsQuery)

	dbQuery := r.db.Where("id IN (?)", subQuery).Where("published = ?", true)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ?", false)
	}

	offset := (page - 1) * pageSize
	err := dbQuery.Order("published_at desc").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

// CountByQuery counts the total number of published posts matching an FTS query, respecting login status.
func (r *PostRepository) CountByQuery(ftsQuery string, isLoggedIn bool) (int64, error) {
	var count int64

	subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", ftsQuery)

	dbQuery := r.db.Model(&models.Post{}).Where("id IN (?)", subQuery).Where("published = ?", true)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ?", false)
	}

	err := dbQuery.Count(&count).Error
	return count, err
}

// SearchPageWithLike searches published posts with pagination using LIKE, respecting login status.
func (r *PostRepository) SearchPageWithLike(query string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Where("published = ?", true)

	keywords := strings.Fields(strings.ReplaceAll(query, ",", " "))
	for _, keyword := range keywords {
		trimmedKeyword := strings.TrimSpace(keyword)
		if trimmedKeyword != "" {
			searchQuery := "%" + trimmedKeyword + "%"
			dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", searchQuery, searchQuery)
		}
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ?", false)
	}

	offset := (page - 1) * pageSize
	err := dbQuery.Order("published_at desc").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

// CountByQueryWithLike counts the total number of published posts matching a LIKE query, respecting login status.
func (r *PostRepository) CountByQueryWithLike(query string, isLoggedIn bool) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{}).Where("published = ?", true)

	keywords := strings.Fields(strings.ReplaceAll(query, ",", " "))
	for _, keyword := range keywords {
		trimmedKeyword := strings.TrimSpace(keyword)
		if trimmedKeyword != "" {
			searchQuery := "%" + trimmedKeyword + "%"
			dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", searchQuery, searchQuery)
		}
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ?", false)
	}

	err := dbQuery.Count(&count).Error
	return count, err
}
