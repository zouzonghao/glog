package repository

import (
	"glog/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

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
		query = query.Where("is_private = ?", false).Where("published_at <= ?", nowUTC())
	}
	err := query.Where("slug = ?", slug).First(&post).Error
	return &post, err
}

func (r *PostRepository) FindBySlugIgnorePrivacy(slug string) (*models.Post, error) {
	var post models.Post
	err := r.db.Where("slug = ?", slug).First(&post).Error
	return &post, err
}

func (r *PostRepository) FindPage(page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	query := r.db.Order("published_at desc")
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", nowUTC())
	}
	err := query.Select("id", "created_at", "updated_at", "published_at", "title", "slug", "tag", "cover", "excerpt", "is_private").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) Count(isLoggedIn bool) (int64, error) {
	var count int64
	query := r.db.Model(&models.Post{})
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", nowUTC())
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

	now := nowUTC()
	switch status {
	case "published":
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, now)
	case "draft":
		dbQuery = dbQuery.Where("published_at > ?", now)
	case "private":
		dbQuery = dbQuery.Where("is_private = ?", true)
	}

	err := dbQuery.Select("id", "published_at", "title", "slug", "is_private", "updated_at").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountAllByAdmin(query, status string) (int64, error) {
	var count int64
	dbQuery := r.db.Model(&models.Post{})

	if query != "" {
		dbQuery = dbQuery.Where("title LIKE ?", "%"+query+"%")
	}

	now := nowUTC()
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

func (r *PostRepository) ReplaceAllPosts(posts []models.Post) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Post{}).Error; err != nil {
			return err
		}
		if len(posts) > 0 {
			return tx.Create(&posts).Error
		}
		return nil
	})
}

func (r *PostRepository) UpsertAllPosts(posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"title", "tag", "content", "content_html", "is_private", "published_at", "excerpt", "cover"}),
		}).Create(&posts).Error
	})
}

func (r *PostRepository) UpdatePrivacyByIDs(ids []uint, isPrivate bool) error {
	return r.db.Model(&models.Post{}).Where("id IN ?", ids).Update("is_private", isPrivate).Error
}

func (r *PostRepository) SearchPageByLike(keywords []string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	dbQuery := r.db.Order("published_at desc")

	for _, keyword := range keywords {
		likeQuery := "%" + keyword + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, nowUTC())
	}

	err := dbQuery.Select("id", "created_at", "updated_at", "published_at", "title", "slug", "cover", "excerpt", "is_private").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
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
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, nowUTC())
	}

	err := dbQuery.Count(&count).Error
	return count, err
}

func (r *PostRepository) FindPageByTag(tag string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	safeTag := escapeLikePattern(strings.ToLower(tag))
	query := r.db.Where("',' || tag || ',' LIKE ? ESCAPE '\\'", "%,"+safeTag+",%").Order("published_at desc")
	if !isLoggedIn {
		query = query.Where("is_private = ?", false).Where("published_at <= ?", nowUTC())
	}
	err := query.Select("id", "created_at", "updated_at", "published_at", "title", "slug", "tag", "cover", "excerpt", "is_private").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountByTag(tag string, isLoggedIn bool) (int64, error) {
	var count int64
	safeTag := escapeLikePattern(strings.ToLower(tag))
	query := r.db.Model(&models.Post{}).Where("',' || tag || ',' LIKE ? ESCAPE '\\'", "%,"+safeTag+",%")
	if !isLoggedIn {
		query = query.Where("is_private = ? AND published_at <= ?", false, nowUTC())
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *PostRepository) GetAllTags(isLoggedIn bool) ([]string, error) {
	var tagStrings []string
	query := r.db.Model(&models.Post{}).Distinct("tag").Where("tag != ?", "")
	if !isLoggedIn {
		query = query.Where("is_private = ? AND published_at <= ?", false, nowUTC())
	}
	err := query.Pluck("tag", &tagStrings).Error
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]bool)
	for _, ts := range tagStrings {
		for _, tag := range strings.Split(ts, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagSet[tag] = true
			}
		}
	}

	result := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		result = append(result, tag)
	}
	return result, nil
}

func (r *PostRepository) SearchPageByLikeAndTag(keywords []string, tag string, page, pageSize int, isLoggedIn bool) ([]models.Post, error) {
	var posts []models.Post
	safeTag := escapeLikePattern(strings.ToLower(tag))
	dbQuery := r.db.Where("',' || tag || ',' LIKE ? ESCAPE '\\'", "%,"+safeTag+",%").Order("published_at desc")

	for _, keyword := range keywords {
		likeQuery := "%" + keyword + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, nowUTC())
	}

	err := dbQuery.Select("id", "created_at", "updated_at", "published_at", "title", "slug", "tag", "cover", "excerpt", "is_private").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, err
}

func (r *PostRepository) CountByQueryByLikeAndTag(keywords []string, tag string, isLoggedIn bool) (int64, error) {
	var count int64
	safeTag := escapeLikePattern(strings.ToLower(tag))
	dbQuery := r.db.Model(&models.Post{}).Where("',' || tag || ',' LIKE ? ESCAPE '\\'", "%,"+safeTag+",%")

	for _, keyword := range keywords {
		likeQuery := "%" + keyword + "%"
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?", likeQuery, likeQuery)
	}

	if !isLoggedIn {
		dbQuery = dbQuery.Where("is_private = ? AND published_at <= ?", false, nowUTC())
	}

	err := dbQuery.Count(&count).Error
	return count, err
}
