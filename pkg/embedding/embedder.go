package embedding

import (
	"context"
	"fmt"
	"time"

	arkemb "github.com/cloudwego/eino-ext/components/embedding/ark"
)

type Embedder = arkemb.Embedder

func NewEmbedder(ctx context.Context, apiKey, model, baseURL, region string, timeoutSec int) (*Embedder, error) {
	timeout := time.Duration(timeoutSec) * time.Second
	retryTimes := 2
	embedder, err := arkemb.NewEmbedder(ctx, &arkemb.EmbeddingConfig{
		APIKey:     apiKey,
		Model:      model,
		BaseURL:    baseURL,
		Region:     region,
		Timeout:    &timeout,
		RetryTimes: &retryTimes,
	})
	if err != nil {
		return nil, fmt.Errorf("create ark embedder failed: %w", err)
	}
	return embedder, nil
}
