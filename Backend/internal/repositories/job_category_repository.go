package repositories

import (
	"Backend/internal/models"
	"gorm.io/gorm"
)

type JobCategoryRepository struct {
	db *gorm.DB
}

func NewJobCategoryRepository(db *gorm.DB) *JobCategoryRepository {
	return &JobCategoryRepository{db: db}
}

// FindAll すべての職種を取得
func (r *JobCategoryRepository) FindAll() ([]models.JobCategory, error) {
	var categories []models.JobCategory
	err := r.db.Where("is_active = ?", true).Order("display_order ASC").Find(&categories).Error
	return categories, err
}

// FindByID IDで職種を取得
func (r *JobCategoryRepository) FindByID(id uint) (*models.JobCategory, error) {
	var category models.JobCategory
	err := r.db.First(&category, id).Error
	return &category, err
}

// FindByName 名前で職種を検索（部分一致）
func (r *JobCategoryRepository) FindByName(name string) ([]models.JobCategory, error) {
	var categories []models.JobCategory
	err := r.db.Where("is_active = ? AND name LIKE ?", true, "%"+name+"%").
		Order("display_order ASC").
		Find(&categories).Error
	return categories, err
}

// FindByIndustry 業界に関連する職種を取得
func (r *JobCategoryRepository) FindByIndustry(industryID uint) ([]models.JobCategory, error) {
	var categories []models.JobCategory
	err := r.db.Raw(`
		SELECT jc.* FROM job_categories jc
		INNER JOIN industry_job_categories ijc ON jc.id = ijc.job_category_id
		WHERE ijc.industry_id = ? AND jc.is_active = true
		ORDER BY ijc.display_order ASC
	`, industryID).Scan(&categories).Error
	return categories, err
}

// GetTopCategories 主要な職種カテゴリを取得（Level 0）
func (r *JobCategoryRepository) GetTopCategories() ([]models.JobCategory, error) {
	var categories []models.JobCategory
	err := r.db.Where("is_active = ? AND level = ?", true, 0).
		Order("display_order ASC").
		Find(&categories).Error
	return categories, err
}
