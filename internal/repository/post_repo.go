package repository

import (
	"glog/internal/models"
	"glog/internal/utils/segmenter"
	"time"

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
		query = query.Where("is_private = ?", false).Where("published_at <= ?", time.Now())
	}
	err := query.Where("slug = ?", slug).First(&post).Error
	return &post, err
}

func (r *PostRepository) FindPage(page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db.Order("published_at desc")
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", time.Now())
	}
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) Count(isLoggedIn bool) (int64, error) {
	var count int64
	query := r.db.Model(&models.Post{})
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", time.Now())
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

	now := time.Now()
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

	now := time.Now()
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

// --- FTS Specific Methods ---

func (r *PostRepository) SearchPage(query string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Table("posts").
		Joins("JOIN posts_fts ON posts.rowid = posts_fts.rowid").
		Where("posts_fts MATCH ?", query).
		Order("bm25(posts_fts, 2, 1)")

	if !isLoggedIn {
		dbQuery = dbQuery.Where("posts.is_private = ? AND posts.published_at <= ?", false, time.Now())
	}

	err := dbQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountByQuery(query string, isLoggedIn bool) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{}).
		Joins("JOIN posts_fts ON posts.rowid = posts_fts.rowid").
		Where("posts_fts MATCH ?", query)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("posts.is_private = ? AND posts.published_at <= ?", false, time.Now())
	}

	err := dbQuery.Count(&count).Error
	return count, err
}

func (r *PostRepository) UpdateFtsIndex(postID uint, title, content string) error {
	return r.db.Exec("INSERT OR REPLACE INTO posts_fts (rowid, title, content) VALUES (?, ?, ?)", postID, title, content).Error
}

func (r *PostRepository) DeleteFtsIndex(postID uint) error {
	return r.db.Exec("DELETE FROM posts_fts WHERE rowid = ?", postID).Error
}

func (r *PostRepository) RebuildFtsIndex() error {
	var posts []models.Post
	if err := r.db.Find(&posts).Error; err != nil {
		return err
	}

	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Exec("DELETE FROM posts_fts").Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, post := range posts {
		segmentedTitle := segmenter.SegmentTextForIndex(post.Title)
		segmentedContent := segmenter.SegmentTextForIndex(post.Content)
		if err := tx.Exec("INSERT INTO posts_fts (rowid, title, content) VALUES (?, ?, ?)", post.ID, segmentedTitle, segmentedContent).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// --- LIKE Search Methods ---

func (r *PostRepository) SearchPageByLike(query string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	likeQuery := "%" + query + "%"
	dbQuery := r.db.Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery).Order("published_at desc")

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, time.Now())
	}

	err := dbQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountByQueryByLike(query string, isLoggedIn bool) (int64, error) {
	var count int64
	likeQuery := "%" + query + "%"
	dbQuery := r.db.Model(&models.Post{}).Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, time.Now())
	}

	err := dbQuery.Count(&count).Error
	return count, err
}
