package controllers

import (
	"Backend/internal/services"
	"encoding/json"
	"net/http"
	"time"
)

type AdminCostsController struct {
	costService *services.APICostService
}

func NewAdminCostsController(costService *services.APICostService) *AdminCostsController {
	return &AdminCostsController{costService: costService}
}

// Summary handles GET /api/admin/costs/summary
func (c *AdminCostsController) Summary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	monthTotal, err := c.costService.GetCurrentMonthTotal()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	since30d := time.Now().UTC().AddDate(0, 0, -30)
	modelBreakdown, err := c.costService.GetModelBreakdown(since30d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_month_cost_usd": monthTotal,
		"model_breakdown":        modelBreakdown,
	})
}

// Daily handles GET /api/admin/costs/daily?days=30
func (c *AdminCostsController) Daily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	days := parseIntQuery(r, "days", 30)
	if days > 90 {
		days = 90
	}
	rows, err := c.costService.GetDailyCosts(days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"daily": rows})
}

// Monthly handles GET /api/admin/costs/monthly?months=12
func (c *AdminCostsController) Monthly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	months := parseIntQuery(r, "months", 12)
	if months > 24 {
		months = 24
	}
	rows, err := c.costService.GetMonthlyCosts(months)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"monthly": rows})
}
