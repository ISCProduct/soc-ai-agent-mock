package repositories

import (
	"Backend/internal/models"
	"errors"

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

func (r *JobCategoryEmbeddingRepository) Upsert(jobCategoryID uint, sourceText, embedding string) error {
	var existing models.JobCategoryEmbedding
	err := r.db.Where("job_category_id = ?", jobCategoryID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record := models.JobCategoryEmbedding{
				JobCategoryID: jobCategoryID,
				SourceText:    sourceText,
				Embedding:     embedding,
			}
			return r.db.Create(&record).Error
		}
		return err
	}

	existing.SourceText = sourceText
	existing.Embedding = embedding
	return r.db.Save(&existing).Error
}
