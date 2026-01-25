package services

import (
	"Backend/internal/models"
	"Backend/internal/openai"
	"Backend/internal/repositories"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type CrawlService struct {
	repo        *repositories.CrawlRepository
	companyRepo *repositories.CompanyRepository
	popularRepo *repositories.CompanyPopularityRepository
	aiClient    *openai.Client
	mu          sync.Mutex
}

func NewCrawlService(repo *repositories.CrawlRepository, companyRepo *repositories.CompanyRepository, popularRepo *repositories.CompanyPopularityRepository, aiClient *openai.Client) *CrawlService {
	return &CrawlService{repo: repo, companyRepo: companyRepo, popularRepo: popularRepo, aiClient: aiClient}
}

type CrawlSourcePayload struct {
	Name         string `json:"name"`
	TargetType   string `json:"target_type"`
	SourceType   string `json:"source_type"`
	SourceURL    string `json:"source_url"`
	ScheduleType string `json:"schedule_type"`
	ScheduleDay  *int   `json:"schedule_day"`
	ScheduleTime string `json:"schedule_time"`
	IsActive     *bool  `json:"is_active"`
}

func (s *CrawlService) ListSources() ([]models.CrawlSource, error) {
	return s.repo.ListSources()
}

func (s *CrawlService) ListRuns(sourceID uint, limit int) ([]models.CrawlRun, error) {
	return s.repo.ListRuns(sourceID, limit)
}

func (s *CrawlService) CreateSource(payload CrawlSourcePayload) (*models.CrawlSource, error) {
	if payload.ScheduleDay == nil {
		return nil, errors.New("schedule_day is required")
	}
	source := &models.CrawlSource{
		Name:         strings.TrimSpace(payload.Name),
		TargetType:   strings.TrimSpace(payload.TargetType),
		SourceType:   strings.TrimSpace(payload.SourceType),
		SourceURL:    strings.TrimSpace(payload.SourceURL),
		ScheduleType: strings.TrimSpace(payload.ScheduleType),
		ScheduleDay:  *payload.ScheduleDay,
		ScheduleTime: strings.TrimSpace(payload.ScheduleTime),
		IsActive:     true,
	}
	if payload.IsActive != nil {
		source.IsActive = *payload.IsActive
	}
	if err := validateCrawlSource(source); err != nil {
		return nil, err
	}
	next := computeNextRun(time.Now(), source)
	source.NextRunAt = next
	if err := s.repo.CreateSource(source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *CrawlService) UpdateSource(id uint, payload CrawlSourcePayload) (*models.CrawlSource, error) {
	source, err := s.repo.GetSource(id)
	if err != nil {
		return nil, err
	}
	if payload.Name != "" {
		source.Name = strings.TrimSpace(payload.Name)
	}
	if payload.TargetType != "" {
		source.TargetType = strings.TrimSpace(payload.TargetType)
	}
	if payload.SourceType != "" {
		source.SourceType = strings.TrimSpace(payload.SourceType)
	}
	if payload.SourceURL != "" {
		source.SourceURL = strings.TrimSpace(payload.SourceURL)
	}
	if payload.ScheduleType != "" {
		source.ScheduleType = strings.TrimSpace(payload.ScheduleType)
	}
	if payload.ScheduleDay != nil {
		source.ScheduleDay = *payload.ScheduleDay
	}
	if payload.ScheduleTime != "" {
		source.ScheduleTime = strings.TrimSpace(payload.ScheduleTime)
	}
	if payload.IsActive != nil {
		source.IsActive = *payload.IsActive
	}
	if err := validateCrawlSource(source); err != nil {
		return nil, err
	}
	source.NextRunAt = computeNextRun(time.Now(), source)
	if err := s.repo.UpdateSource(source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *CrawlService) RunSource(id uint) (*models.CrawlRun, error) {
	source, err := s.repo.GetSource(id)
	if err != nil {
		return nil, err
	}
	return s.runSource(source)
}

func (s *CrawlService) StartScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.runDueSources()
	}
}

func (s *CrawlService) RunDueSources() {
	s.runDueSources()
}

func (s *CrawlService) runDueSources() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	sources, err := s.repo.ListDueSources(now)
	if err != nil {
		return
	}
	for i := range sources {
		_, _ = s.runSource(&sources[i])
	}
}

