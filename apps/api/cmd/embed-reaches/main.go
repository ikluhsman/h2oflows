// embed-reaches generates vector embeddings for all reach content and stores
// them in the reach_embeddings table for RAG-powered river assistant queries.
//
// Already-embedded chunks are skipped (idempotent via ON CONFLICT DO NOTHING).
// Use REEMBED=true to wipe and re-embed everything.
//
//	go run ./cmd/embed-reaches
//	REEMBED=true go run ./cmd/embed-reaches
//
// Env vars: DATABASE_URL, VOYAGE_API_KEY
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
)

func main() {
	ctx := context.Background()

	dbURL   := mustEnv("DATABASE_URL")
	apiKey  := mustEnv("VOYAGE_API_KEY")
	reembed := os.Getenv("REEMBED") == "true"

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	embedder := ai.NewEmbedder(apiKey)

	if reembed {
		fmt.Println("REEMBED=true — deleting existing embeddings")
	}

	embedded, skipped, err := ai.EmbedReachesAll(ctx, pool, embedder, reembed)
	if err != nil {
		log.Fatalf("embed: %v", err)
	}

	fmt.Printf("done — %d embedded, %d skipped\n", embedded, skipped)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("env var %s is required", key)
	}
	return v
}
