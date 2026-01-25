package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"os"
	"strings"
)

func promoteAdminIfMatched(user *models.User, repo *repositories.UserRepository) {
	if user == nil || repo == nil || user.IsAdmin {
		return
	}
	if !isAdminIdentity(user.Email, user.Name) {
		return
	}
	user.IsAdmin = true
	_ = repo.UpdateUser(user)
}

func isAdminIdentity(email string, name string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.ToLower(strings.TrimSpace(name))
	localPart := email
	if at := strings.Index(localPart, "@"); at >= 0 {
		localPart = localPart[:at]
	}
	adminEmails := splitEnvList("ADMIN_EMAILS")
	adminUsers := splitEnvList("ADMIN_USERNAMES")
	for _, v := range adminEmails {
		if v == email {
			return true
		}
	}
	for _, v := range adminUsers {
		if v == name || v == localPart {
			return true
		}
	}
	return false
}

func splitEnvList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		val := strings.ToLower(strings.TrimSpace(p))
		if val == "" {
			continue
		}
		result = append(result, val)
	}
	return result
}
