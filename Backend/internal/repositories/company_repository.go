package repositories

import (
	"Backend/internal/models"

	"gorm.io/gorm"
)

type CompanyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// FindAllActive アクティブな企業を全て取得
func (r *CompanyRepository) FindAllActive() ([]models.Company, error) {
	var companies []models.Company
	err := r.db.Where("is_active = ?", true).Find(&companies).Error
	return companies, err
}

// FindByID IDで企業を取得
func (r *CompanyRepository) FindByID(id uint) (*models.Company, error) {
	var company models.Company
	err := r.db.First(&company, id).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

// GetWeightProfile 企業の重視度プロファイルを取得
func (r *CompanyRepository) GetWeightProfile(companyID uint, jobPositionID *uint) (*models.CompanyWeightProfile, error) {
	var profile models.CompanyWeightProfile
	query := r.db.Where("company_id = ?", companyID)

	if jobPositionID != nil {
		query = query.Where("job_position_id = ?", *jobPositionID)
	} else {
		query = query.Where("job_position_id IS NULL")
	}

	err := query.First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Create 企業を作成
func (r *CompanyRepository) Create(company *models.Company) error {
	return r.db.Create(company).Error
}

// Update 企業情報を更新
func (r *CompanyRepository) Update(company *models.Company) error {
	return r.db.Save(company).Error
}

// CreateJobPosition 募集職種を作成
func (r *CompanyRepository) CreateJobPosition(position *models.CompanyJobPosition) error {
	return r.db.Create(position).Error
}

// FindJobPositionsByCompany 企業の募集職種を取得
func (r *CompanyRepository) FindJobPositionsByCompany(companyID uint) ([]models.CompanyJobPosition, error) {
	var positions []models.CompanyJobPosition
	err := r.db.Where("company_id = ? AND is_active = ?", companyID, true).
		Preload("JobCategory").
		Find(&positions).Error
	return positions, err
}

// CreateOrUpdateWeightProfile 重視度プロファイルを作成または更新
func (r *CompanyRepository) CreateOrUpdateWeightProfile(profile *models.CompanyWeightProfile) error {
	var existing models.CompanyWeightProfile
	query := r.db.Where("company_id = ?", profile.CompanyID)

	if profile.JobPositionID != nil {
		query = query.Where("job_position_id = ?", *profile.JobPositionID)
	} else {
		query = query.Where("job_position_id IS NULL")
	}

	err := query.First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		// 新規作成
		return r.db.Create(profile).Error
	} else if err != nil {
		return err
	}

	// 更新
	profile.ID = existing.ID
	return r.db.Save(profile).Error
}
