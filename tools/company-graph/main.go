package main

import (
	"company-graph/internal/scraper"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9100"
	}

	gbizBaseURL := os.Getenv("GBIZINFO_BASE_URL")
	gbizToken := os.Getenv("GBIZINFO_API_KEY")

	pipeline := &scraper.Pipeline{
		Mynavi:     scraper.NewMynaviScraper(""),
		Rikunabi:   scraper.NewRikunabiScraper(),
		CareerTasu: scraper.NewCareerTasuScraper(),
		GBiz:       scraper.NewGBizClient(gbizBaseURL, gbizToken),
		Threshold:  0.75,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/target-year", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		override := 0
		if v := r.URL.Query().Get("year"); v != "" {
			override, _ = strconv.Atoi(v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"target_year": scraper.ResolveYear(override)})
	})

	mux.HandleFunc("/crawl", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Sites     []string `json:"sites"`
			Query     string   `json:"query"`
			Pages     int      `json:"pages"`
			Year      int      `json:"year"`
			Threshold float64  `json:"threshold"`
		}
		req.Sites = []string{"rikunabi", "career_tasu"}
		req.Query = "IT"
		req.Pages = 2
		req.Threshold = 0.75

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		p := *pipeline
		if req.Threshold > 0 {
			p.Threshold = req.Threshold
		}

		result, err := p.Run(r.Context(), scraper.RunRequest{
			Sites:    req.Sites,
			Query:    req.Query,
			MaxPages: req.Pages,
			Year:     req.Year,
		})

		w.Header().Set("Content-Type", "application/json")

		logs := ""
		if result != nil {
			logs = strings.Join(result.Logs, "\n")
		}

		if err != nil && (result == nil || len(result.Nodes) == 0) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]any{
				"ok":       false,
				"error":    err.Error(),
				"logs":     logs,
				"warnings": result.Warnings,
			})
			return
		}

		targetYear := 0
		if result != nil {
			targetYear = result.TargetYear
		}

		json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"logs":        logs,
			"nodes":       result.Nodes,
			"target_year": targetYear,
			"warnings":    result.Warnings,
		})
	})

	log.Printf("company-graph server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
