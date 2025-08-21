package repository

import (
	"glog/internal/models"

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

func (r *PostRepository) FindAll(page, pageSize int) ([]models.Post, error) {
	var posts []models.Post
	offset := (page - 1) * pageSize
	err := r.db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Count(&count).Error
	return count, err
}

func (r *PostRepository) Search(ftsQuery string, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post

	// Subquery to get matching rowids from FTS table
	subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", ftsQuery)

	// Main query to get posts from the posts table
	dbQuery := r.db.Where("id IN (?)", subQuery).Where("published = ?", true)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ?", false)
	}

	// To maintain relevance order from FTS, we need a more complex query.
	// For simplicity, we'll order by published_at for now.
	err := dbQuery.Order("published_at desc").Find(&posts).Error
	return posts, err
}

// SearchAll searches all posts (published or not) with pagination using FTS.
func (r *PostRepository) SearchAll(ftsQuery string, page, pageSize int) ([]models.Post, error) {
	var posts []models.Post

	subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", ftsQuery)

	offset := (page - 1) * pageSize
	err := r.db.Where("id IN (?)", subQuery).
		Order("created_at desc").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

// CountAllByQuery counts the total number of posts matching an FTS query.
func (r *PostRepository) CountAllByQuery(ftsQuery string) (int64, error) {
	var count int64

	subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", ftsQuery)

	err := r.db.Model(&models.Post{}).Where("id IN (?)", subQuery).Count(&count).Error
	return count, err
}
