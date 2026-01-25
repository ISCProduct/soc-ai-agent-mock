package repositories

import (
	"Backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type CrawlRepository struct {
	db *gorm.DB
}

func NewCrawlRepository(db *gorm.DB) *CrawlRepository {
	return &CrawlRepository{db: db}
}

func (r *CrawlRepository) ListSources() ([]models.CrawlSource, error) {
	var sources []models.CrawlSource
	err := r.db.Order("created_at desc").Find(&sources).Error
	return sources, err
}

func (r *CrawlRepository) GetSource(id uint) (*models.CrawlSource, error) {
	var source models.CrawlSource
	if err := r.db.First(&source, id).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *CrawlRepository) CreateSource(source *models.CrawlSource) error {
	return r.db.Create(source).Error
}

func (r *CrawlRepository) UpdateSource(source *models.CrawlSource) error {
	return r.db.Save(source).Error
}

func (r *CrawlRepository) ListDueSources(now time.Time) ([]models.CrawlSource, error) {
	var sources []models.CrawlSource
	err := r.db.
		Where("is_active = ?", true).
		Where("next_run_at IS NOT NULL AND next_run_at <= ?", now).
		Order("next_run_at asc").
		Find(&sources).Error
	return sources, err
}

func (r *CrawlRepository) CreateRun(run *models.CrawlRun) error {
	return r.db.Create(run).Error
}

func (r *CrawlRepository) UpdateRun(run *models.CrawlRun) error {
	return r.db.Save(run).Error
}

func (r *CrawlRepository) ListRuns(sourceID uint, limit int) ([]models.CrawlRun, error) {
	var runs []models.CrawlRun
	query := r.db.Order("started_at desc")
	if sourceID > 0 {
		query = query.Where("source_id = ?", sourceID)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&runs).Error
	return runs, err
}
