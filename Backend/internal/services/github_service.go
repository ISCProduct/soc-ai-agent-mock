package services

import (
	"Backend/internal/models"
	"Backend/internal/repositories"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	githubAPIBase    = "https://api.github.com"
	githubGraphQLURL = "https://api.github.com/graphql"
	// レート制限: 最終同期から1時間以内は再同期しない
	syncCacheDuration = time.Hour
	// レート制限リトライ: 最大2回
	maxRetries = 2
)

// GitHubService GitHub API連携サービス
type GitHubService struct {
	githubRepo *repositories.GitHubRepository
}

func NewGitHubService(githubRepo *repositories.GitHubRepository) *GitHubService {
	return &GitHubService{githubRepo: githubRepo}
}

// StoreAccessToken GitHubアクセストークンとプロフィール基本情報を保存する
func (s *GitHubService) StoreAccessToken(userID uint, login, accessToken string) error {
	profile := &models.GitHubProfile{
		UserID:      userID,
		GitHubLogin: login,
		AccessToken: accessToken,
	}
	return s.githubRepo.UpsertProfile(profile)
}

// TriggerAsyncSync 非同期でGitHubデータ同期を開始する（ノンブロッキング）
func (s *GitHubService) TriggerAsyncSync(userID uint) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.SyncUserData(ctx, userID); err != nil {
			log.Printf("[GitHubService] async sync failed for user %d: %v", userID, err)
		}
	}()
}

// SyncUserData GitHubからリポジトリ・言語比率・コントリビューション数を取得してDBに保存する
func (s *GitHubService) SyncUserData(ctx context.Context, userID uint) error {
	profile, err := s.githubRepo.GetProfile(userID)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}
	if profile == nil {
		return fmt.Errorf("github profile not found for user %d", userID)
	}

	// キャッシュチェック: 1時間以内に同期済みならスキップ
	if profile.SyncedAt != nil && time.Since(*profile.SyncedAt) < syncCacheDuration {
		log.Printf("[GitHubService] user %d: skipping sync (last synced %s ago)", userID, time.Since(*profile.SyncedAt).Round(time.Minute))
		return nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	token := profile.AccessToken
	login := profile.GitHubLogin

	// 1. リポジトリ一覧取得
	repos, err := s.fetchRepositories(ctx, client, token, login)
	if err != nil {
		return fmt.Errorf("fetch repositories: %w", err)
	}

	// 2. 言語使用比率集計
	langStats := aggregateLanguages(userID, repos)

	// userIDをセット
	for i := range repos {
		repos[i].UserID = userID
	}

	// 3. コントリビューション数取得（GraphQL）
	contributions, err := s.fetchTotalContributions(ctx, client, token, login)
	if err != nil {
		log.Printf("[GitHubService] fetch contributions warning: %v", err)
		// コントリビューション取得失敗はwarn扱いで続行
	}

	// 4. プロフィール統計更新
	now := time.Now()
	profile.TotalContributions = contributions
	profile.PublicRepos = len(repos)
	profile.SyncedAt = &now
	if err := s.githubRepo.UpsertProfile(profile); err != nil {
		return fmt.Errorf("update profile: %w", err)
	}

	// 5. リポジトリ保存
	if err := s.githubRepo.ReplaceRepositories(userID, repos); err != nil {
		return fmt.Errorf("save repositories: %w", err)
	}

	// 6. 言語比率保存
	if err := s.githubRepo.ReplaceLanguageStats(userID, langStats); err != nil {
		return fmt.Errorf("save language stats: %w", err)
	}

	log.Printf("[GitHubService] user %d: sync completed (%d repos, %d contributions)", userID, len(repos), contributions)
	return nil
}

// GetProfile DBからGitHubプロフィールを取得する
func (s *GitHubService) GetProfile(userID uint) (*models.GitHubProfile, error) {
	return s.githubRepo.GetProfile(userID)
}

// GetRepositories DBからリポジトリ一覧を取得する
func (s *GitHubService) GetRepositories(userID uint) ([]models.GitHubRepo, error) {
	return s.githubRepo.GetRepositories(userID)
}

// GetLanguageStats DBから言語使用比率を取得する
func (s *GitHubService) GetLanguageStats(userID uint) ([]models.GitHubLanguageStat, error) {
	return s.githubRepo.GetLanguageStats(userID)
}

// --- 内部ヘルパー ---

// githubAPIRepository GitHub API レスポンスのリポジトリ構造体
type githubAPIRepository struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	StargazersCount int `json:"stargazers_count"`
	ForksCount  int    `json:"forks_count"`
	Fork        bool   `json:"fork"`
	UpdatedAt   string `json:"updated_at"`
}

