package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"Backend/internal/openai"
)

func main() {
	envFile := flag.String("env", "", "path to .env file (optional)")
	model := flag.String("model", "", "OpenAI model (optional)")
	timeout := flag.Int("timeout", 30, "request timeout in seconds")
	flag.Parse()

	// .env を読み込む: -env が指定されていれば必須で読み込む、未指定なら一般的なパスを順に試す
	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			log.Fatalf("failed to load env file %s: %v", *envFile, err)
		}
		log.Printf("loaded env from %s", *envFile)
	} else {
		paths := []string{".env", "Backend/.env", "backend/.env"}
		loaded := false
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				if err := godotenv.Load(p); err == nil {
					log.Printf("loaded env from %s", p)
					loaded = true
					break
				}
			}
		}
		if !loaded {
			log.Printf("no .env file found in common locations (this may be fine if environment variables are set externally)")
		}
	}

	// 必須環境変数の事前チェックで丁寧なエラーメッセージ
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatalf("OPENAI_API_KEY is not set. Set it via `export OPENAI_API_KEY=sk-...` or put it in a .env file and pass -env / place in Backend/.env.")
	}

	// プロンプト: 引数優先、なければ stdin
	prompt := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if prompt == "" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("failed to read stdin: %v", err)
		}
		prompt = strings.TrimSpace(string(b))
	}
	if prompt == "" {
		log.Fatalf("prompt is required (pass as args or via stdin)")
	}

	aiCli, err := openai.NewFromEnv("")
	if err != nil {
		log.Fatalf("failed to initialize OpenAI client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	resp, err := aiCli.Responses(ctx, prompt, *model)
	if err != nil {
		log.Fatalf("OpenAI request failed: %v", err)
	}

	fmt.Println(resp)
}
