package mapper

import (
	"Backend/domain/entity"
	"Backend/internal/models"
)

// CompanyToEntity GORMモデルをドメインエンティティに変換する
func CompanyToEntity(m *models.Company) *entity.Company {
	if m == nil {
		return nil
	}
	return &entity.Company{
		ID:               m.ID,
		Name:             m.Name,
		Description:      m.Description,
		Industry:         m.Industry,
		EmployeeCount:    m.EmployeeCount,
		FoundedYear:      m.FoundedYear,
		Location:         m.Location,
		WebsiteURL:       m.WebsiteURL,
		LogoURL:          m.LogoURL,
		CorporateNumber:  m.CorporateNumber,
		SourceType:       m.SourceType,
		SourceURL:        m.SourceURL,
		SourceFetchedAt:  m.SourceFetchedAt,
		IsProvisional:    m.IsProvisional,
		DataStatus:       m.DataStatus,
		Culture:          m.Culture,
		WorkStyle:        m.WorkStyle,
		WelfareDetails:   m.WelfareDetails,
		TechStack:        m.TechStack,
		DevelopmentStyle: m.DevelopmentStyle,
		MainBusiness:     m.MainBusiness,
		AverageAge:       m.AverageAge,
		FemaleRatio:      m.FemaleRatio,
		IsActive:         m.IsActive,
		IsVerified:       m.IsVerified,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// CompanyWeightProfileToEntity GORMモデルをドメインエンティティに変換する
func CompanyWeightProfileToEntity(m *models.CompanyWeightProfile) *entity.CompanyWeightProfile {
	if m == nil {
		return nil
	}
	return &entity.CompanyWeightProfile{
		ID:                    m.ID,
		CompanyID:             m.CompanyID,
		JobPositionID:         m.JobPositionID,
		TechnicalOrientation:  m.TechnicalOrientation,
		TeamworkOrientation:   m.TeamworkOrientation,
		LeadershipOrientation: m.LeadershipOrientation,
		CreativityOrientation: m.CreativityOrientation,
		StabilityOrientation:  m.StabilityOrientation,
		GrowthOrientation:     m.GrowthOrientation,
		WorkLifeBalance:       m.WorkLifeBalance,
		ChallengeSeeking:      m.ChallengeSeeking,
		DetailOrientation:     m.DetailOrientation,
		CommunicationSkill:    m.CommunicationSkill,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

// UserCompanyMatchToEntity GORMモデルをドメインエンティティに変換する
func UserCompanyMatchToEntity(m *models.UserCompanyMatch) *entity.UserCompanyMatch {
	if m == nil {
		return nil
	}
	e := &entity.UserCompanyMatch{
		ID:                 m.ID,
		UserID:             m.UserID,
		SessionID:          m.SessionID,
		CompanyID:          m.CompanyID,
		MatchScore:         m.MatchScore,
		TechnicalMatch:     m.TechnicalMatch,
		TeamworkMatch:      m.TeamworkMatch,
		LeadershipMatch:    m.LeadershipMatch,
		CreativityMatch:    m.CreativityMatch,
		StabilityMatch:     m.StabilityMatch,
		GrowthMatch:        m.GrowthMatch,
		WorkLifeMatch:      m.WorkLifeMatch,
		ChallengeMatch:     m.ChallengeMatch,
		DetailMatch:        m.DetailMatch,
		CommunicationMatch: m.CommunicationMatch,
		MatchReason:        m.MatchReason,
		IsViewed:           m.IsViewed,
		IsFavorited:        m.IsFavorited,
		IsApplied:          m.IsApplied,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
	if m.Company.ID != 0 {
		e.Company = CompanyToEntity(&m.Company)
	}
	return e
}
