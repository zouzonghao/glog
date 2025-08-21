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
	return r.db.Delete(&models.Post{}, id).Error
}

func (r *PostRepository) FindByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.First(&post, id).Error
	return &post, err
}

func (r *PostRepository) FindBySlug(slug string) (*models.Post, error) {
	var post models.Post
	err := r.db.Where("slug = ? AND published = ?", slug, true).First(&post).Error
	return &post, err
}

func (r *PostRepository) FindAllPublished() ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Where("published = ?", true).Order("published_at desc").Find(&posts).Error
	return posts, err
}

func (r *PostRepository) FindAll() ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Order("created_at desc").Find(&posts).Error
	return posts, err
}

func (r *PostRepository) Search(query string) ([]models.Post, error) {
	var posts []models.Post
	searchQuery := "%" + query + "%"
	err := r.db.Where("published = ? AND (title LIKE ? OR content LIKE ?)",
		true, searchQuery, searchQuery).Order("published_at desc").Find(&posts).Error
	return posts, err
}
