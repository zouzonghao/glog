package services

import (
	"errors"
	"fmt"
	"glog/internal/models"
	"glog/internal/repository"
	"glog/internal/utils"
	"time"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

type PostService struct {
	repo *repository.PostRepository
}

func NewPostService(repo *repository.PostRepository) *PostService {
	return &PostService{repo: repo}
}

func (s *PostService) CreatePost(title, content string, published bool) (*models.Post, error) {
	baseSlug := slug.Make(title)
	finalSlug := baseSlug
	counter := 1
	for {
		_, err := s.repo.FindBySlug(finalSlug)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			break
		}
		if err != nil {
			return nil, err
		}
		finalSlug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	excerpt := utils.GenerateExcerpt(content, 150)

	post := &models.Post{
		Title:     title,
		Slug:      finalSlug,
		Content:   content,
		Excerpt:   excerpt,
		Published: published,
	}

	if published {
		post.PublishedAt = time.Now()
	}

	err := s.repo.Create(post)
	return post, err
}

func (s *PostService) UpdatePost(id uint, title, content string, published bool) (*models.Post, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	post.Title = title
	post.Content = content
	post.Excerpt = utils.GenerateExcerpt(content, 150)

	if !post.Published && published {
		post.PublishedAt = time.Now()
	}
	post.Published = published

	err = s.repo.Update(post)
	return post, err
}

func (s *PostService) GetPostByID(id uint) (*models.Post, error) {
	return s.repo.FindByID(id)
}

func (s *PostService) GetRenderedPostBySlug(slug string) (*models.RenderedPost, error) {
	post, err := s.repo.FindBySlug(slug)
	if err != nil {
		return nil, err
	}

	renderedContent, err := utils.RenderMarkdown(post.Content)
	if err != nil {
		return nil, err
	}

	renderedPost := &models.RenderedPost{
		Model:       post.Model,
		Title:       post.Title,
		Slug:        post.Slug,
		Content:     renderedContent, // Keep as template.HTML
		Excerpt:     post.Excerpt,
		Published:   post.Published,
		PublishedAt: post.PublishedAt,
	}

	return renderedPost, nil
}

func (s *PostService) GetAllPublishedPosts() ([]models.Post, error) {
	return s.repo.FindAllPublished()
}

func (s *PostService) GetAllPosts() ([]models.Post, error) {
	return s.repo.FindAll()
}

func (s *PostService) SearchPosts(query string) ([]models.Post, error) {
	return s.repo.Search(query)
}

func (s *PostService) DeletePost(id uint) error {
	return s.repo.Delete(id)
}
