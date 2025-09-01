package repository

import (
	"glog/internal/models"
	"time"

	"gorm.io/gorm"
)

var (
	shanghaiLocation, _ = time.LoadLocation("Asia/Shanghai")
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

func (r *PostRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Post{}).Where("id = ?", id).Updates(fields).Error
}

func (r *PostRepository) Delete(id uint) error {
	return r.db.Delete(&models.Post{}, id).Error
}

func (r *PostRepository) FindByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.First(&post, id).Error
	return &post, err
}

func (r *PostRepository) FindBySlug(slug string, isLoggedIn bool) (*models.Post, error) {
	var post models.Post
	query := r.db
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", time.Now().In(shanghaiLocation))
	}
	err := query.Where("slug = ?", slug).First(&post).Error
	return &post, err
}

func (r *PostRepository) FindPage(page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db.Order("published_at desc")
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", time.Now().In(shanghaiLocation))
	}
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) Count(isLoggedIn bool) (int64, error) {
	var count int64
	query := r.db.Model(&models.Post{})
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", time.Now().In(shanghaiLocation))
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *PostRepository) FindAllByAdmin(page, pageSize int, query, status string) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Order("published_at desc")

	if query != "" {
		dbQuery = dbQuery.Where("title LIKE ?", "%"+query+"%")
	}

	now := time.Now().In(shanghaiLocation)
	switch status {
	case "published":
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, now)
	case "draft":
		dbQuery = dbQuery.Where("published_at > ?", now)
	case "private":
		dbQuery = dbQuery.Where("is_private = ?", true)
	}

	err := dbQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountAllByAdmin(query, status string) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{})

	if query != "" {
		dbQuery = dbQuery.Where("title LIKE ?", "%"+query+"%")
	}

	now := time.Now().In(shanghaiLocation)
	switch status {
	case "published":
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, now)
	case "draft":
		dbQuery = dbQuery.Where("published_at > ?", now)
	case "private":
		dbQuery = dbQuery.Where("is_private = ?", true)
	}

	err := dbQuery.Count(&count).Error
	return count, err
}

func (r *PostRepository) CheckSlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

func (r *PostRepository) CheckSlugExistsForOtherPost(slug string, postID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Where("slug = ? AND id != ?", slug, postID).Count(&count).Error
	return count > 0, err
}

func (r *PostRepository) FindAllForBackup() ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CreateBatchFromBackup(posts []models.Post) error {
	return r.db.Create(&posts).Error
}

func (r *PostRepository) DeleteByIDs(ids []uint) error {
	return r.db.Delete(&models.Post{}, ids).Error
}

func (r *PostRepository) UpdatePrivacyByIDs(ids []uint, isPrivate bool) error {
	return r.db.Model(&models.Post{}).Where("id IN ?", ids).Update("is_private", isPrivate).Error
}

// --- LIKE Search Methods ---

func (r *PostRepository) SearchPageByLike(keywords []string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Order("published_at desc")

	for _, keyword := range keywords {
		likeQuery := "%" + keyword + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, time.Now().In(shanghaiLocation))
	}

	err := dbQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountByQueryByLike(keywords []string, isLoggedIn bool) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{})

	for _, keyword := range keywords {
		likeQuery := "%" + keyword + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, time.Now().In(shanghaiLocation))
	}

	err := dbQuery.Count(&count).Error
	return count, err
}
