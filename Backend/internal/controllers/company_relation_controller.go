package controllers

import (
	"Backend/internal/models"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type CompanyRelationController struct {
	DB *gorm.DB
}

// GetCompanyRelations 企業IDに関連する企業関係を取得
func (ctrl *CompanyRelationController) GetCompanyRelations(w http.ResponseWriter, r *http.Request) {
	// パスから企業IDを抽出
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	companyIDStr := pathParts[2]
	companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	var relations []models.CompanyRelation

	err = ctrl.DB.
		Preload("Parent").
		Preload("Child").
		Preload("From").
		Preload("To").
		Where("parent_id = ? OR child_id = ? OR from_id = ? OR to_id = ?",
			companyID, companyID, companyID, companyID).
		Where("is_active = ?", true).
		Find(&relations).Error

	if err != nil {
		http.Error(w, "Failed to fetch relations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relations)
}

// GetCompanyMarketInfo 企業の市場情報を取得
func (ctrl *CompanyRelationController) GetCompanyMarketInfo(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	companyIDStr := pathParts[2]
	companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid company ID", http.StatusBadRequest)
		return
	}

	var marketInfo models.CompanyMarketInfo
	err = ctrl.DB.
		Preload("Company").
		Where("company_id = ?", companyID).
		First(&marketInfo).Error

	if err == gorm.ErrRecordNotFound {
		http.Error(w, "Market info not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Failed to fetch market info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marketInfo)
}

// GetAllCompanyRelations 全企業関係を取得（関連図用）
func (ctrl *CompanyRelationController) GetAllCompanyRelations(w http.ResponseWriter, r *http.Request) {
	var relations []models.CompanyRelation

	err := ctrl.DB.
		Preload("Parent").
		Preload("Child").
		Preload("From").
		Preload("To").
		Where("is_active = ?", true).
		Find(&relations).Error

	if err != nil {
		http.Error(w, "Failed to fetch relations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relations)
}

// GetAllMarketInfo 全企業の市場情報を取得
func (ctrl *CompanyRelationController) GetAllMarketInfo(w http.ResponseWriter, r *http.Request) {
	var marketInfos []models.CompanyMarketInfo

	err := ctrl.DB.
		Preload("Company").
		Find(&marketInfos).Error

	if err != nil {
		http.Error(w, "Failed to fetch market info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marketInfos)
}

// GetCompanies 企業一覧を取得
func (ctrl *CompanyRelationController) GetCompanies(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	industry := r.URL.Query().Get("industry")

	limit := 10 // デフォルト
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100 // 最大100件
			}
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := ctrl.DB.Where("is_active = ?", true)

	if industry != "" {
		query = query.Where("industry = ?", industry)
	}

	var companies []models.Company
	err := query.
		Limit(limit).
		Offset(offset).
		Order("RAND()"). // ランダムに取得
		Find(&companies).Error

	if err != nil {
		http.Error(w, "Failed to fetch companies", http.StatusInternalServerError)
		return
	}

	type CompanyResponse struct {
		Companies []models.Company `json:"companies"`
		Total     int64            `json:"total"`
		Limit     int              `json:"limit"`
		Offset    int              `json:"offset"`
	}

	var total int64
	ctrl.DB.Model(&models.Company{}).Where("is_active = ?", true).Count(&total)

	response := CompanyResponse{
		Companies: companies,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
