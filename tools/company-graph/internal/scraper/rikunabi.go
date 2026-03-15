package scraper

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

var jobDescRe = regexp.MustCompile(`/n/job_descriptions/([a-z0-9]+)/`)

// RikunabiScraper scrapes リクナビ using plain HTTP (SSR pages).
// Search → /n/job_search/?keyword={q}&page={p}
//   → extracts /n/job_descriptions/{id}/ links
//
// ParseDetail → /n/job_descriptions/{id}/
//   → h2: "{会社名}｜{業種}"
type RikunabiScraper struct {
	BaseURL   string
	SearchURL string
	UserAgent string
	Limiter   *rate.Limiter
}

func NewRikunabiScraper() *RikunabiScraper {
	return &RikunabiScraper{
		BaseURL:   "https://job.rikunabi.com",
		SearchURL: "https://job.rikunabi.com/n/job_search/",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Limiter:   NewLimiter(3 * time.Second),
	}
}

// Search returns job-description detail page URLs for the given keyword.
func (s *RikunabiScraper) Search(keyword string, maxPages int) ([]string, error) {
	seen := map[string]bool{}
	var urls []string

	for page := 1; page <= maxPages; page++ {
		pageURL := fmt.Sprintf("%s?keyword=%s&page=%d", s.SearchURL, keyword, page)
		body, err := FetchHTML(pageURL, s.UserAgent, s.Limiter)
		if err != nil {
			return urls, fmt.Errorf("rikunabi search page %d: %w", page, err)
		}

		newFound := 0
		for _, match := range jobDescRe.FindAllString(body, -1) {
			full := "https://job.rikunabi.com" + match
			if !strings.HasSuffix(full, "/") {
				full += "/"
			}
			if !seen[full] {
				seen[full] = true
				urls = append(urls, full)
				newFound++
			}
		}
		if newFound == 0 {
			break
		}
	}
	return urls, nil
}

// ParseDetail fetches a job-description page and extracts company info.
func (s *RikunabiScraper) ParseDetail(url string) (*RawCompany, error) {
	body, err := FetchHTML(url, s.UserAgent, s.Limiter)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// h2: "{会社名}｜{業種}"
	h2Text := doc.Find("h2").First().Text()
	companyName := strings.SplitN(h2Text, "｜", 2)[0]
	companyName = strings.TrimSpace(companyName)
	if companyName == "" {
		return nil, nil
	}

	description := strings.TrimSpace(doc.Find("h1").First().Text())

	return &RawCompany{
		SourceSite:          "rikunabi",
		SourceURL:           url,
		RawName:             companyName,
		BusinessDescription: description,
	}, nil
}
