package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// GBizRecord is one entry from the gBizINFO API.
type GBizRecord struct {
	CorporateNumber  string `json:"corporate_number"`
	Name             string `json:"name"`
	PostalCode       string `json:"postal_code"`
	Location         string `json:"location"`
	BusinessSummary  struct {
		MajorClassificationName string `json:"major_classification_name"`
	} `json:"business_summary"`
	CompanyURL string `json:"company_url"`
}

// GBizClient calls the gBizINFO API and normalises company names.
type GBizClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
	Limiter *rate.Limiter

	mu    sync.Mutex
	cache map[string][]GBizRecord
}

func NewGBizClient(baseURL, token string) *GBizClient {
	if baseURL == "" {
		baseURL = "https://info.gbiz.go.jp/hojin/v1/hojin"
	}
	return &GBizClient{
		BaseURL: baseURL,
		Token:   token,
		Client:  &http.Client{Timeout: 15 * time.Second},
		Limiter: NewLimiter(1 * time.Second),
		cache:   map[string][]GBizRecord{},
	}
}

// Search queries gBizINFO for a company name (with optional postal code).
func (c *GBizClient) Search(ctx context.Context, name, postalCode string) ([]GBizRecord, error) {
	cacheKey := name + "|" + postalCode
	c.mu.Lock()
	if cached, ok := c.cache[cacheKey]; ok {
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{"name": {name}, "limit": {"10"}}
	if postalCode != "" {
		params.Set("postal_code", postalCode)
	}
	reqURL := c.BaseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("X-hojinInfo-api-token", c.Token)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gbizinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("gbizinfo: 401 Unauthorized (check GBIZINFO_API_TOKEN)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gbizinfo: HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Records []GBizRecord `json:"hojin-infos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("gbizinfo decode: %w", err)
	}

	c.mu.Lock()
	c.cache[cacheKey] = payload.Records
	c.mu.Unlock()

	return payload.Records, nil
}

// Match finds the best gBizINFO record for a RawCompany.
// Returns (record, score, error). When no record is found or token is missing,
// a fallback CompanyNode with corporate_number = "UNKNOWN_{name}" is returned.
func (c *GBizClient) Match(ctx context.Context, raw *RawCompany, threshold float64) (*CompanyNode, error) {
	normalized := NormalizeName(raw.RawName)

	// Try with postal code first, then without
	var candidates []GBizRecord
	var err error
	if raw.PostalCode != "" {
		candidates, err = c.Search(ctx, normalized, raw.PostalCode)
	}
	if err != nil || len(candidates) == 0 {
		candidates, err = c.Search(ctx, normalized, "")
	}

	if err != nil || len(candidates) == 0 {
		return fallbackNode(raw), nil
	}

	// Score each candidate
	best, bestScore := candidates[0], 0.0
	for _, rec := range candidates {
		s := Similarity(normalized, NormalizeName(rec.Name))
		if s > bestScore {
			bestScore = s
			best = rec
		}
	}

	needsReview := bestScore < threshold
	return &CompanyNode{
		CorporateNumber:      best.CorporateNumber,
		OfficialName:         best.Name,
		SourceURLs:           []string{raw.SourceURL},
		BusinessCategory:     best.BusinessSummary.MajorClassificationName,
		Address:              best.Location,
		Website:              best.CompanyURL,
		Capital:              raw.Capital,
		Employees:            raw.Employees,
		MatchScore:           bestScore,
		NeedsReview:          needsReview,
		RelatedCompaniesText: raw.RelatedCompaniesText,
		BusinessPartnersText: raw.BusinessPartnersText,
	}, nil
}

func fallbackNode(raw *RawCompany) *CompanyNode {
	return &CompanyNode{
		CorporateNumber:      "UNKNOWN_" + NormalizeName(raw.RawName),
		OfficialName:         raw.RawName,
		SourceURLs:           []string{raw.SourceURL},
		Address:              raw.Address,
		Website:              raw.Website,
		Capital:              raw.Capital,
		Employees:            raw.Employees,
		MatchScore:           0,
		NeedsReview:          true,
		RelatedCompaniesText: raw.RelatedCompaniesText,
		BusinessPartnersText: raw.BusinessPartnersText,
	}
}
