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

// UpdateFtsIndex 更新文章的 FTS 索引
func (r *PostRepository) UpdateFtsIndex(id uint, title, content string) error {
	// 使用 INSERT OR REPLACE 来插入或更新索引
	query := `INSERT OR REPLACE INTO posts_fts (rowid, title, content) VALUES (?, ?, ?)`
	return r.db.Exec(query, id, title, content).Error
}

// DeleteFtsIndex 删除文章的 FTS 索引
func (r *PostRepository) DeleteFtsIndex(id uint) error {
	query := `DELETE FROM posts_fts WHERE rowid = ?`
	return r.db.Exec(query, id).Error
}

func (r *PostRepository) FindByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.First(&post, id).Error
	return &post, err
}

func (r *PostRepository) FindBySlug(slug string, isLoggedIn bool) (*models.Post, error) {
	var post models.Post
	query := r.db.Where("slug = ?", slug)
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	err := query.First(&post).Error
	return &post, err
}

func (r *PostRepository) CheckSlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) CheckSlugExistsForOtherPost(slug string, id uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Post{}).Where("slug = ? AND id != ?", slug, id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) FindAll(isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	err := query.Order("published_at desc").Find(&posts).Error
	return posts, err
}

func (r *PostRepository) FindPage(page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	offset := (page - 1) * pageSize
	err := query.Order("published_at desc").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) Count(isLoggedIn bool) (int64, error) {
	var count int64
	query := r.db.Model(&models.Post{})
	if !isLoggedIn {
		query = query.Where("is_private = ?", false)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *PostRepository) FindAllByAdmin(page, pageSize int, query, status string) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Order("published_at desc")

	if query != "" {
		subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", query)
		dbQuery = dbQuery.Where("id IN (?)", subQuery)
	}

	offset := (page - 1) * pageSize
	err := dbQuery.Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountAllByAdmin(query, status string) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{})

	if query != "" {
		subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", query)
		dbQuery = dbQuery.Where("id IN (?)", subQuery)
	}

	err := dbQuery.Count(&count).Error
	return count, err
}

// SearchPage searches posts with pagination using FTS, ordering by relevance (rank).
func (r *PostRepository) SearchPage(ftsQuery string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post

	// Use JOIN for potentially better performance and to access the rank column.
	// Order by FTS5's rank to ensure the most relevant results appear first.
	dbQuery := r.db.Table("posts").
		Select("posts.*, posts_fts.rank").
		Joins("JOIN posts_fts ON posts.id = posts_fts.rowid").
		Where("posts_fts MATCH ?", ftsQuery)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("posts.is_private = ?", false)
	}

	offset := (page - 1) * pageSize
	// Ordering by rank is crucial for a good search experience.
	// FTS5's rank is a negative number, so ascending order is correct.
	err := dbQuery.Order("posts_fts.rank").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

// CountByQuery counts the total number of posts matching an FTS query, respecting login status.
func (r *PostRepository) CountByQuery(ftsQuery string, isLoggedIn bool) (int64, error) {
	var count int64

	subQuery := r.db.Table("posts_fts").Select("rowid").Where("posts_fts MATCH ?", ftsQuery)

	dbQuery := r.db.Model(&models.Post{}).Where("id IN (?)", subQuery)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ?", false)
	}

	err := dbQuery.Count(&count).Error
	return count, err
}
