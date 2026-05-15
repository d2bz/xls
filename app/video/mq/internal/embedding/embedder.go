package embedding

import (
	"context"

	"xls/pkg/embedding"
)

type Embedder = embedding.Embedder

func NewEmbedder(ctx context.Context, apiKey, model, baseURL, region string, timeoutSec int) (*Embedder, error) {
	return embedding.NewEmbedder(ctx, apiKey, model, baseURL, region, timeoutSec)
}