// fetchRepositories GitHub APIからリポジトリ一覧を全ページ取得する
func (s *GitHubService) fetchRepositories(ctx context.Context, client *http.Client, token, login string) ([]models.GitHubRepo, error) {
	var allRepos []models.GitHubRepo
	page := 1

	for {
		url := fmt.Sprintf("%s/users/%s/repos?type=owner&sort=updated&per_page=100&page=%d", githubAPIBase, login, page)
		body, err := s.doRequestWithRetry(ctx, client, token, url)
		if err != nil {
			return nil, err
		}

		var apiRepos []githubAPIRepository
		if err := json.Unmarshal(body, &apiRepos); err != nil {
			return nil, fmt.Errorf("unmarshal repos page %d: %w", page, err)
		}

		if len(apiRepos) == 0 {
			break
		}

		for _, r := range apiRepos {
			updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)
			allRepos = append(allRepos, models.GitHubRepo{
				Name:            r.Name,
				FullName:        r.FullName,
				Description:     r.Description,
				Language:        r.Language,
				Stars:           r.StargazersCount,
				Forks:           r.ForksCount,
				IsForked:        r.Fork,
				GitHubUpdatedAt: updatedAt,
			})
		}

		if len(apiRepos) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}

// githubLanguagesResponse GitHub言語APIレスポンス（言語名→バイト数のmap）
type githubLanguagesResponse map[string]int64

// aggregateLanguages リポジトリ一覧から言語使用統計を集計する
func aggregateLanguages(userID uint, repos []models.GitHubRepo) []models.GitHubLanguageStat {
	langBytes := make(map[string]int64)
	var total int64

	for _, r := range repos {
		if r.Language != "" {
			// リポジトリのメイン言語のみ集計（バイト数は不明なので件数ベース）
			langBytes[r.Language]++
			total++
		}
	}

	if total == 0 {
		return nil
	}

	stats := make([]models.GitHubLanguageStat, 0, len(langBytes))
	for lang, count := range langBytes {
		stats = append(stats, models.GitHubLanguageStat{
			UserID:     userID,
			Language:   lang,
			Bytes:      count,
			Percentage: float64(count) / float64(total) * 100,
		})
	}
	return stats
}

// contributionsGraphQLResponse GraphQLレスポンス構造体
type contributionsGraphQLResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				ContributionCalendar struct {
					TotalContributions int `json:"totalContributions"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// fetchTotalContributions GraphQL APIで年間コントリビューション数を取得する
func (s *GitHubService) fetchTotalContributions(ctx context.Context, client *http.Client, token, login string) (int, error) {
	query := fmt.Sprintf(`{"query":"query{user(login:\"%s\"){contributionsCollection{contributionCalendar{totalContributions}}}}"}`, login)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubGraphQLURL, bytes.NewBufferString(query))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if err := checkRateLimit(resp); err != nil {
		return 0, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result contributionsGraphQLResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("unmarshal graphql response: %w", err)
	}
	if len(result.Errors) > 0 {
		return 0, fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}

	return result.Data.User.ContributionsCollection.ContributionCalendar.TotalContributions, nil
}

// doRequestWithRetry レート制限対応GETリクエスト（最大maxRetriesリトライ）
func (s *GitHubService) doRequestWithRetry(ctx context.Context, client *http.Client, token, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		body, err := s.doGet(ctx, client, token, url)
		if err == nil {
			return body, nil
		}
		lastErr = err

		// レート制限エラーの場合はリトライしない
		if isRateLimitError(err) {
			return nil, err
		}

		if attempt < maxRetries {
			wait := time.Duration(attempt+1) * 2 * time.Second
			log.Printf("[GitHubService] request failed (attempt %d/%d), retrying in %s: %v", attempt+1, maxRetries, wait, err)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return nil, lastErr
}

// doGet GitHub APIへGETリクエストを送る
func (s *GitHubService) doGet(ctx context.Context, client *http.Client, token, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkRateLimit(resp); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// rateLimitError レート制限エラー型
type rateLimitError struct {
	resetAt time.Time
}

func (e *rateLimitError) Error() string {
	return fmt.Sprintf("github rate limit exceeded, resets at %s", e.resetAt.Format(time.RFC3339))
}

func isRateLimitError(err error) bool {
	_, ok := err.(*rateLimitError)
	return ok
}

// checkRateLimit レスポンスヘッダーからレート制限を確認する
func checkRateLimit(resp *http.Response) error {
	if resp.StatusCode != 403 && resp.StatusCode != 429 {
		return nil
	}
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	if remaining == "0" || resp.StatusCode == 429 {
		resetUnix, _ := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
		resetAt := time.Unix(resetUnix, 0)
		return &rateLimitError{resetAt: resetAt}
	}
	return nil
}
