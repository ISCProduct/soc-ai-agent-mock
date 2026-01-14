package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
)

type chatRequest struct {
	UserID        uint   `json:"user_id"`
	SessionID     string `json:"session_id"`
	Message       string `json:"message"`
	IndustryID    uint   `json:"industry_id"`
	JobCategoryID uint   `json:"job_category_id"`
}

type phaseProgress struct {
	PhaseName string `json:"phase_name"`
}

type chatResponse struct {
	Response      string         `json:"response"`
	CurrentPhase  *phaseProgress `json:"current_phase"`
	IsComplete    bool           `json:"is_complete"`
	AnsweredCount int            `json:"answered_questions"`
	TotalCount    int            `json:"total_questions"`
}

func main() {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	userID := uint(1001)
	sessionID := fmt.Sprintf("session_choice_check_%d", time.Now().Unix())

	fmt.Println("Base URL:", baseURL)
	fmt.Println("Session:", sessionID)

	resp, err := post(baseURL, chatRequest{
		UserID:     userID,
		SessionID:  sessionID,
		Message:    "START_SESSION",
		IndustryID: 1,
	})
	if err != nil {
		fmt.Printf("START_SESSION failed: %v\n", err)
		os.Exit(1)
	}
	printStep(1, "START_SESSION", resp)

	resp, err = post(baseURL, chatRequest{
		UserID:     userID,
		SessionID:  sessionID,
		Message:    "Webエンジニア",
		IndustryID: 1,
	})
	if err != nil {
		fmt.Printf("Job selection failed: %v\n", err)
		os.Exit(1)
	}
	printStep(2, "Webエンジニア", resp)

	seen := map[string]bool{}
	for i := 3; i <= 18; i++ {
		answer := "授業やアルバイトで経験があります"
		if isChoiceQuestion(resp.Response) {
			answer = "1"
		}
		resp, err = post(baseURL, chatRequest{
			UserID:     userID,
			SessionID:  sessionID,
			Message:    answer,
			IndustryID: 1,
		})
		if err != nil {
			fmt.Printf("Step %d failed: %v\n", i, err)
			os.Exit(1)
		}
		printStep(i, answer, resp)
		if resp.CurrentPhase != nil && resp.CurrentPhase.PhaseName != "" {
			seen[resp.CurrentPhase.PhaseName] = true
		}
		if seen["job_analysis"] && seen["interest_analysis"] && seen["aptitude_analysis"] && seen["future_analysis"] {
			break
		}
		if resp.IsComplete {
			break
		}
	}
}

func post(baseURL string, req chatRequest) (chatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return chatResponse{}, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return chatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return chatResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return chatResponse{}, fmt.Errorf("status %d", resp.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return chatResponse{}, err
	}
	return parsed, nil
}

func printStep(step int, answer string, resp chatResponse) {
	phase := ""
	if resp.CurrentPhase != nil {
		phase = resp.CurrentPhase.PhaseName
	}
	fmt.Printf("\nStep %d\n", step)
	fmt.Printf("Answer: %s\n", answer)
	fmt.Printf("Phase: %s\n", phase)
	fmt.Printf("Choice: %v\n", isChoiceQuestion(resp.Response))
	fmt.Printf("Response: %s\n", trim(resp.Response, 320))
}

func trim(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func isChoiceQuestion(text string) bool {
	if text == "" {
		return false
	}
	return choicePattern().MatchString(text)
}

var choiceRE *regexp.Regexp

func choicePattern() *regexp.Regexp {
	if choiceRE != nil {
		return choiceRE
	}
	parts := []string{
		`A\)`, `B\)`, `C\)`, `D\)`, `E\)`,
		`A：`, `B：`, `C：`, `D：`, `E：`,
		`A、`, `B、`, `C、`, `D、`, `E、`,
		`1\)`, `2\)`, `3\)`, `4\)`, `5\)`,
		`①`, `②`, `③`, `④`, `⑤`,
		`1〜5`, `1～5`, `1-5`,
		`1\.`, `2\.`, `3\.`, `4\.`, `5\.`,
		`1．`, `2．`, `3．`, `4．`, `5．`,
	}
	choiceRE = regexp.MustCompile("(" + join(parts, "|") + ")")
	return choiceRE
}

func join(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += sep + items[i]
	}
	return out
}
