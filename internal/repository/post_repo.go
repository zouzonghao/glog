package repository

import (
	"glog/internal/models"
	"glog/internal/utils/segmenter"

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

func (r *PostRepository) UpdateFtsIndex(id uint, title, content string) error {
	query := `INSERT OR REPLACE INTO posts_fts (rowid, title, content) VALUES (?, ?, ?)`
	return r.db.Exec(query, id, title, content).Error
}

func (r *PostRepository) DeleteFtsIndex(id uint) error {
	query := `DELETE FROM posts_fts WHERE rowid = ?`
	return r.db.Exec(query, id).Error
}

func (r *PostRepository) RebuildFtsIndex() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Clear the existing FTS index
		if err := tx.Exec("DELETE FROM posts_fts").Error; err != nil {
			return err
		}

		// 2. Fetch all posts
		var posts []models.Post
		if err := tx.Select("id, title, content").Find(&posts).Error; err != nil {
			return err
		}

		// 3. Re-populate the FTS index
		for _, post := range posts {
			segmentedTitle := segmenter.SegmentTextForIndex(post.Title)
			segmentedContent := segmenter.SegmentTextForIndex(post.Content)
			query := `INSERT OR REPLACE INTO posts_fts (rowid, title, content) VALUES (?, ?, ?)`
			if err := tx.Exec(query, post.ID, segmentedTitle, segmentedContent).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostRepository) CreateBatchFromBackup(posts []models.Post) error {
	return r.db.Create(&posts).Error
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

func (r *PostRepository) SearchPage(ftsQuery string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Table("posts").
		Select("posts.*, posts_fts.rank").
		Joins("JOIN posts_fts ON posts.id = posts_fts.rowid").
		Where("posts_fts MATCH ?", ftsQuery)

	if !isLoggedIn {
		dbQuery = dbQuery.Where("posts.is_private = ?", false)
	}

	offset := (page - 1) * pageSize
	err := dbQuery.Order("posts_fts.rank").Offset(offset).Limit(pageSize).Find(&posts).Error
	return posts, err
}

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

func (r *PostRepository) FindAllForBackup() ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Select("title, content, is_private, published_at").
		Order("id asc").
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Post{}).Where("id = ?", id).Updates(fields).Error
}

func (r *PostRepository) DeleteByIDs(ids []uint) error {
	return r.db.Where("id IN ?", ids).Delete(&models.Post{}).Error
}

func (r *PostRepository) UpdatePrivacyByIDs(ids []uint, isPrivate bool) error {
	return r.db.Model(&models.Post{}).Where("id IN ?", ids).Update("is_private", isPrivate).Error
}

func (r *PostRepository) GetDB() *gorm.DB {
	return r.db
}
