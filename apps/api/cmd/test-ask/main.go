package main

import (
	"context"
	"fmt"
	"os"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
)

func main() {
	ctx := context.Background()
	pool, err := db.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil { fmt.Println("db:", err); return }
	defer pool.Close()

	embedder := ai.NewEmbedder(os.Getenv("VOYAGE_API_KEY"))
	vecs, err := embedder.Embed(ctx, []string{"what flows are best for rafting"})
	if err != nil { fmt.Println("embed error:", err); return }
	if len(vecs) == 0 || vecs[0] == nil { fmt.Println("nil vec"); return }

	fv := ai.FormatVector(vecs[0])
	fmt.Printf("vec dims=%d fmt_prefix=%s\n", len(vecs[0]), fv[:40])

	tx, _ := pool.Begin(ctx)
	tx.Exec(ctx, "SET LOCAL enable_indexscan = off")
	sql := fmt.Sprintf(`
		SELECT chunk_type, LEFT(content, 60)
		FROM reach_embeddings
		WHERE reach_id = '01e7a4db-688c-4d6a-81bb-a7b87d3af28b'
		ORDER BY embedding <=> '%s'::vector
		LIMIT 3
	`, fv)
	rows, err := tx.Query(ctx, sql)
	if err != nil { fmt.Println("query error:", err); return }
	defer rows.Close()
	n := 0
	for rows.Next() {
		var ct, c string
		rows.Scan(&ct, &c)
		fmt.Printf("  [%s] %s\n", ct, c)
		n++
	}
	fmt.Printf("total rows: %d, err: %v\n", n, rows.Err())
}
