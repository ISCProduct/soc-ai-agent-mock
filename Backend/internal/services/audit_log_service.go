package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"encoding/json"
)

type AuditLogService struct {
	repo *repositories.AuditLogRepository
}

func NewAuditLogService(repo *repositories.AuditLogRepository) *AuditLogService {
	return &AuditLogService{repo: repo}
}

func (s *AuditLogService) Record(actorEmail, action, targetType string, targetID uint, metadata map[string]interface{}) {
	if s == nil || s.repo == nil {
		return
	}
	meta := ""
	if metadata != nil {
		if raw, err := json.Marshal(metadata); err == nil {
			meta = string(raw)
		}
	}
	_ = s.repo.Create(&models.AuditLog{
		ActorEmail: actorEmail,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Metadata:   meta,
	})
}

func (s *AuditLogService) List(limit int) ([]models.AuditLog, error) {
	return s.repo.List(limit)
}
