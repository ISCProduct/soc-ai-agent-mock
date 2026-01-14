package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type JobCategoryEmbeddingRepository struct {
	db *gorm.DB
}

func NewJobCategoryEmbeddingRepository(db *gorm.DB) *JobCategoryEmbeddingRepository {
	return &JobCategoryEmbeddingRepository{db: db}
}

func (r *JobCategoryEmbeddingRepository) FindByJobCategoryID(jobCategoryID uint) (*models.JobCategoryEmbedding, error) {
	var embedding models.JobCategoryEmbedding
	err := r.db.Where("job_category_id = ?", jobCategoryID).First(&embedding).Error
	if err != nil {
		return nil, err
	}
	return &embedding, nil
}
