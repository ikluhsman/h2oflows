package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const EmbeddingModel = "voyage-3"
const EmbeddingDims = 1024

// Embedder calls the Voyage AI embeddings API.
type Embedder struct {
	apiKey string
	client *http.Client
}

// NewEmbedder returns an Embedder backed by the Voyage AI API.
func NewEmbedder(apiKey string) *Embedder {
	return &Embedder{
		apiKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Embed sends texts to Voyage AI and returns one embedding per input,
// in the same order as the input slice. Each embedding has EmbeddingDims dimensions.
// Retries up to 5 times with exponential backoff on 429 rate-limit responses.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(embeddingRequest{
		Model: EmbeddingModel,
		Input: texts,
	})
	if err != nil {
		return nil, err
	}

	const maxRetries = 5
	wait := 22 * time.Second // 3 RPM free tier = 20s minimum between requests

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("    rate limited — retrying in %s…\n", wait)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			wait *= 2
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.voyageai.com/v1/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+e.apiKey)

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("voyage: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			continue
		}

		var out embeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("voyage: decode: %w", err)
		}
		resp.Body.Close()

		if out.Error != nil {
			return nil, fmt.Errorf("voyage: %s (%s)", out.Error.Message, out.Error.Type)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("voyage: status %d", resp.StatusCode)
		}

		embeddings := make([][]float32, len(texts))
		for _, d := range out.Data {
			if d.Index < len(embeddings) {
				embeddings[d.Index] = d.Embedding
			}
		}
		return embeddings, nil
	}

	return nil, fmt.Errorf("voyage: exceeded %d retries on rate limit", maxRetries)
}

// FormatVector formats a float32 slice as a pgvector literal: [f1,f2,...].
// Uses %f (fixed-point) to avoid scientific notation that pgvector cannot parse.
func FormatVector(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%f", f))
	}
	sb.WriteByte(']')
	return sb.String()
}
