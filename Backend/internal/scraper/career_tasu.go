package scraper

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

var (
	corpPathRe  = regexp.MustCompile(`/corp/\d+/`)
	capitalRe   = regexp.MustCompile(`資本金[：:]\s*([^\n]{1,30})`)
	employeesRe = regexp.MustCompile(`従業員数[：:]\s*([^\n]{1,30})`)
	prefRe      = regexp.MustCompile(
		`(北海道|青森|岩手|宮城|秋田|山形|福島|茨城|栃木|群馬|埼玉|千葉|東京|神奈川` +
			`|新潟|富山|石川|福井|山梨|長野|岐阜|静岡|愛知|三重|滋賀|京都|大阪|兵庫|奈良` +
			`|和歌山|鳥取|島根|岡山|広島|山口|徳島|香川|愛媛|高知|福岡|佐賀|長崎|熊本` +
			`|大分|宮崎|鹿児島|沖縄)[都道府県]?`)
	postalRe = regexp.MustCompile(`〒?\s*(\d{3}[-ー]\d{4})`)
)

// CareerTasuScraper scrapes キャリタス就活 using plain HTTP (SSR pages).
// Search → /condition-search/result/?keyword={q}&p={p}  → /corp/{id}/default/
// ParseDetail → /corp/{id}/default/ → title / capital / employees / address
type CareerTasuScraper struct {
	BaseURL   string
	SearchURL string
	UserAgent string
	Limiter   *rate.Limiter
}

func NewCareerTasuScraper() *CareerTasuScraper {
	return &CareerTasuScraper{
		BaseURL:   "https://job.career-tasu.jp",
		SearchURL: "https://job.career-tasu.jp/condition-search/result/",
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Limiter:   NewLimiter(3 * time.Second),
	}
}

// Search returns company detail page URLs for the given keyword.
func (s *CareerTasuScraper) Search(keyword string, maxPages int) ([]string, error) {
	seen := map[string]bool{}
	var urls []string

	for page := 1; page <= maxPages; page++ {
		pageURL := fmt.Sprintf("%s?keyword=%s&p=%d", s.SearchURL, keyword, page)
		body, err := FetchHTML(pageURL, s.UserAgent, s.Limiter)
		if err != nil {
			return urls, fmt.Errorf("career_tasu search page %d: %w", page, err)
		}

		newFound := 0
		for _, match := range corpPathRe.FindAllString(body, -1) {
			full := s.BaseURL + match + "default/"
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

// ParseDetail fetches a company detail page and extracts company info.
func (s *CareerTasuScraper) ParseDetail(url string) (*RawCompany, error) {
	body, err := FetchHTML(url, s.UserAgent, s.Limiter)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Company name from <title>: "株式会社XX | キャリタス就活" or "XX社のYYY | キャリタス就活"
	companyName := ""
	titleText := strings.TrimSpace(doc.Find("title").Text())
	if titleText != "" {
		// Strip suffix
		parts := strings.SplitN(titleText, "|", 2)
		companyName = strings.TrimSpace(parts[0])
		// Strip the "の" possessive suffix if present (e.g. "株式会社XXのYY部門")
		if idx := strings.Index(companyName, "の"); idx > 0 {
			companyName = companyName[:idx]
		}
	}
	if companyName == "" {
		companyName = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if companyName == "" {
		return nil, nil
	}

	// Extract fields from full page text
	text := doc.Text()

	capital := ""
	if m := capitalRe.FindStringSubmatch(text); len(m) > 1 {
		capital = strings.TrimSpace(m[1])
	}
	employees := ""
	if m := employeesRe.FindStringSubmatch(text); len(m) > 1 {
		employees = strings.TrimSpace(m[1])
	}
	address := ""
	if m := prefRe.FindString(text); m != "" {
		address = m
	}
	postalCode := ""
	if m := postalRe.FindStringSubmatch(text); len(m) > 1 {
		postalCode = strings.ReplaceAll(m[1], "ー", "-")
	}

	return &RawCompany{
		SourceSite: "career_tasu",
		SourceURL:  url,
		RawName:    companyName,
		Address:    address,
		PostalCode: postalCode,
		Capital:    capital,
		Employees:  employees,
	}, nil
}
