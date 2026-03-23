package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/time/rate"
)

// MynaviScraper scrapes マイナビ using chromedp (headless Chrome).
// Mynavi renders company search results with JavaScript, so a real browser is required.
//
// RemoteURL, if set, is a Chrome DevTools WebSocket URL (e.g. "ws://chromedp:9222") and
// causes the scraper to connect to an external Chrome instance.
// When RemoteURL is empty, a local Chromium binary is launched via chromedp.NewExecAllocator.
//
// !! 利用規約リスク !!
// マイナビの利用規約はスクレイピング・自動収集を禁止している可能性があります。
// 本スクレイパーの運用前に必ず最新の利用規約（https://job.mynavi.jp/）を確認し、
// 法務部門の承認を取得してください。
// 違反した場合、サービス停止・法的措置のリスクがあります。
// 代替手段として公式API・RSS・プレスリリース情報源への切り替えを検討してください。
// 現在のセレクタバージョン: MynaviSelectorVersion（scraper.go 参照）
type MynaviScraper struct {
	RemoteURL string
	UserAgent string
	Limiter   *rate.Limiter
}

func NewMynaviScraper(remoteURL string) *MynaviScraper {
	return &MynaviScraper{
		RemoteURL: remoteURL,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Limiter:   NewLimiter(3 * time.Second),
	}
}

// newAllocator returns an allocator context that targets either a remote Chrome
// (when RemoteURL is set) or a locally installed Chromium binary.
func (s *MynaviScraper) newAllocator(parent context.Context) (context.Context, context.CancelFunc) {
	if s.RemoteURL != "" {
		return chromedp.NewRemoteAllocator(parent, s.RemoteURL)
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.UserAgent(s.UserAgent),
	)
	return chromedp.NewExecAllocator(parent, opts...)
}

// Search navigates to the マイナビ company search and collects detail page URLs.
// year2d is the 2-digit graduation year (e.g. 27 for 2027).
func (s *MynaviScraper) Search(keyword string, year, maxPages int) ([]string, error) {
	year2d := year % 100
	allocCtx, allocCancel := s.newAllocator(context.Background())
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, timeoutCancel := context.WithTimeout(ctx, 120*time.Second)
	defer timeoutCancel()

	var urls []string
	seen := map[string]bool{}

	for page := 1; page <= maxPages; page++ {
		searchURL := fmt.Sprintf(
			"https://job.mynavi.jp/%02d/pc/corpinfo/displayCorpSearch/index?tab=corp&keyword=%s&pageNum=%d",
			year2d, url.QueryEscape(keyword), page,
		)

		if err := s.Limiter.Wait(ctx); err != nil {
			return urls, err
		}

		// Navigate and wait for company list
		var hrefs []string
		err := chromedp.Run(ctx,
			chromedp.Navigate(searchURL),
			chromedp.WaitReady(`body`, chromedp.ByQuery),
			// Wait up to 10s for the company list; if not found, break
			chromedp.ActionFunc(func(ctx context.Context) error {
				tCtx, c := context.WithTimeout(ctx, 10*time.Second)
				defer c()
				return chromedp.WaitVisible(
					`ul.companyList, .companyCassette, .corpSearchList`,
					chromedp.ByQuery,
				).Do(tCtx)
			}),
			// Extract all company detail links
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('ul.companyList a, .companyCassette a, .corpSearchList a'))
					.map(a => a.href)
					.filter(h => h.includes('/corpinfo/displayCorpInfo/'))
			`, &hrefs),
		)
		if err != nil {
			// Company list didn't appear (login required or no results); stop pagination
			break
		}

		newFound := 0
		for _, h := range hrefs {
			if !seen[h] {
				seen[h] = true
				urls = append(urls, h)
				newFound++
			}
		}
		if newFound == 0 {
			break
		}
	}
	return urls, nil
}

// ParseDetail fetches a マイナビ company detail page via chromedp and extracts fields.
func (s *MynaviScraper) ParseDetail(detailURL string) (*RawCompany, error) {
	allocCtx, allocCancel := s.newAllocator(context.Background())
	defer allocCancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, timeoutCancel := context.WithTimeout(ctx, 60*time.Second)
	defer timeoutCancel()

	if err := s.Limiter.Wait(ctx); err != nil {
		return nil, err
	}

	var companyName, address, website, capital, employees string
	var relatedCompanies, businessPartners, businessDescription string

	// JS helper: find table cell where the preceding <th> text matches a label
	thTd := func(label string) string {
		return fmt.Sprintf(`
			(function(){
				var ths = document.querySelectorAll('.basicInfo th');
				for(var i=0; i<ths.length; i++){
					if(ths[i].textContent.trim().includes(%q)){
						var td = ths[i].nextElementSibling;
						return td ? td.innerText.trim() : '';
					}
				}
				return '';
			})()
		`, label)
	}

	err := chromedp.Run(ctx,
		chromedp.Navigate(detailURL),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		// Company name
		chromedp.Evaluate(`
			(document.querySelector('h1.companyName') ||
			 document.querySelector('.corpNameArea h1') ||
			 document.querySelector('h1'))?.innerText?.trim() || ''
		`, &companyName),
		chromedp.Evaluate(thTd("本社所在地"), &address),
		chromedp.Evaluate(thTd("URL"), &website),
		chromedp.Evaluate(thTd("資本金"), &capital),
		chromedp.Evaluate(thTd("従業員数"), &employees),
		chromedp.Evaluate(thTd("関連会社"), &relatedCompanies),
		chromedp.Evaluate(thTd("主要取引先"), &businessPartners),
		chromedp.Evaluate(thTd("事業内容"), &businessDescription),
	)
	if err != nil {
		return nil, fmt.Errorf("mynavi detail %s: %w", detailURL, err)
	}

	if strings.TrimSpace(companyName) == "" {
		return nil, nil
	}

	// Extract postal code from address
	postal := ""
	if m := postalRe.FindStringSubmatch(address); len(m) > 1 {
		postal = strings.ReplaceAll(m[1], "ー", "-")
	}

	return &RawCompany{
		SourceSite:           "mynavi",
		SourceURL:            detailURL,
		RawName:              strings.TrimSpace(companyName),
		Address:              strings.TrimSpace(address),
		PostalCode:           postal,
		Website:              strings.TrimSpace(website),
		Capital:              strings.TrimSpace(capital),
		Employees:            strings.TrimSpace(employees),
		RelatedCompaniesText: strings.TrimSpace(relatedCompanies),
		BusinessPartnersText: strings.TrimSpace(businessPartners),
		BusinessDescription:  strings.TrimSpace(businessDescription),
	}, nil
}