func (s *CrawlService) runSource(source *models.CrawlSource) (*models.CrawlRun, error) {
	if source == nil {
		return nil, errors.New("source not found")
	}
	run := &models.CrawlRun{
		SourceID:  source.ID,
		Status:    "running",
		Message:   "",
		StartedAt: time.Now(),
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}

	message := ""
	if err := s.executeCrawl(source); err != nil {
		message = err.Error()
		run.Status = "failed"
	} else {
		run.Status = "success"
		message = "completed"
	}
	run.Message = message
	finished := time.Now()
	run.EndedAt = &finished
	_ = s.repo.UpdateRun(run)

	source.LastRunAt = &finished
	source.NextRunAt = computeNextRun(finished, source)
	_ = s.repo.UpdateSource(source)

	return run, nil
}

func (s *CrawlService) executeCrawl(source *models.CrawlSource) error {
	switch source.TargetType {
	case "company":
		return s.executeCompanyCrawl(source)
	case "popular_companies":
		return s.executePopularCompaniesCrawl(source)
	default:
		return fmt.Errorf("unsupported target_type: %s", source.TargetType)
	}
}

func validateCrawlSource(source *models.CrawlSource) error {
	if strings.TrimSpace(source.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(source.TargetType) == "" {
		return errors.New("target_type is required")
	}
	if source.TargetType != "company" && source.TargetType != "popular_companies" {
		return errors.New("target_type must be company or popular_companies")
	}
	if source.TargetType == "popular_companies" && strings.TrimSpace(source.SourceURL) == "" {
		return errors.New("source_url is required for popular_companies")
	}
	if source.ScheduleType != "weekly" && source.ScheduleType != "monthly" {
		return errors.New("schedule_type must be weekly or monthly")
	}
	if source.ScheduleType == "weekly" {
		if source.ScheduleDay < 0 || source.ScheduleDay > 6 {
			return errors.New("schedule_day must be 0-6 for weekly")
		}
	} else {
		if source.ScheduleDay < 1 || source.ScheduleDay > 31 {
			return errors.New("schedule_day must be 1-31 for monthly")
		}
	}
	if source.ScheduleTime == "" || !isValidTime(source.ScheduleTime) {
		return errors.New("schedule_time must be HH:MM")
	}
	return nil
}

func (s *CrawlService) executeCompanyCrawl(source *models.CrawlSource) error {
	if strings.TrimSpace(source.Name) == "" {
		return errors.New("company name is required for company crawl")
	}
	now := time.Now()
	company, err := s.companyRepo.FindByName(source.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if company == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		newCompany := &models.Company{
			Name:            source.Name,
			SourceType:      source.SourceType,
			SourceURL:       source.SourceURL,
			SourceFetchedAt: &now,
			IsProvisional:   true,
			DataStatus:      "draft",
		}
		return s.companyRepo.Create(newCompany)
	}
	company.SourceType = source.SourceType
	company.SourceURL = source.SourceURL
	company.SourceFetchedAt = &now
	return s.companyRepo.Update(company)
}

type popularCompanyExtraction struct {
	Companies []struct {
		Name     string `json:"name"`
		Evidence string `json:"evidence"`
		Summary  string `json:"summary"`
		Rank     *int   `json:"rank,omitempty"`
	} `json:"companies"`
}

func (s *CrawlService) executePopularCompaniesCrawl(source *models.CrawlSource) error {
	if s.aiClient == nil {
		return errors.New("openai client is required for popular_companies crawl")
	}
	body, err := fetchText(source.SourceURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return errors.New("empty content from source_url")
	}

	extracted, err := s.extractPopularCompanies(source, body)
	if err != nil {
		return err
	}
	if len(extracted.Companies) == 0 {
		return errors.New("no companies extracted from source")
	}

	now := time.Now()
	for _, item := range extracted.Companies {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		company, err := s.companyRepo.FindByName(name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if company == nil || errors.Is(err, gorm.ErrRecordNotFound) {
			newCompany := &models.Company{
				Name:            name,
				SourceType:      source.SourceType,
				SourceURL:       source.SourceURL,
				SourceFetchedAt: &now,
				IsProvisional:   true,
				DataStatus:      "draft",
			}
			if err := s.companyRepo.Create(newCompany); err != nil {
				return err
			}
			company = newCompany
		}

		record := &models.CompanyPopularityRecord{
			CompanyID:    company.ID,
			SourceName:   source.Name,
			SourceURL:    source.SourceURL,
			EvidenceText: strings.TrimSpace(item.Evidence),
			Summary:      strings.TrimSpace(item.Summary),
			Rank:         item.Rank,
			FetchedAt:    now,
		}
		if err := s.popularRepo.Create(record); err != nil {
			return err
		}
	}
	return nil
}

func (s *CrawlService) extractPopularCompanies(source *models.CrawlSource, rawHTML string) (*popularCompanyExtraction, error) {
	clean := normalizeHTMLText(rawHTML)
	if len(clean) > 12000 {
		clean = clean[:12000]
	}
	systemPrompt := `You are a data extraction assistant. Use only the provided text. Do not infer or guess.`
	userPrompt := fmt.Sprintf(`Extract popular companies mentioned in the text below.
Return JSON with the following shape:
{
  "companies": [
    {
      "name": "Company Name",
      "evidence": "Exact excerpt from the text",
      "summary": "Why the company is described as popular, based only on the text",
      "rank": 1
    }
  ]
}
Rules:
- If rank is not shown, omit it or set it to null.
- evidence must be a verbatim excerpt from the text.
- summary must be a short, factual sentence based on the evidence only.

Text:
%s`, clean)

	content, err := s.aiClient.ChatCompletionJSON(context.Background(), systemPrompt, userPrompt, 0.2, 800)
	if err != nil {
		return nil, err
	}
	var parsed popularCompanyExtraction
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func fetchText(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SocAI/1.0; +https://example.com/bot)")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch failed: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func normalizeHTMLText(rawHTML string) string {
	clean := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</\1>`).ReplaceAllString(rawHTML, " ")
	clean = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(clean, " ")
	clean = html.UnescapeString(clean)
	clean = strings.ReplaceAll(clean, "\u00a0", " ")
	clean = regexp.MustCompile(`\s+`).ReplaceAllString(clean, " ")
	return strings.TrimSpace(clean)
}

func isValidTime(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}

func computeNextRun(now time.Time, source *models.CrawlSource) *time.Time {
	if source == nil {
		return nil
	}
	hourMin, err := time.Parse("15:04", source.ScheduleTime)
	if err != nil {
		return nil
	}
	hour := hourMin.Hour()
	min := hourMin.Minute()
	loc := now.Location()
	var next time.Time
	if source.ScheduleType == "weekly" {
		target := time.Weekday(source.ScheduleDay)
		days := (int(target) - int(now.Weekday()) + 7) % 7
		next = time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc).AddDate(0, 0, days)
		if !next.After(now) {
			next = next.AddDate(0, 0, 7)
		}
	} else {
		day := source.ScheduleDay
		year, month := now.Year(), now.Month()
		lastDay := lastDayOfMonth(year, month, loc)
		if day > lastDay {
			day = lastDay
		}
		next = time.Date(year, month, day, hour, min, 0, 0, loc)
		if !next.After(now) {
			nextMonth := next.AddDate(0, 1, 0)
			year, month = nextMonth.Year(), nextMonth.Month()
			day = source.ScheduleDay
			lastDay = lastDayOfMonth(year, month, loc)
			if day > lastDay {
				day = lastDay
			}
			next = time.Date(year, month, day, hour, min, 0, 0, loc)
		}
	}
	return &next
}

func lastDayOfMonth(year int, month time.Month, loc *time.Location) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
}
