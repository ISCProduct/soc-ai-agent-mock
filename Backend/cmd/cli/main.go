package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"Backend/internal/openai"
)

type chatRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
}

type chatResponse struct {
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

func loadEnv(envFile string) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			log.Fatalf("failed to load env file %s: %v", envFile, err)
		}
		log.Printf("loaded env from %s", envFile)
		return
	}
	// Walk up directory tree to find .env
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("failed to get working dir: %v", err)
		return
	}
	for i := 0; i < 4 && cwd != "" && cwd != string(filepath.Separator); i++ {
		envPath := filepath.Join(cwd, ".env")
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("loaded env from %s", envPath)
				return
			}
		}
		cwd = filepath.Dir(cwd)
	}
	log.Printf("no .env file found in common locations (this may be fine if environment variables are set externally)")
}

func main() {
	envFile := flag.String("env", "", "path to .env file (optional)")
	timeoutSec := flag.Int("timeout", 30, "request timeout in seconds")
	flag.Parse()

	loadEnv(*envFile)

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatalf("OPENAI_API_KEY is not set. Set it via `export OPENAI_API_KEY=sk-...` or put it in a .env file and pass -env / place in Backend/.env.")
	}

	aiCli, err := openai.NewFromEnv("")
	if err != nil {
		log.Fatalf("failed to initialize OpenAI client: %v", err)
	}

	http.HandleFunc("/v1/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(chatResponse{Error: "invalid JSON body"})
			return
		}
		if req.Prompt == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(chatResponse{Error: "prompt is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(*timeoutSec)*time.Second)
		defer cancel()

		resp, err := aiCli.Responses(ctx, req.Prompt, req.Model)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(chatResponse{Error: fmt.Sprintf("openai request failed: %v", err)})
			return
		}

		_ = json.NewEncoder(w).Encode(chatResponse{Response: resp})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
